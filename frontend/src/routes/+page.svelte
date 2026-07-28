<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, Cpu, Database, Server, Thermometer, Zap } from '@lucide/svelte';
  import ProcessTable from '$lib/components/ProcessTable.svelte';
  import type { GpuMetricsRow, GpuProcessRow } from '$lib/server/questdb';

  type LiveSample = {
    timestamp: number;
    hostname: string;
    gpu_id: string;
    memory_used_gb: number;
    memory_total_gb: number;
    power_draw: number;
    power_limit: number;
  };

  type ServerGroup = {
    hostname: string;
    gpus: GpuMetricsRow[];
    processes: GpuProcessRow[];
    avgPower: number;
    avgVram: number;
    memoryUsed: number;
    memoryTotal: number;
    avgUtil: number;
    avgTemp: number;
  };

  type ChartLine = {
    id: string;
    label: string;
    kind: 'vram' | 'power';
    color: string;
    path: string;
    latestX: number;
    latestY: number;
    latestValue: number;
  };

  type ChartData = {
    lines: ChartLine[];
    vramMax: number;
    powerMax: number;
    startLabel: string;
    endLabel: string;
  };

  const MAX_CHART_POINTS = 36;
  const CHART_COLORS = ['#2563eb', '#16a34a', '#f97316', '#dc2626', '#7c3aed', '#0891b2', '#ca8a04', '#db2777'];
  const CHART = {
    width: 1000,
    height: 320,
    left: 64,
    right: 72,
    top: 28,
    bottom: 58
  };

  let gpus = $state<GpuMetricsRow[]>([]);
  let processes = $state<GpuProcessRow[]>([]);
  let chartSamples = $state<LiveSample[]>([]);
  let error = $state('');
  let lastUpdate = $state<Date | null>(null);
  let interval: ReturnType<typeof setInterval>;

  function gpuKey(gpu: Pick<GpuMetricsRow, 'hostname' | 'gpu_id'>) {
    return `${gpu.hostname}|${gpu.gpu_id}`;
  }

  function sampleKey(sample: Pick<LiveSample, 'hostname' | 'gpu_id'>) {
    return `${sample.hostname}|${sample.gpu_id}`;
  }

  function gpuLabel(hostname: string, gpuId: string) {
    return `${hostname}-gpu-${gpuId}`;
  }

  function compareGpuId(a: string, b: string) {
    const aNum = Number(a);
    const bNum = Number(b);
    if (Number.isFinite(aNum) && Number.isFinite(bNum)) return aNum - bNum;
    return a.localeCompare(b);
  }

  function formatGb(value: number) {
    return value >= 10 ? value.toFixed(0) : value.toFixed(1);
  }

  function percent(value: number, total: number) {
    return total > 0 ? Math.min(100, Math.max(0, (value / total) * 100)) : 0;
  }

  function updateChartSamples(rows: GpuMetricsRow[]) {
    const timestamp = Date.now();
    const activeKeys = new Set(rows.map(gpuKey));
    const grouped = new Map<string, LiveSample[]>();

    for (const sample of chartSamples) {
      const key = sampleKey(sample);
      if (!activeKeys.has(key)) continue;
      const samples = grouped.get(key) ?? [];
      samples.push(sample);
      grouped.set(key, samples);
    }

    for (const gpu of rows) {
      const key = gpuKey(gpu);
      const samples = grouped.get(key) ?? [];
      samples.push({
        timestamp,
        hostname: gpu.hostname,
        gpu_id: gpu.gpu_id,
        memory_used_gb: (gpu.memory_used ?? 0) / 1024,
        memory_total_gb: (gpu.memory_total ?? 0) / 1024,
        power_draw: gpu.power_draw ?? 0,
        power_limit: gpu.power_limit ?? 0
      });
      grouped.set(key, samples.slice(-MAX_CHART_POINTS));
    }

    chartSamples = [...grouped.values()].flat();
  }

  async function fetchData() {
    try {
      const res = await fetch('/api/gpu');
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        gpus = data.gpus;
        processes = data.processes;
        updateChartSamples(data.gpus);
        error = '';
        lastUpdate = new Date();
      }
    } catch {
      error = 'Failed to fetch GPU data';
    }
  }

  onMount(() => {
    fetchData();
    interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  });

  function buildServerGroups(rows: GpuMetricsRow[], processRows: GpuProcessRow[]): ServerGroup[] {
    const map = new Map<string, GpuMetricsRow[]>();
    for (const gpu of rows) {
      const hostGpus = map.get(gpu.hostname) ?? [];
      hostGpus.push(gpu);
      map.set(gpu.hostname, hostGpus);
    }

    return [...map.entries()]
      .map(([hostname, hostGpus]) => {
        const sortedGpus = hostGpus.toSorted((a, b) => compareGpuId(a.gpu_id, b.gpu_id));
        const hostProcesses = processRows.filter((process) => process.hostname === hostname);
        const memoryUsed = sortedGpus.reduce((sum, gpu) => sum + (gpu.memory_used ?? 0), 0);
        const memoryTotal = sortedGpus.reduce((sum, gpu) => sum + (gpu.memory_total ?? 0), 0);

        return {
          hostname,
          gpus: sortedGpus,
          processes: hostProcesses,
          avgPower: sortedGpus.length > 0 ? sortedGpus.reduce((sum, gpu) => sum + (gpu.power_draw ?? 0), 0) / sortedGpus.length : 0,
          avgVram: sortedGpus.length > 0 ? memoryUsed / sortedGpus.length / 1024 : 0,
          memoryUsed,
          memoryTotal,
          avgUtil: sortedGpus.length > 0 ? sortedGpus.reduce((sum, gpu) => sum + (gpu.utilization_gpu ?? 0), 0) / sortedGpus.length : 0,
          avgTemp: sortedGpus.length > 0 ? sortedGpus.reduce((sum, gpu) => sum + (gpu.temperature ?? 0), 0) / sortedGpus.length : 0
        };
      })
      .sort((a, b) => a.hostname.localeCompare(b.hostname));
  }

  function scalePoint(timestamp: number, value: number, maxValue: number, minTime: number, maxTime: number) {
    const plotWidth = CHART.width - CHART.left - CHART.right;
    const plotHeight = CHART.height - CHART.top - CHART.bottom;
    const x = maxTime === minTime
      ? CHART.left + plotWidth
      : CHART.left + ((timestamp - minTime) / (maxTime - minTime)) * plotWidth;
    const y = CHART.top + plotHeight - (Math.min(value, maxValue) / maxValue) * plotHeight;
    return { x, y };
  }

  function buildPath(points: { x: number; y: number }[]) {
    if (points.length === 0) return '';
    if (points.length === 1) {
      const point = points[0];
      return `M ${point.x - 3} ${point.y} L ${point.x + 3} ${point.y}`;
    }
    return points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x.toFixed(1)} ${point.y.toFixed(1)}`).join(' ');
  }

  function buildChartData(samples: LiveSample[]): ChartData {
    if (samples.length === 0) {
      return { lines: [], vramMax: 1, powerMax: 1, startLabel: '', endLabel: '' };
    }

    const times = samples.map((sample) => sample.timestamp);
    const minTime = Math.min(...times);
    const maxTime = Math.max(...times);
    const vramMax = Math.max(1, ...samples.map((sample) => sample.memory_total_gb || sample.memory_used_gb));
    const powerMax = Math.max(1, ...samples.map((sample) => sample.power_limit || sample.power_draw));
    const grouped = new Map<string, LiveSample[]>();

    for (const sample of samples) {
      const key = sampleKey(sample);
      const values = grouped.get(key) ?? [];
      values.push(sample);
      grouped.set(key, values);
    }

    const lines: ChartLine[] = [];
    const sortedGroups = [...grouped.entries()].sort(([a], [b]) => a.localeCompare(b));

    for (const [key, groupSamples] of sortedGroups) {
      const sortedSamples = groupSamples.toSorted((a, b) => a.timestamp - b.timestamp);
      const latest = sortedSamples[sortedSamples.length - 1];
      const color = CHART_COLORS[lines.length / 2 % CHART_COLORS.length | 0];
      const vramPoints = sortedSamples.map((sample) => scalePoint(sample.timestamp, sample.memory_used_gb, vramMax, minTime, maxTime));
      const powerPoints = sortedSamples.map((sample) => scalePoint(sample.timestamp, sample.power_draw, powerMax, minTime, maxTime));
      const label = gpuLabel(latest.hostname, latest.gpu_id);

      lines.push({
        id: `${key}-vram`,
        label,
        kind: 'vram',
        color,
        path: buildPath(vramPoints),
        latestX: vramPoints[vramPoints.length - 1].x,
        latestY: vramPoints[vramPoints.length - 1].y,
        latestValue: latest.memory_used_gb
      });
      lines.push({
        id: `${key}-power`,
        label,
        kind: 'power',
        color,
        path: buildPath(powerPoints),
        latestX: powerPoints[powerPoints.length - 1].x,
        latestY: powerPoints[powerPoints.length - 1].y,
        latestValue: latest.power_draw
      });
    }

    return {
      lines,
      vramMax,
      powerMax,
      startLabel: new Date(minTime).toLocaleTimeString(),
      endLabel: new Date(maxTime).toLocaleTimeString()
    };
  }

  let totalServers = $derived(new Set(gpus.map((gpu) => gpu.hostname)).size);
  let avgPower = $derived(gpus.length > 0 ? gpus.reduce((sum, gpu) => sum + (gpu.power_draw ?? 0), 0) / gpus.length : 0);
  let avgVram = $derived(gpus.length > 0 ? gpus.reduce((sum, gpu) => sum + (gpu.memory_used ?? 0), 0) / gpus.length / 1024 : 0);
  let totalMemory = $derived(gpus.reduce((sum, gpu) => sum + (gpu.memory_used ?? 0), 0));
  let maxMemory = $derived(gpus.reduce((sum, gpu) => sum + (gpu.memory_total ?? 0), 0));
  let serverGroups = $derived(buildServerGroups(gpus, processes));
  let chartData = $derived(buildChartData(chartSamples));
</script>

<svelte:head>
  <title>Dashboard - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Dashboard</h1>
      <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">Real-time GPU monitoring by server</p>
    </div>
    {#if lastUpdate}
      <span class="text-xs" style="color: var(--tblr-muted);">
        Updated {lastUpdate.toLocaleTimeString()}
      </span>
    {/if}
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">
        {error}
      </div>
    </div>
  {/if}

  <div class="grid grid-cols-2 gap-4 xl:grid-cols-4">
    <div class="stat-card">
      <div class="flex items-start justify-between gap-3">
        <div>
          <div class="stat-label">Total GPU</div>
          <div class="stat-value" style="color: var(--tblr-primary);">{gpus.length}</div>
        </div>
        <Cpu class="h-5 w-5" style="color: var(--tblr-primary);" />
      </div>
    </div>
    <div class="stat-card">
      <div class="flex items-start justify-between gap-3">
        <div>
          <div class="stat-label">Total Server</div>
          <div class="stat-value">{totalServers}</div>
        </div>
        <Server class="h-5 w-5" style="color: var(--tblr-success);" />
      </div>
    </div>
    <div class="stat-card">
      <div class="flex items-start justify-between gap-3">
        <div>
          <div class="stat-label">Avg Power</div>
          <div class="stat-value">
            {avgPower.toFixed(0)}
            <span class="text-sm font-normal" style="color: var(--tblr-muted);">W</span>
          </div>
        </div>
        <Zap class="h-5 w-5" style="color: var(--tblr-warning);" />
      </div>
    </div>
    <div class="stat-card">
      <div class="flex items-start justify-between gap-3">
        <div>
          <div class="stat-label">Avg VRAM</div>
          <div class="stat-value">
            {formatGb(avgVram)}
            <span class="text-sm font-normal" style="color: var(--tblr-muted);">GB</span>
          </div>
        </div>
        <Database class="h-5 w-5" style="color: #7c3aed;" />
      </div>
    </div>
  </div>

  <div class="card">
    <div class="card-header flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <div>VRAM and Power Trend</div>
        <div class="text-xs font-normal mt-0.5" style="color: var(--tblr-muted);">
          {chartSamples.length > 0 ? `${chartData.startLabel} - ${chartData.endLabel}` : 'Waiting for live samples'}
        </div>
      </div>
      <div class="flex items-center gap-4 text-xs" style="color: var(--tblr-muted);">
        <span class="inline-flex items-center gap-1.5"><span class="h-0.5 w-5 rounded bg-blue-600"></span>VRAM</span>
        <span class="inline-flex items-center gap-1.5"><span class="h-0.5 w-5 border-t-2 border-dashed border-blue-600"></span>Power</span>
      </div>
    </div>
    <div class="card-body">
      {#if chartData.lines.length > 0}
        <div class="overflow-x-auto">
          <svg
            viewBox="0 0 {CHART.width} {CHART.height}"
            class="min-w-[760px] w-full"
            role="img"
            aria-label="Live VRAM and power chart for all GPUs"
          >
            <rect x="0" y="0" width={CHART.width} height={CHART.height} rx="8" fill="transparent" />
            {#each [0, 1, 2, 3] as tick}
              {@const y = CHART.top + ((CHART.height - CHART.top - CHART.bottom) / 3) * tick}
              <line x1={CHART.left} x2={CHART.width - CHART.right} y1={y} y2={y} stroke="var(--tblr-card-border)" stroke-width="1" />
            {/each}
            <line x1={CHART.left} x2={CHART.left} y1={CHART.top} y2={CHART.height - CHART.bottom} stroke="var(--tblr-muted)" stroke-width="1" opacity="0.55" />
            <line x1={CHART.left} x2={CHART.width - CHART.right} y1={CHART.height - CHART.bottom} y2={CHART.height - CHART.bottom} stroke="var(--tblr-muted)" stroke-width="1" opacity="0.55" />

            <text x={CHART.left - 12} y={CHART.top + 4} text-anchor="end" class="fill-current text-[22px]" style="color: var(--tblr-muted);">
              {formatGb(chartData.vramMax)}GB
            </text>
            <text x={CHART.width - CHART.right + 12} y={CHART.top + 4} class="fill-current text-[22px]" style="color: var(--tblr-muted);">
              {chartData.powerMax.toFixed(0)}W
            </text>
            <text x={CHART.left} y={CHART.height - 18} class="fill-current text-[22px]" style="color: var(--tblr-muted);">
              {chartData.startLabel}
            </text>
            <text x={CHART.width - CHART.right} y={CHART.height - 18} text-anchor="end" class="fill-current text-[22px]" style="color: var(--tblr-muted);">
              {chartData.endLabel}
            </text>

            {#each chartData.lines as line (line.id)}
              <path
                d={line.path}
                fill="none"
                stroke={line.color}
                stroke-width={line.kind === 'vram' ? 3 : 2}
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-dasharray={line.kind === 'power' ? '7 7' : undefined}
                opacity={line.kind === 'vram' ? 0.92 : 0.72}
              />
              <circle cx={line.latestX} cy={line.latestY} r={line.kind === 'vram' ? 4 : 3} fill={line.color} opacity={line.kind === 'vram' ? 1 : 0.8} />
            {/each}
          </svg>
        </div>
        <div class="mt-4 flex flex-wrap gap-x-4 gap-y-2 text-xs">
          {#each serverGroups.flatMap((server) => server.gpus) as gpu (gpuKey(gpu))}
            {@const color = CHART_COLORS[serverGroups.flatMap((server) => server.gpus).findIndex((item) => gpuKey(item) === gpuKey(gpu)) % CHART_COLORS.length]}
            <a href="/gpus/{encodeURIComponent(gpu.hostname)}/{gpu.gpu_id}" class="inline-flex items-center gap-1.5" style="color: var(--tblr-muted);">
              <span class="h-2.5 w-2.5 rounded-full" style="background: {color};"></span>
              {gpuLabel(gpu.hostname, gpu.gpu_id)}
            </a>
          {/each}
        </div>
      {:else}
        <div class="flex h-64 items-center justify-center text-sm" style="color: var(--tblr-muted);">
          No GPU data available for chart
        </div>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 gap-4 xl:grid-cols-2 2xl:grid-cols-3">
    {#each serverGroups as server (server.hostname)}
      <section class="card">
        <div class="card-header flex items-center justify-between">
          <div>
            <div>{server.hostname}</div>
            <div class="text-xs font-normal mt-0.5" style="color: var(--tblr-muted);">
              {server.gpus.length} GPU{server.gpus.length === 1 ? '' : 's'} · {server.processes.length} processes
            </div>
          </div>
          <span class="badge {server.avgUtil >= 80 ? 'badge-danger' : server.avgUtil >= 50 ? 'badge-warning' : 'badge-success'}">
            {server.avgUtil.toFixed(0)}% util
          </span>
        </div>
        <div class="card-body space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div class="rounded border p-3" style="border-color: var(--tblr-card-border);">
              <div class="stat-label">Avg Power</div>
              <div class="mt-1 text-lg font-semibold">{server.avgPower.toFixed(0)} W</div>
            </div>
            <div class="rounded border p-3" style="border-color: var(--tblr-card-border);">
              <div class="stat-label">Avg VRAM</div>
              <div class="mt-1 text-lg font-semibold">{formatGb(server.avgVram)} GB</div>
            </div>
            <div class="rounded border p-3" style="border-color: var(--tblr-card-border);">
              <div class="stat-label">Memory Used</div>
              <div class="mt-1 text-lg font-semibold">
                {formatGb(server.memoryUsed / 1024)}
                <span class="text-xs font-normal" style="color: var(--tblr-muted);">/ {formatGb(server.memoryTotal / 1024)} GB</span>
              </div>
            </div>
            <div class="rounded border p-3" style="border-color: var(--tblr-card-border);">
              <div class="stat-label">Avg Temp</div>
              <div class="mt-1 text-lg font-semibold {server.avgTemp >= 85 ? 'temp-high' : server.avgTemp >= 70 ? 'temp-medium' : 'temp-low'}">
                {server.avgTemp.toFixed(0)}C
              </div>
            </div>
          </div>

          <div class="divide-y" style="border-color: var(--tblr-card-border);">
            {#each server.gpus as gpu (gpu.gpu_id)}
              {@const memPercent = percent(gpu.memory_used ?? 0, gpu.memory_total ?? 0)}
              <a
                href="/gpus/{encodeURIComponent(gpu.hostname)}/{gpu.gpu_id}"
                class="block py-4 first:pt-0 last:pb-0"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-semibold">GPU {gpu.gpu_id}</span>
                      <span class="badge {memPercent >= 90 ? 'badge-danger' : memPercent >= 70 ? 'badge-warning' : 'badge-success'}">
                        {memPercent.toFixed(0)}% VRAM
                      </span>
                    </div>
                    <div class="mt-0.5 truncate text-xs" style="color: var(--tblr-muted);">{gpu.gpu_name}</div>
                  </div>
                  <div class="text-right text-xs" style="color: var(--tblr-muted);">
                    <div class="font-mono">{gpu.gpu_uuid?.slice(0, 8)}</div>
                    <div>{(gpu.power_draw ?? 0).toFixed(0)} W</div>
                  </div>
                </div>

                <div class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                  <div class="flex items-center justify-between gap-2">
                    <span class="inline-flex items-center gap-1.5" style="color: var(--tblr-muted);">
                      <Activity class="h-3.5 w-3.5" /> Util
                    </span>
                    <span class="font-medium">{(gpu.utilization_gpu ?? 0).toFixed(0)}%</span>
                  </div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="inline-flex items-center gap-1.5" style="color: var(--tblr-muted);">
                      <Thermometer class="h-3.5 w-3.5" /> Temp
                    </span>
                    <span class="font-medium {(gpu.temperature ?? 0) >= 85 ? 'temp-high' : (gpu.temperature ?? 0) >= 70 ? 'temp-medium' : 'temp-low'}">
                      {(gpu.temperature ?? 0).toFixed(0)}C
                    </span>
                  </div>
                </div>

                <div class="mt-3 space-y-2">
                  <div>
                    <div class="mb-1 flex justify-between text-xs">
                      <span style="color: var(--tblr-muted);">VRAM</span>
                      <span>{formatGb((gpu.memory_used ?? 0) / 1024)} / {formatGb((gpu.memory_total ?? 0) / 1024)} GB</span>
                    </div>
                    <div class="progress">
                      <div class="progress-bar {memPercent >= 90 ? 'util-bar-high' : memPercent >= 70 ? 'util-bar-medium' : 'util-bar-low'}" style="width: {memPercent}%"></div>
                    </div>
                  </div>
                  <div>
                    <div class="mb-1 flex justify-between text-xs">
                      <span style="color: var(--tblr-muted);">Power</span>
                      <span>{(gpu.power_draw ?? 0).toFixed(0)} / {(gpu.power_limit ?? 0).toFixed(0)} W</span>
                    </div>
                    <div class="progress">
                      <div
                        class="progress-bar {percent(gpu.power_draw ?? 0, gpu.power_limit ?? 0) >= 90 ? 'util-bar-high' : percent(gpu.power_draw ?? 0, gpu.power_limit ?? 0) >= 70 ? 'util-bar-medium' : 'util-bar-low'}"
                        style="width: {percent(gpu.power_draw ?? 0, gpu.power_limit ?? 0)}%"
                      ></div>
                    </div>
                  </div>
                </div>
              </a>
            {/each}
          </div>
        </div>
      </section>
    {/each}

    {#if serverGroups.length === 0 && !error}
      <div class="card">
        <div class="card-body text-sm" style="color: var(--tblr-muted);">No GPU data available</div>
      </div>
    {/if}
  </div>

  <div class="card">
    <div class="card-header flex items-center justify-between">
      <span>Running Processes</span>
      <span class="text-xs" style="color: var(--tblr-muted);">{processes.length} processes</span>
    </div>
    <div class="card-body p-0">
      <ProcessTable {processes} />
    </div>
  </div>
</div>
