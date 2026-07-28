import { env } from '$env/dynamic/private';

const QUESTDB_URL = env.QUESTDB_URL || 'http://localhost:9000';
const QUESTDB_USER = env.QUESTDB_USER || 'admin';
const QUESTDB_PASS = env.QUESTDB_PASS || 'quest';

export async function querySql<T = Record<string, unknown>>(sql: string): Promise<T[]> {
  const params = new URLSearchParams({ query: sql });
  const res = await fetch(`${QUESTDB_URL}/exec?${params.toString()}`, {
    headers: {
      Authorization: 'Basic ' + btoa(`${QUESTDB_USER}:${QUESTDB_PASS}`)
    }
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`QuestDB query failed (${res.status}): ${body}`);
  }

  const json = await res.json();

  if (!json.dataset || json.dataset.length === 0) return [];

  const cols = json.columns.map((c: { name: string }) => c.name);
  return json.dataset.map((row: unknown[]) => {
    const obj: Record<string, unknown> = {};
    cols.forEach((col: string, i: number) => {
      obj[col] = row[i];
    });
    return obj as T;
  });
}

export interface GpuMetricsRow {
  timestamp: string;
  hostname: string;
  gpu_id: string;
  gpu_uuid: string;
  gpu_name: string;
  utilization_gpu: number;
  utilization_mem: number;
  memory_used: number;
  memory_total: number;
  temperature: number;
  power_draw: number;
  power_limit: number;
  fan_speed: number;
}

export interface GpuProcessRow {
  timestamp: string;
  hostname: string;
  gpu_id: string;
  pid: number;
  process_name: string;
  username: string;
  container_id: string;
  container_name: string;
  used_memory: number;
}

export interface GpuTrendRow {
  timestamp: string;
  hostname: string;
  gpu_id: string;
  memory_used: number;
  memory_total: number;
  power_draw: number;
  power_limit: number;
}

function dedupLatestByHostGpu(rows: GpuMetricsRow[]): GpuMetricsRow[] {
  const map = new Map<string, GpuMetricsRow>();
  for (const r of rows) {
    const key = `${r.hostname}|${r.gpu_id}`;
    const existing = map.get(key);
    if (!existing || r.timestamp > existing.timestamp) {
      map.set(key, r);
    }
  }
  return [...map.values()].sort((a, b) => a.hostname.localeCompare(b.hostname) || a.gpu_id.localeCompare(b.gpu_id));
}

function dedupLatestByHostGpuPid(rows: GpuProcessRow[]): GpuProcessRow[] {
  const map = new Map<string, GpuProcessRow>();
  for (const r of rows) {
    const key = `${r.hostname}|${r.gpu_id}|${r.pid}`;
    const existing = map.get(key);
    if (!existing || r.timestamp > existing.timestamp) {
      map.set(key, r);
    }
  }
  return [...map.values()].sort((a, b) => a.hostname.localeCompare(b.hostname) || a.gpu_id.localeCompare(b.gpu_id) || a.pid - b.pid);
}

export async function getLatestGpuMetrics(): Promise<GpuMetricsRow[]> {
  const rows = await querySql<GpuMetricsRow>(`
    SELECT * FROM gpu_metrics
    WHERE "timestamp" > dateadd('s', -15, now())
    ORDER BY hostname, gpu_id, "timestamp" DESC
  `);
  return dedupLatestByHostGpu(rows);
}

export async function getLatestProcesses(): Promise<GpuProcessRow[]> {
  const rows = await querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE "timestamp" > dateadd('s', -15, now())
    ORDER BY hostname, gpu_id, pid, "timestamp" DESC
  `);
  return dedupLatestByHostGpuPid(rows);
}

export async function getGpuHistory(
  hostname: string,
  gpuId: string,
  from: string,
  to: string,
  sampleBy = '1m'
): Promise<GpuMetricsRow[]> {
  return querySql<GpuMetricsRow>(`
    SELECT * FROM gpu_metrics
    WHERE hostname = '${hostname}'
      AND gpu_id = '${gpuId}'
      AND "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${sampleBy} ALIGN TO CALENDAR
    ORDER BY "timestamp"
  `);
}

export async function getGpuProcessesHistory(
  hostname: string,
  gpuId: string,
  from: string,
  to: string,
  sampleBy = '1m'
): Promise<GpuProcessRow[]> {
  return querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE hostname = '${hostname}'
      AND gpu_id = '${gpuId}'
      AND "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${sampleBy} ALIGN TO CALENDAR
    ORDER BY "timestamp"
  `);
}

export async function getHistory(
  from: string,
  to: string,
  sampleBy = '5m'
): Promise<GpuMetricsRow[]> {
  return querySql<GpuMetricsRow>(`
    SELECT * FROM gpu_metrics
    WHERE "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${sampleBy} ALIGN TO CALENDAR
    ORDER BY "timestamp", gpu_id
  `);
}

export async function getGpuTrend(
  from: string,
  to: string,
  sampleBy = '30m'
): Promise<GpuTrendRow[]> {
  return querySql<GpuTrendRow>(`
    SELECT
      "timestamp",
      hostname,
      gpu_id,
      avg(memory_used) memory_used,
      max(memory_total) memory_total,
      avg(power_draw) power_draw,
      max(power_limit) power_limit
    FROM gpu_metrics
    WHERE "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${sampleBy} ALIGN TO CALENDAR
    ORDER BY "timestamp", hostname, gpu_id
  `);
}

export async function getUsersSummary(): Promise<
  { username: string; hostname: string; gpu_id: string; total_memory: number; process_count: number }[]
> {
  const rows = await querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE "timestamp" > dateadd('s', -15, now())
  `);
  const latest = dedupLatestByHostGpuPid(rows);
  const map = new Map<string, { username: string; hostname: string; gpu_id: string; total_memory: number; process_count: number }>();
  for (const r of latest) {
    const key = `${r.username}|${r.hostname}|${r.gpu_id}`;
    const existing = map.get(key);
    if (existing) {
      existing.total_memory += r.used_memory;
      existing.process_count += 1;
    } else {
      map.set(key, { username: r.username, hostname: r.hostname, gpu_id: r.gpu_id, total_memory: r.used_memory, process_count: 1 });
    }
  }
  return [...map.values()].sort((a, b) => b.total_memory - a.total_memory);
}

export async function getContainersSummary(): Promise<
  {
    container_id: string;
    container_name: string;
    hostname: string;
    gpu_id: string;
    total_memory: number;
    process_count: number;
  }[]
> {
  const rows = await querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE "timestamp" > dateadd('s', -15, now())
  `);
  const latest = dedupLatestByHostGpuPid(rows).filter(r => r.container_id !== '' && r.container_id != null);
  const map = new Map<string, { container_id: string; container_name: string; hostname: string; gpu_id: string; total_memory: number; process_count: number }>();
  for (const r of latest) {
    const key = `${r.container_id}|${r.hostname}|${r.gpu_id}`;
    const existing = map.get(key);
    if (existing) {
      existing.total_memory += r.used_memory;
      existing.process_count += 1;
    } else {
      map.set(key, { container_id: r.container_id, container_name: r.container_name, hostname: r.hostname, gpu_id: r.gpu_id, total_memory: r.used_memory, process_count: 1 });
    }
  }
  return [...map.values()].sort((a, b) => b.total_memory - a.total_memory);
}
