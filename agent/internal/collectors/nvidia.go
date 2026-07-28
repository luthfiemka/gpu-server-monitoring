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

type GpuMetrics struct {
	GpuID          string
	GpuUUID        string
	GpuName        string
	UtilizationGPU *float64
	UtilizationMem *float64
	MemoryUsed     *float64
	MemoryTotal    *float64
	Temperature    *float64
	PowerDraw      *float64
	PowerLimit     *float64
	FanSpeed       *float64
}

type GpuProcess struct {
	GpuID        string
	GpuUUID      string
	PID          int
	ProcessName  string
	UsedMemory   *float64
	MemAlloc     *float64
	SharedMemory *float64
}

func parseFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "[N/A]" || s == "N/A" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

func runNvidiaSmi(args []string) (string, error) {
	cmd := exec.Command("nvidia-smi", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("nvidia-smi %v: %w (%s)", args, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func CollectGpus() ([]GpuMetrics, error) {
	fields := []string{
		"index", "uuid", "name",
		"utilization.gpu", "utilization.memory",
		"memory.used", "memory.total",
		"temperature.gpu",
		"power.draw", "power.limit",
		"fan.speed",
	}
	raw, err := runNvidiaSmi([]string{
		"--query-gpu=" + strings.Join(fields, ","),
		"--format=csv,noheader,nounits",
	})
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(raw))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse nvidia-smi output: %w", err)
	}

	gpus := make([]GpuMetrics, 0, len(records))
	for _, row := range records {
		if len(row) < len(fields) {
			log.Printf("warn: nvidia-smi row has %d fields, want %d", len(row), len(fields))
			continue
		}
		gpus = append(gpus, GpuMetrics{
			GpuID:          strings.TrimSpace(row[0]),
			GpuUUID:        strings.TrimSpace(row[1]),
			GpuName:        strings.TrimSpace(row[2]),
			UtilizationGPU: parseFloat(row[3]),
			UtilizationMem: parseFloat(row[4]),
			MemoryUsed:     parseFloat(row[5]),
			MemoryTotal:    parseFloat(row[6]),
			Temperature:    parseFloat(row[7]),
			PowerDraw:      parseFloat(row[8]),
			PowerLimit:     parseFloat(row[9]),
			FanSpeed:       parseFloat(row[10]),
		})
	}
	return gpus, nil
}

func CollectProcesses() ([]GpuProcess, error) {
	fields := []string{"gpu_uuid", "pid", "process_name", "used_gpu_memory"}
	raw, err := runNvidiaSmi([]string{
		"--query-compute-apps=" + strings.Join(fields, ","),
		"--format=csv,noheader,nounits",
	})
	if err != nil {
		log.Printf("debug: no compute apps or query failed: %v", err)
		return nil, nil
	}

	reader := csv.NewReader(strings.NewReader(raw))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse nvidia-smi processes: %w", err)
	}

	uuidToID := map[string]string{}
	gpuRaw, err := runNvidiaSmi([]string{
		"--query-gpu=index,uuid",
		"--format=csv,noheader,nounits",
	})
	if err == nil {
		gpuReader := csv.NewReader(strings.NewReader(gpuRaw))
		gpuRecords, _ := gpuReader.ReadAll()
		for _, row := range gpuRecords {
			if len(row) >= 2 {
				uuidToID[strings.TrimSpace(row[1])] = strings.TrimSpace(row[0])
			}
		}
	}

	procs := make([]GpuProcess, 0, len(records))
	for _, row := range records {
		if len(row) < 3 {
			continue
		}
		uuid := strings.TrimSpace(row[0])
		pid, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			continue
		}
		var mem *float64
		if len(row) > 3 {
			mem = parseFloat(row[3])
		}
		procs = append(procs, GpuProcess{
			GpuID:       uuidToID[uuid],
			GpuUUID:     uuid,
			PID:         pid,
			ProcessName: strings.TrimSpace(row[2]),
			UsedMemory:  mem,
			MemAlloc:    mem,
		})
	}
	return procs, nil
}
