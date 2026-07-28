package writers

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"gpu-monitoring-agent/internal/collectors"
)

type QuestDBWriter struct {
	host  string
	port  int
	token string
}

func NewQuestDBWriter(host string, ilpPort int, token string) *QuestDBWriter {
	return &QuestDBWriter{
		host:  host,
		port:  ilpPort,
		token: token,
	}
}

func floatToStr(f *float64) string {
	if f == nil {
		return "NaN"
	}
	return fmt.Sprintf("%g", *f)
}

func escapeTagVal(s string) string {
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "=", `\=`)
	s = strings.ReplaceAll(s, " ", `\ `)
	return s
}

func (w *QuestDBWriter) WriteBatch(
	hostname string,
	gpus []collectors.GpuMetrics,
	procs []collectors.GpuProcess,
) error {
	now := time.Now().UnixNano()
	var sb strings.Builder

	for _, g := range gpus {
		sb.WriteString("gpu_metrics,")
		sb.WriteString(fmt.Sprintf("hostname=%s,", escapeTagVal(hostname)))
		sb.WriteString(fmt.Sprintf("gpu_id=%s,", escapeTagVal(g.GpuID)))
		sb.WriteString(fmt.Sprintf("gpu_uuid=%s,", escapeTagVal(g.GpuUUID)))
		sb.WriteString(fmt.Sprintf("gpu_name=%s ", escapeTagVal(g.GpuName)))
		sb.WriteString(fmt.Sprintf("utilization_gpu=%s,", floatToStr(g.UtilizationGPU)))
		sb.WriteString(fmt.Sprintf("utilization_mem=%s,", floatToStr(g.UtilizationMem)))
		sb.WriteString(fmt.Sprintf("memory_used=%s,", floatToStr(g.MemoryUsed)))
		sb.WriteString(fmt.Sprintf("memory_total=%s,", floatToStr(g.MemoryTotal)))
		sb.WriteString(fmt.Sprintf("temperature=%s,", floatToStr(g.Temperature)))
		sb.WriteString(fmt.Sprintf("power_draw=%s,", floatToStr(g.PowerDraw)))
		sb.WriteString(fmt.Sprintf("power_limit=%s,", floatToStr(g.PowerLimit)))
		sb.WriteString(fmt.Sprintf("fan_speed=%s ", floatToStr(g.FanSpeed)))
		sb.WriteString(fmt.Sprintf("%d\n", now))
	}

	for _, p := range procs {
		info := collectors.DetectContainer(p.PID)
		cid := ""
		cname := ""
		if info != nil {
			cid = info.ID
			cname = info.Name
		}

		username := getUsername(p.PID)

		sb.WriteString("gpu_processes,")
		sb.WriteString(fmt.Sprintf("hostname=%s,", escapeTagVal(hostname)))
		sb.WriteString(fmt.Sprintf("gpu_id=%s,", escapeTagVal(p.GpuID)))
		sb.WriteString(fmt.Sprintf("process_name=%s,", escapeTagVal(p.ProcessName)))
		sb.WriteString(fmt.Sprintf("username=%s,", escapeTagVal(username)))
		sb.WriteString(fmt.Sprintf("container_id=%s,", escapeTagVal(cid)))
		sb.WriteString(fmt.Sprintf("container_name=%s ", escapeTagVal(cname)))
		sb.WriteString(fmt.Sprintf("pid=%di,", p.PID))
		sb.WriteString(fmt.Sprintf("used_memory=%s ", floatToStr(p.UsedMemory)))
		sb.WriteString(fmt.Sprintf("%d\n", now))
	}

	if sb.Len() == 0 {
		return nil
	}

	return w.send(sb.String())
}

func (w *QuestDBWriter) send(body string) error {
	addr := fmt.Sprintf("%s:%d", w.host, w.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to questdb ILP %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	if w.token != "" {
		if _, err := conn.Write([]byte(w.token + "\n")); err != nil {
			return fmt.Errorf("send ILP auth token: %w", err)
		}
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			return fmt.Errorf("ILP auth failed: %s", line)
		}
	}

	if _, err := conn.Write([]byte(body)); err != nil {
		return fmt.Errorf("write to questdb ILP: %w", err)
	}

	return nil
}

func getUsername(pid int) string {
	path := fmt.Sprintf("/proc/%d/status", pid)
	f, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				var uid int
				fmt.Sscanf(fields[1], "%d", &uid)
				return lookupUID(uid)
			}
		}
	}
	return "unknown"
}

func lookupUID(uid int) string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return fmt.Sprintf("%d", uid)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) >= 3 {
			var id int
			fmt.Sscanf(parts[2], "%d", &id)
			if id == uid {
				return parts[0]
			}
		}
	}
	return fmt.Sprintf("%d", uid)
}
