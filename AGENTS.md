# GPU Server Monitoring — Agent Instructions

GPU metrics collector agent + web dashboard. Agent polls `nvidia-smi`, writes to QuestDB. Frontend reads QuestDB via SvelteKit server routes.

## Project Structure

```
agent/
  cmd/agent/main.go          # Entrypoint — flag parsing, ticker loop, signal handling
  internal/
    config/config.go          # INI parser + env var overrides
    collectors/nvidia.go      # nvidia-smi GPU + process collection (pure CSV parsing)
    collectors/docker.go      # Container detection via /proc/<pid>/cgroup regex
    writers/questdb.go        # QuestDB HTTP /imp ingest + /proc UID lookup
  go.mod                      # Go 1.23, zero external dependencies (stdlib only)
frontend/
  src/
    routes/                   # SvelteKit pages — login, /, /gpus/[id], /users, /containers, /history
    lib/
      server/questdb.ts       # QuestDB REST API proxy (server-side only, credentials never reach client)
      components/             # GpuCard, ProcessTable (Svelte 5 runes)
      stores/theme.ts         # Dark/light mode persistence
  package.json                # SvelteKit 2 + Svelte 5 + Tailwind CSS 4 + @lucide/svelte
Dockerfile.agent              # Multi-stage Alpine build → amd64 + arm64 binaries
Dockerfile.frontend           # Multi-stage Node build → production SvelteKit adapter-node
docker-compose.yml            # QuestDB 8.2.3 + frontend (ports: 5173→3000, 9000 web, 9009 ILP, 9001 PG, 9003 REST)
etc/gpu-monitoring-agent.conf  # Sample config (INI format)
Makefile                      # build-agent, clean
release/                      # Pre-built Linux binaries (amd64 + arm64)
```

## Build & Run

```bash
# Agent — cross-compile via Docker
make build-agent

# Agent — run locally (requires nvidia-smi + QuestDB)
go run ./cmd/agent -config etc/gpu-monitoring-agent.conf

# Frontend — dev mode
cd frontend && npm install && npm run dev

# Frontend — production build
cd frontend && npm run build && node build

# All services via Docker Compose
docker compose up --build
```

No test suite exists. No lint/typecheck config beyond `go vet` and `svelte-check`.

## Frontend Architecture

- **Stack**: SvelteKit 2 + Svelte 5 (runes: `$state`, `$derived`, `$props`) + TypeScript strict + Tailwind CSS 4
- **Icons**: `@lucide/svelte` (not deprecated `lucide-svelte`)
- **Adapter**: `@sveltejs/adapter-node` — production build runs as `node build`
- **QuestDB access**: all queries go through SvelteKit `+server.ts` API routes → `src/lib/server/questdb.ts` → QuestDB REST `/exec` endpoint. Credentials stay server-side only.
- **Auth**: cookie-based session (HTTP-only). Hardcoded admin user via `ADMIN_USER`/`ADMIN_PASS` env vars (default: admin/admin). Login at `/login`, all other routes protected via `hooks.server.ts`.
- **Dark mode**: Tailwind `dark:` variant with class strategy on `<html>`. Theme persisted in `localStorage`, respects `prefers-color-scheme`.
- **Data refresh**: dashboard polls `/api/gpu` every 5s. GPU detail + history pages fetch on demand.
- **Env vars**: `QUESTDB_URL`, `QUESTDB_USER`, `QUESTDB_PASS`, `ADMIN_USER`, `ADMIN_PASS`, `ORIGIN`

## Routes

| Route | Purpose |
|-------|---------|
| `/login` | Username/password, centered card |
| `/` | Dashboard — KPI cards + GPU cards + process table |
| `/gpus/[id]` | Single GPU — live stats + history table with time range + processes |
| `/users` | Usage grouped by Linux user |
| `/containers` | Usage grouped by Docker container |
| `/history` | Historical query with time range picker + sample-by selector |

## Agent Architecture

- **Config**: INI at `/etc/gpu-monitoring/gpu-monitoring-agent.conf` (fail fast if missing)
- **Env var overrides**: `GPU_DASH_QUESTDB_HOST`, `_PORT`, `_USER`, `_PASSWORD`, `_PROTOCOL`, `_INTERVAL` — take precedence over INI values
- **Collection cycle**: ticker-based (configurable interval, default 5s), runs once immediately on start
- **GPU metrics**: `nvidia-smi --query-gpu=... --format=csv,noheader,nounits` — pure CSV parsing, no nvidia-ml-py
- **Process mapping**: `nvidia-smi --query-compute-apps=...`, PID→user via `/proc/<pid>/status` Uid + `/etc/passwd`
- **Container detection**: `/proc/<pid>/cgroup` regex for docker/containerd IDs (no docker socket dependency)
- **QuestDB write**: HTTP POST to `/imp` endpoint with basic auth — not ILP TCP, not Python questdb package
- **Graceful shutdown**: catches SIGTERM/SIGINT, clean log and exit
- **Logging**: stdout with date+time+microseconds (designed for journald capture)

## Key Gotchas

- `agent/go.mod` has **zero dependencies** — everything is stdlib. Keep it that way unless there's a strong reason.
- QuestDB agent writes via HTTP `/imp` (not ILP TCP port 9009). Docker-compose exposes 9009 but the agent doesn't use it.
- Frontend queries QuestDB via REST `/exec` (port 9000), not PGWire. Same endpoint the web console uses.
- `DetectContainer()` in `docker.go` runs per-process in `WriteBatch()` — N reads of `/proc/<pid>/cgroup` per write cycle.
- `lookupUID()` reads `/etc/passwd` on every call (no caching). Same caveat.
- Config parser is hand-rolled (not `go-ini` or similar). Section/key parsing only. Comments: `#` or `;` prefix.
- `@lucide/svelte` is the current package — `lucide-svelte` is deprecated and will fail to install.
- Frontend uses Svelte 5 runes syntax (`$state`, `$derived`, `$props`) — not Svelte 4 stores or `let` reactive declarations.
- Tailwind CSS 4 uses `@import 'tailwindcss'` and `@custom-variant dark` in CSS — not `@tailwind` directives or `tailwind.config.js`.

## Out of Scope

- Multi-cluster federation, alerting, K8s operator, GPU MIG, billing
