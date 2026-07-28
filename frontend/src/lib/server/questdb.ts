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

const ALLOWED_SAMPLE_BY = new Set(['30s', '1m', '5m', '15m', '30m', '1h', '1d']);

const GPU_METRICS_SAMPLE_SELECT = `
      "timestamp",
      hostname,
      gpu_id,
      first(gpu_uuid) gpu_uuid,
      first(gpu_name) gpu_name,
      avg(utilization_gpu) utilization_gpu,
      avg(utilization_mem) utilization_mem,
      avg(memory_used) memory_used,
      max(memory_total) memory_total,
      avg(temperature) temperature,
      avg(power_draw) power_draw,
      max(power_limit) power_limit,
      avg(fan_speed) fan_speed
`;

function normalizeSampleBy(sampleBy: string) {
  if (!ALLOWED_SAMPLE_BY.has(sampleBy)) {
    throw new Error(`Unsupported sample_by interval: ${sampleBy}`);
  }
  return sampleBy;
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
  const interval = normalizeSampleBy(sampleBy);
  return querySql<GpuMetricsRow>(`
    SELECT
${GPU_METRICS_SAMPLE_SELECT}
    FROM gpu_metrics
    WHERE hostname = '${hostname}'
      AND gpu_id = '${gpuId}'
      AND "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${interval} ALIGN TO CALENDAR
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
  const interval = normalizeSampleBy(sampleBy);
  return querySql<GpuProcessRow>(`
    SELECT
      "timestamp",
      hostname,
      gpu_id,
      process_name,
      username,
      container_id,
      container_name,
      first(pid) pid,
      avg(used_memory) used_memory
    FROM gpu_processes
    WHERE hostname = '${hostname}'
      AND gpu_id = '${gpuId}'
      AND "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${interval} ALIGN TO CALENDAR
    ORDER BY "timestamp"
  `);
}

export async function getHistory(
  from: string,
  to: string,
  sampleBy = '5m'
): Promise<GpuMetricsRow[]> {
  const interval = normalizeSampleBy(sampleBy);
  return querySql<GpuMetricsRow>(`
    SELECT
${GPU_METRICS_SAMPLE_SELECT}
    FROM gpu_metrics
    WHERE "timestamp" BETWEEN '${from}' AND '${to}'
    SAMPLE BY ${interval} ALIGN TO CALENDAR
    ORDER BY "timestamp", hostname, gpu_id
  `);
}

export async function getGpuTrend(
  from: string,
  to: string,
  sampleBy = '30m'
): Promise<GpuTrendRow[]> {
  const interval = normalizeSampleBy(sampleBy);
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
    SAMPLE BY ${interval} ALIGN TO CALENDAR
    ORDER BY "timestamp", hostname, gpu_id
  `);
}

export async function getUsersSummary(): Promise<
  { username: string; hostname: string; gpu_id: string; total_memory: number; memory_total: number; process_count: number }[]
> {
  const rows = await querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE "timestamp" > dateadd('s', -15, now())
  `);
  const gpuRows = await getLatestGpuMetrics();
  const memoryTotals = new Map(gpuRows.map((gpu) => [`${gpu.hostname}|${gpu.gpu_id}`, gpu.memory_total ?? 0]));
  const latest = dedupLatestByHostGpuPid(rows);
  const map = new Map<string, { username: string; hostname: string; gpu_id: string; total_memory: number; memory_total: number; process_count: number }>();
  for (const r of latest) {
    const key = `${r.username}|${r.hostname}|${r.gpu_id}`;
    const memoryTotal = memoryTotals.get(`${r.hostname}|${r.gpu_id}`) ?? 0;
    const existing = map.get(key);
    if (existing) {
      existing.total_memory += r.used_memory;
      existing.process_count += 1;
      existing.memory_total = Math.max(existing.memory_total, memoryTotal);
    } else {
      map.set(key, { username: r.username, hostname: r.hostname, gpu_id: r.gpu_id, total_memory: r.used_memory, memory_total: memoryTotal, process_count: 1 });
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
    memory_total: number;
    process_count: number;
  }[]
> {
  const rows = await querySql<GpuProcessRow>(`
    SELECT * FROM gpu_processes
    WHERE "timestamp" > dateadd('s', -15, now())
  `);
  const gpuRows = await getLatestGpuMetrics();
  const memoryTotals = new Map(gpuRows.map((gpu) => [`${gpu.hostname}|${gpu.gpu_id}`, gpu.memory_total ?? 0]));
  const latest = dedupLatestByHostGpuPid(rows).filter(r => r.container_id !== '' && r.container_id != null);
  const map = new Map<string, { container_id: string; container_name: string; hostname: string; gpu_id: string; total_memory: number; memory_total: number; process_count: number }>();
  for (const r of latest) {
    const key = `${r.container_id}|${r.hostname}|${r.gpu_id}`;
    const memoryTotal = memoryTotals.get(`${r.hostname}|${r.gpu_id}`) ?? 0;
    const existing = map.get(key);
    if (existing) {
      existing.total_memory += r.used_memory;
      existing.process_count += 1;
      existing.memory_total = Math.max(existing.memory_total, memoryTotal);
    } else {
      map.set(key, { container_id: r.container_id, container_name: r.container_name, hostname: r.hostname, gpu_id: r.gpu_id, total_memory: r.used_memory, memory_total: memoryTotal, process_count: 1 });
    }
  }
  return [...map.values()].sort((a, b) => b.total_memory - a.total_memory);
}
