# GPU Server Monitoring

Real-time GPU monitoring dashboard. Go agent collects metrics via `nvidia-smi`, stores in QuestDB. Web dashboard built with SvelteKit + Tailwind CSS.

## Architecture

```
┌─────────────────┐     ┌────────────────┐     ┌──────────────┐
│  GPU Servers     │────▶│  Monitoring    │────▶│  QuestDB     │
│  (nvidia-smi)   │     │  Agent (Go)    │     │  (time-series)│
└─────────────────┘     └────────────────┘     └──────┬───────┘
                                                       │
                                                ┌──────▼───────┐
                                                │  Frontend    │
                                                │  (SvelteKit) │
                                                └──────────────┘
```

## Features

- **GPU Metrics**: utilization, memory, temperature, power, fan speed
- **Process Tracking**: per-process GPU memory usage with user/container info
- **Container Detection**: auto-detects Docker/containerd containers via `/proc/<pid>/cgroup`
- **Historical Data**: time-range queries with configurable sample-by intervals
- **Multi-GPU**: supports multiple GPUs per server, multi-server via hostname tagging
- **Dark Mode**: built-in dark/light theme toggle

---

## Quick Start (Docker Compose)

Run all services with one command:

```bash
git clone <repo-url> gpu-server-monitoring
cd gpu-server-monitoring
docker compose up -d
```

**Services:**
| Service    | Port | Description             |
|------------|------|-------------------------|
| Frontend   | 5173 | Web dashboard           |
| QuestDB    | 9000 | Web console / REST API  |
| QuestDB    | 9001 | PostgreSQL wire protocol|
| QuestDB    | 9009 | ILP TCP (not used)      |

**Login:** `admin` / `admin` (change via `ADMIN_USER` / `ADMIN_PASS` env vars)

---

## Installing the GPU Agent

The agent runs on each GPU server and reports metrics to QuestDB.

### Requirements

- Linux with NVIDIA GPU(s)
- `nvidia-smi` installed and working
- Network access to QuestDB (port 9000)

### Step 1: Download the Binary

Pre-built binaries are in the `release/` directory:

```bash
# Check your architecture
uname -m
# x86_64 → use amd64
# aarch64 → use arm64

# Copy the binary to your GPU server
scp release/gpu-monitoring-agent-linux-amd64 user@gpu-server:/usr/local/bin/gpu-monitoring-agent
ssh user@gpu-server
chmod +x /usr/local/bin/gpu-monitoring-agent
```

### Step 2: Create Config File

```bash
sudo mkdir -p /etc/gpu-monitoring
sudo vim /etc/gpu-monitoring/gpu-monitoring-agent.conf
```

Paste this config:

```ini
[questdb]
host = 192.168.1.100      # QuestDB server IP
port = 9000                # QuestDB REST API port
user = admin               # QuestDB username
password = quest           # QuestDB password
protocol = http            # http or https
interval = 5               # Collection interval in seconds

[agent]
hostname = gpu-server-01   # Custom name (auto-detects hostname if empty)
```

### Step 3: Create systemd Service

```bash
sudo nano /etc/systemd/system/gpu-monitoring-agent.service
```

```ini
[Unit]
Description=GPU Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gpu-monitoring-agent -config /etc/gpu-monitoring/gpu-monitoring-agent.conf
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gpu-monitoring-agent
sudo systemctl status gpu-monitoring-agent
```

### Step 4: Verify

```bash
# Check agent logs
journalctl -u gpu-monitoring-agent -f

# Should see:
# agent started: host=gpu-server-01 questdb=192.168.1.100:9000/http interval=5s
# wrote 4 gpu rows, 12 process rows
```

---

## Configuration Reference

### Config File (`/etc/gpu-monitoring/gpu-monitoring-agent.conf`)

```ini
[questdb]
# QuestDB connection settings
host = localhost            # QuestDB host
port = 9000                 # QuestDB REST API port (default: 9000)
user = admin                # QuestDB username (default: admin)
password = quest            # QuestDB password (default: quest)
protocol = http             # http or https (default: http)
interval = 5                # Collection interval in seconds (default: 5)

[agent]
# Agent identification
hostname =                  # Custom hostname tag (empty = auto-detect)
```

### Environment Variables (override config file)

| Variable                     | Config Fallback   | Default  |
|------------------------------|-------------------|----------|
| `GPU_DASH_QUESTDB_HOST`      | `[questdb] host`  | localhost|
| `GPU_DASH_QUESTDB_PORT`      | `[questdb] port`  | 9000     |
| `GPU_DASH_QUESTDB_USER`      | `[questdb] user`  | admin    |
| `GPU_DASH_QUESTDB_PASSWORD`  | `[questdb] password`| quest  |
| `GPU_DASH_QUESTDB_PROTOCOL`  | `[questdb] protocol`| http   |
| `GPU_DASH_QUESTDB_INTERVAL`  | `[questdb] interval`| 5      |

Environment variables take precedence over config file values.

---

## Multi-Server Setup

Deploy the agent to multiple GPU servers, each pointing to the same QuestDB:

```
GPU Server 1 ──▶ agent ──┐
GPU Server 2 ──▶ agent ──┤──▶ QuestDB ◀── Frontend (1 web UI)
GPU Server 3 ──▶ agent ──┘
```

Each agent should have a unique `[agent] hostname` value:

```ini
# gpu-server-01
[agent]
hostname = gpu-server-01

# gpu-server-02
[agent]
hostname = gpu-server-02
```

The dashboard shows all servers, filterable by hostname.

---

## Frontend Environment Variables

| Variable       | Default                          | Description                    |
|----------------|----------------------------------|--------------------------------|
| `QUESTDB_URL`  | `http://localhost:9000`          | QuestDB REST API URL           |
| `QUESTDB_USER` | `admin`                          | QuestDB username               |
| `QUESTDB_PASS` | `quest`                          | QuestDB password               |
| `ADMIN_USER`   | `admin`                          | Dashboard login username       |
| `ADMIN_PASS`   | `admin`                          | Dashboard login password       |
| `ORIGIN`       | `http://localhost:5173`          | Allowed origin for CORS        |
| `HOST`         | `0.0.0.0`                       | Listen host                    |
| `PORT`         | `3000`                          | Listen port                    |

---

## Building from Source

### Agent (requires Go 1.23+)

```bash
# Native build
cd agent
go build -o gpu-monitoring-agent ./cmd/agent

# Cross-compile for Linux (amd64 + arm64) via Docker
make build-agent
# Binaries: release/gpu-monitoring-agent-linux-amd64, release/gpu-monitoring-agent-linux-arm64
```

### Frontend (requires Node.js 22+)

```bash
cd frontend
npm install
npm run build     # production build
node build        # run production server
```

### Docker

```bash
# All services
docker compose up --build

# Agent only (cross-compile)
docker build -f Dockerfile.agent -t gpu-monitoring-agent .
```

---

## Database Schema

Tables are auto-created by QuestDB on first write.

### `gpu_metrics`

| Column            | Type   | Description                    |
|-------------------|--------|--------------------------------|
| timestamp         | TIMESTAMP | Collection time              |
| hostname          | SYMBOL | Server hostname                |
| gpu_id            | SYMBOL | GPU index (0, 1, 2...)         |
| gpu_uuid          | SYMBOL | NVIDIA GPU UUID                |
| gpu_name          | SYMBOL | GPU model name                 |
| utilization_gpu   | DOUBLE | GPU utilization %              |
| utilization_mem   | DOUBLE | Memory utilization %           |
| memory_used       | DOUBLE | Used VRAM (MiB)                |
| memory_total      | DOUBLE | Total VRAM (MiB)               |
| temperature       | DOUBLE | Temperature (°C)               |
| power_draw        | DOUBLE | Power draw (W)                 |
| power_limit       | DOUBLE | Power limit (W)                |
| fan_speed         | DOUBLE | Fan speed %                    |

### `gpu_processes`

| Column           | Type   | Description                   |
|------------------|--------|-------------------------------|
| timestamp        | TIMESTAMP | Collection time             |
| hostname         | SYMBOL | Server hostname               |
| gpu_id           | SYMBOL | GPU index                     |
| pid              | INT    | Process ID                    |
| process_name     | SYMBOL | Process name                  |
| username         | SYMBOL | Linux user                    |
| container_id     | SYMBOL | Docker container ID (12 char) |
| container_name   | SYMBOL | Docker container name         |
| used_memory      | DOUBLE | GPU memory used (MiB)         |

---

## Troubleshooting

### Agent: "ERROR: collect gpus: nvidia-smi ..."

- Ensure `nvidia-smi` is installed: `nvidia-smi`
- Check NVIDIA driver: `nvidia-smi --query-gpu=name --format=csv`
- If running in Docker, use `--gpus all` or `--privileged`

### Agent: "ERROR: write to questdb: ..."

- Verify QuestDB is reachable: `curl http://questdb-host:9000`
- Check credentials match config
- Ensure QuestDB REST API is on port 9000 (not PGWire 9001)

### Frontend: No data shown

- Verify agent is running: `journalctl -u gpu-monitoring-agent`
- Check QuestDB web console: `http://questdb-host:9000`
- Query manually: `SELECT count() FROM gpu_metrics`

### Frontend: Login fails

- Default credentials: `admin` / `admin`
- Change via `ADMIN_USER` / `ADMIN_PASS` environment variables
- Clear browser cookies if needed

---

## Project Structure

```
gpu-server-monitoring/
├── agent/                         # Go agent (zero dependencies)
│   ├── cmd/agent/main.go          # Entrypoint
│   ├── internal/
│   │   ├── config/config.go       # INI parser + env var overrides
│   │   ├── collectors/
│   │   │   ├── nvidia.go          # nvidia-smi GPU + process collection
│   │   │   └── docker.go          # Container detection via /proc/<pid>/cgroup
│   │   └── writers/
│   │       └── questdb.go         # QuestDB HTTP /imp ingest
│   └── go.mod                     # Go 1.23, zero external deps
├── frontend/                      # SvelteKit 2 + Svelte 5 + Tailwind CSS 4
│   ├── src/
│   │   ├── routes/                # Pages: login, dashboard, GPU detail, users, containers, history
│   │   ├── lib/
│   │   │   ├── server/questdb.ts  # QuestDB REST API proxy (server-side only)
│   │   │   └── components/        # GpuCard, ProcessTable (Svelte 5 runes)
│   │   └── app.css                # Tabler-style CSS
│   ├── Dockerfile.frontend
│   └── package.json
├── etc/
│   └── gpu-monitoring-agent.conf  # Sample config
├── release/                       # Pre-built Linux binaries (amd64 + arm64)
├── Dockerfile.agent               # Multi-stage Go build
├── docker-compose.yml             # QuestDB + frontend
├── Makefile                       # build-agent, clean
└── README.md
```

---

## License

Internal use only.
