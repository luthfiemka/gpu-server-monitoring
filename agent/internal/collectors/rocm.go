package collectors

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func runRocmSmi(args []string) (string, error) {
	cmd := exec.Command("rocm-smi", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rocm-smi %v: %w (%s)", args, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func HasNvidiaSmi() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func HasRocmSmi() bool {
	_, err := exec.LookPath("rocm-smi")
	return err == nil
}

func CollectGpusAMD() ([]GpuMetrics, error) {
	raw, err := runRocmSmi([]string{
		"--showuse", "--showmemuse", "--showtemp",
		"--showpower", "--showfan", "--showmeminfo", "vram",
		"--csv",
	})
	if err != nil {
		return nil, err
	}

	reader := newRocmCSVReader(raw)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse rocm-smi output: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("rocm-smi: no GPU data")
	}

	header := records[0]
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	gpus := make([]GpuMetrics, 0, len(records)-1)
	for _, row := range records[1:] {
		gpuIdx, ok := colIdx["device"]
		if !ok {
			return nil, fmt.Errorf("rocm-smi: missing device column")
		}
		if gpuIdx < 0 || gpuIdx >= len(row) {
			continue
		}

		g := GpuMetrics{
			GpuID:   strings.TrimSpace(row[gpuIdx]),
			GpuName: "AMD GPU",
		}

		g.GpuUUID = g.GpuID

		if i, ok := colIdx["gpu use (%)"]; ok && i < len(row) {
			g.UtilizationGPU = parseFloat(row[i])
		}
		if i, ok := colIdx["gpu memory use (%)"]; ok && i < len(row) {
			g.UtilizationMem = parseFloat(row[i])
		}
		if i, ok := colIdx["temperature (edge) (c)"]; ok && i < len(row) {
			g.Temperature = parseFloat(row[i])
		} else if i, ok := colIdx["temperature (c)"]; ok && i < len(row) {
			g.Temperature = parseFloat(row[i])
		}
		if i, ok := colIdx["average socket power (w)"]; ok && i < len(row) {
			g.PowerDraw = parseFloat(row[i])
		} else if i, ok := colIdx["power (w)"]; ok && i < len(row) {
			g.PowerDraw = parseFloat(row[i])
		}
		if i, ok := colIdx["fan speed (%)"]; ok && i < len(row) {
			g.FanSpeed = parseFloat(row[i])
		}

		if i, ok := colIdx["vram total memory (b)"]; ok && i < len(row) {
			if v := parseFloat(row[i]); v != nil {
				mb := *v / (1024 * 1024)
				g.MemoryTotal = &mb
			}
		}
		if i, ok := colIdx["vram total used memory (b)"]; ok && i < len(row) {
			if v := parseFloat(row[i]); v != nil {
				mb := *v / (1024 * 1024)
				g.MemoryUsed = &mb
			}
		}

		gpus = append(gpus, g)
	}
	return gpus, nil
}

func CollectProcessesAMD() ([]GpuProcess, error) {
	raw, err := runRocmSmi([]string{"--showpids", "--csv"})
	if err != nil {
		log.Printf("debug: no compute processes or rocm-smi query failed: %v", err)
		return nil, nil
	}

	return parseRocmProcesses(raw)
}

func newRocmCSVReader(raw string) *csv.Reader {
	reader := csv.NewReader(strings.NewReader(raw))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	return reader
}

func parseRocmProcesses(raw string) ([]GpuProcess, error) {
	reader := newRocmCSVReader(raw)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse rocm-smi processes: %w", err)
	}

	if len(records) < 2 {
		return nil, nil
	}

	headerIdx, header, colIdx, gpuIdx, pidIdx := findRocmProcessHeader(records)
	if pidIdx < 0 {
		return nil, nil
	}

	procs := make([]GpuProcess, 0)
	for _, row := range records[headerIdx+1:] {
		if pidIdx >= len(row) {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(row[pidIdx]))
		if err != nil {
			continue
		}

		gpuID := "unknown"
		if gpuIdx >= 0 && gpuIdx < len(row) {
			gpuID = strings.TrimSpace(row[gpuIdx])
		}

		p := GpuProcess{
			GpuID:   gpuID,
			GpuUUID: gpuID,
			PID:     pid,
		}

		nameIdx, hasName := findRocmColumn(colIdx, rocmProcessNameColumns)
		if hasName && nameIdx >= 0 && nameIdx < len(row) {
			name := row[nameIdx]
			if nameIdx == len(header)-1 && len(row) > len(header) {
				name = strings.Join(row[nameIdx:], ",")
			}
			p.ProcessName = strings.TrimSpace(name)
		}

		if mem := parseRocmProcessMemory(row, colIdx); mem != nil {
			p.UsedMemory = mem
			p.MemAlloc = mem
		}

		procs = append(procs, p)
	}
	return procs, nil
}

var (
	rocmGPUColumns         = []string{"device", "gpu", "gpu id", "gpu_id", "card", "card id", "card_id"}
	rocmPIDColumns         = []string{"pid", "process id", "process_id", "process pid"}
	rocmProcessNameColumns = []string{"name", "process name", "process_name", "command", "cmd"}
)

func findRocmProcessHeader(records [][]string) (int, []string, map[string]int, int, int) {
	for i, row := range records {
		colIdx := makeRocmColumnIndex(row)
		pidIdx, hasPID := findRocmColumn(colIdx, rocmPIDColumns)
		if !hasPID {
			continue
		}
		gpuIdx, _ := findRocmColumn(colIdx, rocmGPUColumns)
		return i, row, colIdx, gpuIdx, pidIdx
	}
	return -1, nil, nil, -1, -1
}

func makeRocmColumnIndex(header []string) map[string]int {
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[normalizeRocmColumnName(h)] = i
	}
	return colIdx
}

func normalizeRocmColumnName(name string) string {
	name = strings.Trim(strings.TrimSpace(strings.ToLower(name)), `"'`)
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.Join(strings.Fields(name), " ")
	return name
}

func findRocmColumn(colIdx map[string]int, names []string) (int, bool) {
	for _, name := range names {
		if i, ok := colIdx[normalizeRocmColumnName(name)]; ok {
			return i, true
		}
	}
	return -1, false
}

func parseRocmProcessMemory(row []string, colIdx map[string]int) *float64 {
	for _, name := range []string{
		"vram used memory (b)",
		"vram memory used (b)",
		"used memory (b)",
		"vram used (b)",
		"mem usage (b)",
	} {
		if i, ok := colIdx[name]; ok && i >= 0 && i < len(row) {
			if v := parseFloat(row[i]); v != nil {
				mb := *v / (1024 * 1024)
				return &mb
			}
		}
	}
	for _, name := range []string{
		"vram used memory (mb)",
		"vram memory used (mb)",
		"used memory (mb)",
		"vram used (mb)",
		"mem usage (mb)",
	} {
		if i, ok := colIdx[name]; ok && i >= 0 && i < len(row) {
			return parseFloat(row[i])
		}
	}
	return nil
}
