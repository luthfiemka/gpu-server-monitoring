package collectors

import "testing"

func TestParseRocmProcessesAllowsBareQuotesInProcessName(t *testing.T) {
	raw := `device,pid,name
card0,1234,python -c "print('hello')"
`

	procs, err := parseRocmProcesses(raw)
	if err != nil {
		t.Fatalf("parseRocmProcesses() error = %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("len(procs) = %d, want 1", len(procs))
	}
	if procs[0].GpuID != "card0" {
		t.Fatalf("GpuID = %q, want card0", procs[0].GpuID)
	}
	if procs[0].PID != 1234 {
		t.Fatalf("PID = %d, want 1234", procs[0].PID)
	}
	if procs[0].ProcessName != `python -c "print('hello')"` {
		t.Fatalf("ProcessName = %q", procs[0].ProcessName)
	}
}

func TestParseRocmProcessesJoinsCommandWithComma(t *testing.T) {
	raw := `device,pid,name
card1,5678,python -c "print(1, 2)"
`

	procs, err := parseRocmProcesses(raw)
	if err != nil {
		t.Fatalf("parseRocmProcesses() error = %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("len(procs) = %d, want 1", len(procs))
	}
	if procs[0].ProcessName != `python -c "print(1, 2)"` {
		t.Fatalf("ProcessName = %q", procs[0].ProcessName)
	}
}

func TestParseRocmProcessesReturnsEmptyWhenPIDColumnMissing(t *testing.T) {
	procs, err := parseRocmProcesses("device,name\ncard0,python\n")
	if err != nil {
		t.Fatalf("parseRocmProcesses() error = %v", err)
	}
	if len(procs) != 0 {
		t.Fatalf("len(procs) = %d, want 0", len(procs))
	}
}

func TestParseRocmProcessesReadsMemoryBytes(t *testing.T) {
	raw := `device,pid,name,vram used memory (B)
card0,1234,python,2147483648
`

	procs, err := parseRocmProcesses(raw)
	if err != nil {
		t.Fatalf("parseRocmProcesses() error = %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("len(procs) = %d, want 1", len(procs))
	}
	if procs[0].UsedMemory == nil {
		t.Fatal("UsedMemory = nil, want value")
	}
	if *procs[0].UsedMemory != 2048 {
		t.Fatalf("UsedMemory = %g, want 2048", *procs[0].UsedMemory)
	}
	if procs[0].MemAlloc == nil {
		t.Fatal("MemAlloc = nil, want value")
	}
	if *procs[0].MemAlloc != 2048 {
		t.Fatalf("MemAlloc = %g, want 2048", *procs[0].MemAlloc)
	}
}

func TestParseRocmProcessesAcceptsAlternateHeadersAndPreamble(t *testing.T) {
	raw := `ROCm System Management Interface
GPU,Process ID,Process Name,VRAM Used (MB)
0,4321,python,512
`

	procs, err := parseRocmProcesses(raw)
	if err != nil {
		t.Fatalf("parseRocmProcesses() error = %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("len(procs) = %d, want 1", len(procs))
	}
	if procs[0].GpuID != "0" {
		t.Fatalf("GpuID = %q, want 0", procs[0].GpuID)
	}
	if procs[0].PID != 4321 {
		t.Fatalf("PID = %d, want 4321", procs[0].PID)
	}
	if procs[0].ProcessName != "python" {
		t.Fatalf("ProcessName = %q, want python", procs[0].ProcessName)
	}
	if procs[0].UsedMemory == nil || *procs[0].UsedMemory != 512 {
		t.Fatalf("UsedMemory = %v, want 512", procs[0].UsedMemory)
	}
}
