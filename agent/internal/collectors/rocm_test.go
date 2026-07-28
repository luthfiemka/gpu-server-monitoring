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

func TestParseRocmProcessesRequiresDeviceAndPIDColumns(t *testing.T) {
	_, err := parseRocmProcesses("device,name\ncard0,python\n")
	if err == nil {
		t.Fatal("parseRocmProcesses() error = nil, want missing column error")
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
