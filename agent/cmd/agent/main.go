package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gpu-monitoring-agent/internal/collectors"
	"gpu-monitoring-agent/internal/config"
	"gpu-monitoring-agent/internal/writers"
)

type gpuBackend string

const (
	backendNvidia gpuBackend = "nvidia"
	backendROCm   gpuBackend = "rocm"
	backendNone   gpuBackend = "none"
)

func detectBackend() gpuBackend {
	if collectors.HasNvidiaSmi() {
		return backendNvidia
	}
	if collectors.HasRocmSmi() {
		return backendROCm
	}
	return backendNone
}

func main() {
	configPath := flag.String("config", "/etc/gpu-monitoring/gpu-monitoring-agent.conf", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if cfg.Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.Hostname = hostname
	}

	backend := detectBackend()
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("agent started: host=%s backend=%s questdb=%s:%d/ilp:%d interval=%ds",
		cfg.Hostname, backend, cfg.QuestDB.Host, cfg.QuestDB.Port, cfg.QuestDB.ILPPort, cfg.QuestDB.Interval)

	if backend == backendNone {
		log.Printf("FATAL: no GPU tool found (nvidia-smi or rocm-smi)")
		os.Exit(1)
	}

	w := writers.NewQuestDBWriter(
		cfg.QuestDB.Host,
		cfg.QuestDB.ILPPort,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(time.Duration(cfg.QuestDB.Interval) * time.Second)
	defer ticker.Stop()

	// run once immediately
	collectAndWrite(cfg, w, backend)

	for {
		select {
		case sig := <-sigCh:
			log.Printf("received %s — shutting down", sig)
			return
		case <-ticker.C:
			collectAndWrite(cfg, w, backend)
		}
	}
}

func collectAndWrite(cfg *config.Agent, w *writers.QuestDBWriter, backend gpuBackend) {
	var gpus []collectors.GpuMetrics
	var procs []collectors.GpuProcess
	var err error

	switch backend {
	case backendNvidia:
		gpus, err = collectors.CollectGpus()
		if err != nil {
			log.Printf("ERROR: collect gpus (nvidia): %v", err)
			return
		}
		procs, err = collectors.CollectProcesses()
		if err != nil {
			log.Printf("ERROR: collect processes (nvidia): %v", err)
			return
		}
	case backendROCm:
		gpus, err = collectors.CollectGpusAMD()
		if err != nil {
			log.Printf("ERROR: collect gpus (rocm): %v", err)
			return
		}
		procs, err = collectors.CollectProcessesAMD()
		if err != nil {
			log.Printf("ERROR: collect processes (rocm): %v", err)
			return
		}
	}

	if err := w.WriteBatch(cfg.Hostname, gpus, procs); err != nil {
		log.Printf("ERROR: write to questdb: %v", err)
		return
	}

	log.Printf("wrote %d gpu rows, %d process rows", len(gpus), len(procs))
}
