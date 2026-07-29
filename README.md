# GPU Server Monitoring

Real-time dashboard for monitoring NVIDIA (and AMD) GPU servers. Lightweight Go agent polls `nvidia-smi` (or `rocm-smi`) every few seconds and sends metrics to QuestDB time-series database. SvelteKit frontend displays everything in a clean web dashboard.

**Use case:** You have 1–50 GPU servers and want a simple web UI to see utilization, memory, temperature, power, running processes, users, and Docker containers — without paying for a commercial monitoring tool.

## Screenshots (what you'll build)

![GPU Server Monitoring](GPU%20Server%20Monitoring.png)

```
Dashboard:    /  → 6 GPU cards across 3 servers, KPI totals, process table
GPU detail:   /gpus/gpu-server-01/0 → live stats + history chart + processes
Users:        /users → GPU memory grouped by Linux user
Containers:   /containers → GPU usage grouped by Docker container
History:      /history → time-range query with sample-by interval
Settings:     /settings → brand name, logo, manage non-admin users
```

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start (try it now)](#quick-start-try-it-now)
3. [Production Setup (step by step)](#production-setup-step-by-step)
   - [Step 1: Deploy QuestDB + Frontend](#step-1-deploy-questdb--frontend)
   - [Step 2: Install Agent on GPU Servers](#step-2-install-agent-on-gpu-servers)
   - [Step 3: Verify Data Flow](#step-3-verify-data-flow)
   - [Step 4: Configure Dashboard](#step-4-configure-dashboard)
   - [Step 5: Add More GPU Servers](#step-5-add-more-gpu-servers)
4. [Configuration Reference](#configuration-reference)
5. [Building from Source](#building-from-source)
6. [Project Structure](#project-structure)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### For the web dashboard server (any machine):
- **Docker** + **Docker Compose** (v2.24+)
- Git
- 1 GB RAM, 10 GB disk

### For each GPU server (the machines with NVIDIA/AMD GPUs):
- Linux (Ubuntu 20.04+, Debian 11+, RHEL 8+, etc.)
- **NVIDIA drivers** with `nvidia-smi` working (or AMD ROCm with `rocm-smi`)
- Network access to the QuestDB server (port 9000 for REST, 9009 for ILP)
- Systemd (for auto-start)

### For building from source (optional):
- Go 1.23+ (agent)
- Node.js 22+ (frontend)

---

## Quick Start (try it now)

This runs QuestDB + the frontend on your local machine. You won't see GPU data (no GPU agent yet), but you can see the UI, play with settings, and confirm everything works.

```bash
# 1. Clone the repo
git clone https://github.com/your-org/gpu-server-monitoring.git
cd gpu-server-monitoring

# 2. Start QuestDB and frontend
docker compose up -d

# 3. Open in browser
open http://localhost:5173
```

**Login:** `admin` / `admin`

You'll see an empty dashboard. That's normal — no GPU agent is sending data yet. Proceed to [Production Setup](#production-setup-step-by-step) to connect real GPU servers.

### What just happened?

| Container | Image | Port | Purpose |
|-----------|-------|------|---------|
| `questdb` | questdb/questdb:8.2.3 | 9000 (web console), 9009 (ILP ingest) | Time-series database |
| `gpu-monitoring-frontend` | local build | 5173 → 3000 | Web dashboard |

QuestDB web console: http://localhost:9000

---

## Production Setup (step by step)

### Step 1: Deploy QuestDB + Frontend

On your dashboard server (can be a small VM, does NOT need a GPU):

```bash
git clone https://github.com/your-org/gpu-server-monitoring.git
cd gpu-server-monitoring
docker compose up -d
```

**That's it.** You now have a running dashboard at `http://your-server-ip:5173`.

#### Custom admin credentials (important!)

Edit `docker-compose.yml` before starting, or use an `.env` file:

```bash
# Create .env file in project root
echo "ADMIN_USER=myadmin" >> .env
echo "ADMIN_PASS=mysecretpassword" >> .env
echo "ORIGIN=http://your-server-ip:5173" >> .env
```

Then:
```bash
docker compose up -d
```

---

### Step 2: Install Agent on GPU Servers

Repeat this on **every** machine that has GPUs.

#### 2a. Download the agent binary

Pre-built binaries are in the `release/` folder:

```bash
# On the GPU server:
# Check your architecture first
uname -m
#   x86_64 → use gpu-monitoring-agent-linux-amd64
#   aarch64 → use gpu-monitoring-agent-linux-arm64

# Copy from your dashboard server (or download directly)
# From dashboard server:
scp release/gpu-monitoring-agent-linux-amd64 user@gpu-server-01:/tmp/

# On the GPU server:
sudo mv /tmp/gpu-monitoring-agent-linux-amd64 /usr/local/bin/gpu-monitoring-agent
sudo chmod +x /usr/local/bin/gpu-monitoring-agent
```

#### 2b. Create config file

```bash
sudo mkdir -p /etc/gpu-monitoring
sudo nano /etc/gpu-monitoring/gpu-monitoring-agent.conf
```

Paste this, changing the values to match your setup:

```ini
[questdb]
host = 192.168.1.100      # <-- IP of your QuestDB server
port = 9000                # REST API port (for health checks)
ilp_port = 9009            # ILP TCP port (for data ingest, default 9009)
ilp_auth =                 # ILP auth token if set in QuestDB (leave blank if none)
user = admin               # QuestDB user
password = quest           # QuestDB password
protocol = http
interval = 5               # Collect and send every 5 seconds

[agent]
hostname = gpu-server-01   # Unique name for this server (auto-detects hostname if empty)
```

> **Port 9009 vs 9000:** The agent writes data via QuestDB's **ILP TCP protocol** on port 9009 (faster, built for time-series). Port 9000 is the REST API used by the dashboard and web console.

#### 2c. Create systemd service (auto-start on boot)

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

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gpu-monitoring-agent
sudo systemctl status gpu-monitoring-agent  # Should show "active (running)"
```

#### 2d. Verify agent is working

```bash
# Watch the logs
sudo journalctl -u gpu-monitoring-agent -f
```

You should see output like:

```
agent started: host=gpu-server-01 backend=nvidia questdb=192.168.1.100:9000/ilp:9009 interval=5s
wrote 2 gpu rows, 5 process rows
wrote 2 gpu rows, 4 process rows
```

The number of GPU rows = number of GPUs in the server. Process rows = running processes using GPU.

---

### Step 3: Verify Data Flow

Open your dashboard: `http://your-server-ip:5173`

- Dashboard should show GPU cards with utilization, temperature, memory, etc.
- If you have multiple GPUs, each gets its own card.
- Click on a GPU card to see detailed history and charts.
- The "Processes" table shows what's running on each GPU, which user, and which container.

**Still no data?** See [Troubleshooting](#troubleshooting).

#### QuestDB Web Console (for debugging)

Open `http://your-server-ip:9000` and run:

```sql
SELECT * FROM gpu_metrics LIMIT 5;
SELECT count() FROM gpu_metrics;
SELECT * FROM gpu_processes LIMIT 5;
```

If these return empty, the agent isn't writing. Check agent logs (`journalctl -u gpu-monitoring-agent -f`).

---

### Step 4: Configure Dashboard

Open the dashboard and log in as admin.

| Feature | How |
|---------|-----|
| **Brand name** | Go to `/settings`, General tab — change "GPU Dashboard" to your org name |
| **Logo** | Same page — set a logo URL (e.g., your company logo hosted somewhere) |
| **Add users** | Settings → Users tab — create non-admin accounts so colleagues can view |
| **Dark mode** | Toggle in sidebar |

Logo appears in the sidebar. Brand name shows in the page title and sidebar header.

---

### Step 5: Add More GPU Servers

Repeat [Step 2](#step-2-install-agent-on-gpu-servers) for each additional server. Give each one a unique hostname:

```ini
# On gpu-server-02
[agent]
hostname = gpu-server-02

# On gpu-server-03
[agent]
hostname = gpu-server-03
```

All agents send data to the **same QuestDB instance**. The dashboard automatically groups by hostname and shows all servers together.

For 50+ servers, consider a more powerful QuestDB machine with more RAM and fast SSD storage.

---

## Configuration Reference

### Agent Config File (`/etc/gpu-monitoring/gpu-monitoring-agent.conf`)

```ini
[questdb]
host = 192.168.1.100       # QuestDB server hostname or IP
port = 9000                # REST API port (for dashboard queries)
ilp_port = 9009            # ILP TCP port for data ingest (default: 9009)
ilp_auth =                 # ILP authentication token (optional)
user = admin               # QuestDB username (default: admin)
password = quest           # QuestDB password (default: quest)
protocol = http            # http or https (default: http)
interval = 5               # Seconds between collection cycles (default: 5)

[agent]
hostname = gpu-server-01   # Display name in dashboard (empty = use OS hostname)
```

### Environment Variables (override config file)

| Variable                     | Overrides          | Default     |
|------------------------------|--------------------|-------------|
| `GPU_DASH_QUESTDB_HOST`      | `[questdb] host`   | `localhost` |
| `GPU_DASH_QUESTDB_PORT`      | `[questdb] port`   | `9000`      |
| `GPU_DASH_QUESTDB_ILP_PORT`  | `[questdb] ilp_port` | `9009`   |
| `GPU_DASH_QUESTDB_ILP_AUTH`  | `[questdb] ilp_auth` | (empty)  |
| `GPU_DASH_QUESTDB_USER`      | `[questdb] user`   | `admin`     |
| `GPU_DASH_QUESTDB_PASSWORD`  | `[questdb] password` | `quest`  |
| `GPU_DASH_QUESTDB_PROTOCOL`  | `[questdb] protocol` | `http`   |
| `GPU_DASH_QUESTDB_INTERVAL`  | `[questdb] interval` | `5`      |

Environment variables take precedence over config file values. Useful when deploying with Docker or Kubernetes.

### Frontend Environment Variables

| Variable       | Default                 | Description                             |
|----------------|-------------------------|-----------------------------------------|
| `QUESTDB_URL`  | `http://localhost:9000` | QuestDB REST API endpoint               |
| `QUESTDB_USER` | `admin`                 | QuestDB username for queries            |
| `QUESTDB_PASS` | `quest`                 | QuestDB password                        |
| `ADMIN_USER`   | `admin`                 | Dashboard admin login username          |
| `ADMIN_PASS`   | `admin`                 | Dashboard admin login password          |
| `SETTINGS_DIR` | `data/settings`         | Directory for brand.json and users.json |
| `ORIGIN`       | `http://localhost:5173` | URL where dashboard is hosted (required for production) |
| `HOST`         | `0.0.0.0`              | Frontend listen host                    |
| `PORT`         | `3000`                  | Frontend listen port                    |

### Database Schema

Tables auto-create on first write. No migration needed.

#### `gpu_metrics` — one row per GPU per collection cycle

| Column            | Type      | Example        | Description           |
|-------------------|-----------|----------------|-----------------------|
| timestamp         | TIMESTAMP | 2026-07-29T... | Collection time       |
| hostname          | SYMBOL    | gpu-server-01  | Server identifier     |
| gpu_id            | INT       | 0              | GPU index             |
| gpu_uuid          | SYMBOL    | GPU-abc123...  | NVIDIA GPU UUID       |
| gpu_name          | SYMBOL    | RTX 4090       | GPU model name        |
| utilization_gpu   | DOUBLE    | 45.2           | GPU utilization %     |
| utilization_mem   | DOUBLE    | 30.0           | Memory utilization %  |
| memory_used       | DOUBLE    | 8192           | Used VRAM (MiB)       |
| memory_total      | DOUBLE    | 24576          | Total VRAM (MiB)      |
| temperature       | DOUBLE    | 68.5           | Temperature (°C)      |
| power_draw        | DOUBLE    | 250.0          | Power draw (W)        |
| power_limit       | DOUBLE    | 350.0          | Power limit (W)       |
| fan_speed         | DOUBLE    | 45.0           | Fan speed %           |

#### `gpu_processes` — one row per process per collection cycle

| Column         | Type      | Example             | Description            |
|----------------|-----------|---------------------|------------------------|
| timestamp      | TIMESTAMP | 2026-07-29T...      | Collection time        |
| hostname       | SYMBOL    | gpu-server-01       | Server identifier      |
| gpu_id         | INT       | 0                   | GPU index              |
| pid            | INT       | 12345               | Process ID             |
| process_name   | SYMBOL    | python3             | Process name           |
| username       | SYMBOL    | alice               | Linux user             |
| container_id   | SYMBOL    | a1b2c3d4e5f6        | Docker container ID    |
| container_name | SYMBOL    | my-training-job     | Docker container name  |
| used_memory    | DOUBLE    | 4096                | GPU memory used (MiB)  |

---

## Building from Source

### Agent (Go)

```bash
cd agent
go build -o gpu-monitoring-agent ./cmd/agent

# Or cross-compile for all Linux targets via Docker:
make build-agent
# Output: release/gpu-monitoring-agent-linux-amd64
#         release/gpu-monitoring-agent-linux-arm64
```

### Frontend (Node.js)

```bash
cd frontend
npm install
npm run dev        # Development server with hot reload
npm run build      # Production build
node build         # Run production server on port 3000
```

### Full Docker Build

```bash
# Build everything locally
docker compose up --build

# Or build individual images
docker build -f Dockerfile.agent -t gpu-monitoring-agent .
docker build -f Dockerfile.frontend -t gpu-monitoring-frontend .
```

---

## Project Structure

```
gpu-server-monitoring/
├── agent/                         # Go agent (zero external dependencies)
│   ├── cmd/agent/main.go          # Entry point, flag parsing, ticker loop
│   ├── internal/
│   │   ├── config/config.go       # INI parser + env var overrides
│   │   ├── collectors/
│   │   │   ├── nvidia.go          # nvidia-smi GPU + process collection
│   │   │   ├── rocm.go            # rocm-smi AMD GPU collection
│   │   │   └── docker.go          # Container detection via /proc/<pid>/cgroup
│   │   └── writers/
│   │       └── questdb.go         # QuestDB ILP TCP writer
│   └── go.mod                     # Go 1.23
├── frontend/                      # SvelteKit 2 + Svelte 5 + Tailwind CSS 4
│   ├── src/
│   │   ├── routes/                # Pages: /login, /, /gpus/[id], /users, /containers, /history, /settings
│   │   ├── lib/
│   │   │   ├── server/
│   │   │   │   ├── questdb.ts     # QuestDB REST API proxy (server-side only)
│   │   │   │   └── settingsStore.ts  # Brand + user file persistence
│   │   │   └── components/        # GpuCard, ProcessTable (Svelte 5 runes)
│   │   └── hooks.server.ts        # Auth middleware
│   ├── Dockerfile.frontend
│   └── package.json
├── etc/
│   └── gpu-monitoring-agent.conf  # Sample agent config
├── release/                       # Pre-built Linux binaries
├── Dockerfile.agent
├── docker-compose.yml             # QuestDB + frontend
├── Makefile
└── README.md
```

---

## Troubleshooting

### "No data" on dashboard

```
1. Check agent is running:
   ssh gpu-server
   sudo systemctl status gpu-monitoring-agent

2. Check agent logs for errors:
   sudo journalctl -u gpu-monitoring-agent -f

3. Check QuestDB has data:
   Open http://questdb-ip:9000 in browser
   Run: SELECT count() FROM gpu_metrics
   Run: SELECT count() FROM gpu_processes

4. If QuestDB shows data but dashboard doesn't:
   - Wait 15 seconds (dashboard uses a rolling 15s window)
   - Check frontend logs: docker compose logs frontend
   - Verify QUESTDB_URL is correct in docker-compose.yml
```

### Agent won't start

```bash
# "ERROR: open config: ..." → config file missing or wrong path
# Fix: create /etc/gpu-monitoring/gpu-monitoring-agent.conf

# "ERROR: collect gpus: nvidia-smi ..." → NVIDIA driver issue
nvidia-smi  # Should show GPU info
# If not, install NVIDIA drivers

# "FATAL: no GPU tool found" → no GPU driver at all
# Install NVIDIA drivers or AMD ROCm
```

### Agent can't write to QuestDB

```bash
# "connect to questdb ILP :9009: ..." → QuestDB not reachable
curl http://questdb-ip:9000  # Should return QuestDB version
# From GPU server, test: nc -zv questdb-ip 9009

# "ERROR: write to questdb: connect to questdb ILP ..."
# Check firewall: port 9009 must be open between agent and QuestDB
# Check QuestDB is running: docker compose ps
```

### Network diagram (firewall ports)

```
GPU Server                    Dashboard Server
┌──────────┐                  ┌──────────────────┐
│  Agent   │── port 9009 ────▶│  QuestDB         │
│  (out)   │── port 9000 ────▶│  (in)            │
└──────────┘                  │                  │
                              │  Frontend        │
Your browser ── port 5173 ───▶│  (in)            │
                              └──────────────────┘
```

If QuestDB and Frontend are on different machines, adjust ports accordingly.

### Login issues

```
# Default credentials: admin / admin
# Changed via ADMIN_USER / ADMIN_PASS env vars

# Can't log in with admin after adding users?
# Admin login uses ADMIN_USER/ADMIN_PASS env vars
# Non-admin users are managed via Settings → Users tab

# Clear browser cookies and try again
```

### Docker volume permission issues

```bash
# If settings file fails to write:
# The SETTINGS_DIR directory must be writable by the container user
sudo chown -R 1000:1000 ./data/settings
# Or delete and let Docker recreate it
```

---

## License

Apache 2.0. See [LICENSE](LICENSE).
