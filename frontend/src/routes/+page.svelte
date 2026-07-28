<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, Cpu, Database, Server, Thermometer, Zap } from '@lucide/svelte';
  import ProcessTable from '$lib/components/ProcessTable.svelte';
  import type { GpuMetricsRow, GpuProcessRow, GpuTrendRow } from '$lib/server/questdb';

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

  type ChartMetric = 'vram' | 'power';

  type ChartLine = {
    id: string;
    label: string;
    color: string;
    path: string;
    latestX: number;
    latestY: number;
    labelY: number;
    latestValue: number;
  };

  type ChartData = {
    kind: ChartMetric;
    title: string;
    unit: string;
    lines: ChartLine[];
    maxValue: number;
    ticks: number[];
    startLabel: string;
    endLabel: string;
  };

  const WEEKLY_CHART_SAMPLE_BY = '30m';
  const WEEKLY_CHART_REFRESH_MS = 5 * 60 * 1000;
  const CHART_COLORS = ['#2563eb', '#16a34a', '#f97316', '#dc2626', '#7c3aed', '#0891b2', '#ca8a04', '#db2777'];
  const CHART = {
    width: 1040,
    height: 240,
    left: 54,
    right: 138,
    top: 22,
    bottom: 38
  };

  let gpus = $state<GpuMetricsRow[]>([]);
  let processes = $state<GpuProcessRow[]>([]);
  let chartSamples = $state<LiveSample[]>([]);
  let error = $state('');
  let chartError = $state('');
  let lastUpdate = $state<Date | null>(null);
  let chartLastUpdate = $state<Date | null>(null);
  let interval: ReturnType<typeof setInterval>;
  let trendInterval: ReturnType<typeof setInterval>;

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

  async function fetchData() {
    try {
      const res = await fetch('/api/gpu');
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        gpus = data.gpus;
        processes = data.processes;
        error = '';
        lastUpdate = new Date();
      }
    } catch {
      error = 'Failed to fetch GPU data';
    }
  }

  function mapTrendRows(rows: GpuTrendRow[]) {
    return rows
      .map((row) => ({
        timestamp: new Date(row.timestamp).getTime(),
        hostname: row.hostname,
        gpu_id: row.gpu_id,
        memory_used_gb: (row.memory_used ?? 0) / 1024,
        memory_total_gb: (row.memory_total ?? 0) / 1024,
        power_draw: row.power_draw ?? 0,
        power_limit: row.power_limit ?? 0
      }))
      .filter((sample) => Number.isFinite(sample.timestamp))
      .sort((a, b) => a.timestamp - b.timestamp || a.hostname.localeCompare(b.hostname) || compareGpuId(a.gpu_id, b.gpu_id));
  }

  async function fetchWeeklyTrend() {
    try {
      const params = new URLSearchParams({ sample_by: WEEKLY_CHART_SAMPLE_BY });
      const res = await fetch(`/api/gpu/trend?${params}`);
      const data = await res.json();
      if (data.error) {
        chartError = data.error;
      } else {
        chartSamples = mapTrendRows(data.trend ?? []);
        chartError = '';
        chartLastUpdate = new Date();
      }
    } catch {
      chartError = 'Failed to fetch weekly GPU trend';
    }
  }

  onMount(() => {
    fetchData();
    fetchWeeklyTrend();
    interval = setInterval(fetchData, 5000);
    trendInterval = setInterval(fetchWeeklyTrend, WEEKLY_CHART_REFRESH_MS);
    return () => {
      clearInterval(interval);
      clearInterval(trendInterval);
    };
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

  function niceMax(value: number) {
    if (value <= 1) return 1;
    const magnitude = 10 ** Math.floor(Math.log10(value));
    const normalized = value / magnitude;
    const nice = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
    return nice * magnitude;
  }

  function scalePoint(timestamp: number, value: number, maxValue: number, minTime: number, maxTime: number) {
    const plotWidth = CHART.width - CHART.left - CHART.right;
    const plotHeight = CHART.height - CHART.top - CHART.bottom;
    const x = maxTime === minTime
      ? CHART.left + plotWidth / 2
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

  function formatChartValue(value: number, unit: string) {
    const formatted = unit === 'GB' ? formatGb(value) : value.toFixed(0);
    return `${formatted}${unit}`;
  }

  function formatChartTime(timestamp: number) {
    return new Date(timestamp).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function distributeLabelY(lines: ChartLine[]) {
    const minY = CHART.top + 4;
    const maxY = CHART.height - CHART.bottom - 4;
    const minGap = 18;
    const sorted = lines.toSorted((a, b) => a.latestY - b.latestY);

    for (const [index, line] of sorted.entries()) {
      line.labelY = Math.max(minY, Math.min(maxY, index === 0 ? line.latestY : Math.max(line.latestY, sorted[index - 1].labelY + minGap)));
    }

    for (let index = sorted.length - 2; index >= 0; index -= 1) {
      sorted[index].labelY = Math.min(sorted[index].labelY, sorted[index + 1].labelY - minGap);
    }

    for (const line of sorted) {
      line.labelY = Math.max(minY, Math.min(maxY, line.labelY));
    }
  }

  function buildChartData(samples: LiveSample[], kind: ChartMetric): ChartData {
    const unit = kind === 'vram' ? 'GB' : 'W';
    const title = kind === 'vram' ? 'VRAM Usage' : 'Power Draw';

    if (samples.length === 0) {
      return { kind, title, unit, lines: [], maxValue: 1, ticks: [1, 0.5, 0], startLabel: '', endLabel: '' };
    }

    const times = samples.map((sample) => sample.timestamp);
    const minTime = Math.min(...times);
    const maxTime = Math.max(...times);
    const rawMax = Math.max(
      1,
      ...samples.map((sample) => kind === 'vram'
        ? sample.memory_total_gb || sample.memory_used_gb
        : sample.power_limit || sample.power_draw
      )
    );
    const maxValue = niceMax(rawMax);
    const ticks = [maxValue, maxValue * 0.75, maxValue * 0.5, maxValue * 0.25, 0];
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
      const color = CHART_COLORS[lines.length % CHART_COLORS.length];
      const points = sortedSamples.map((sample) => scalePoint(
        sample.timestamp,
        kind === 'vram' ? sample.memory_used_gb : sample.power_draw,
        maxValue,
        minTime,
        maxTime
      ));
      const label = gpuLabel(latest.hostname, latest.gpu_id);

      lines.push({
        id: `${key}-${kind}`,
        label,
        color,
        path: buildPath(points),
        latestX: points[points.length - 1].x,
        latestY: points[points.length - 1].y,
        labelY: points[points.length - 1].y,
        latestValue: kind === 'vram' ? latest.memory_used_gb : latest.power_draw
      });
    }

    distributeLabelY(lines);

    return {
      kind,
      title,
      unit,
      lines,
      maxValue,
      ticks,
      startLabel: formatChartTime(minTime),
      endLabel: formatChartTime(maxTime)
    };
  }

  let totalServers = $derived(new Set(gpus.map((gpu) => gpu.hostname)).size);
  let avgPower = $derived(gpus.length > 0 ? gpus.reduce((sum, gpu) => sum + (gpu.power_draw ?? 0), 0) / gpus.length : 0);
  let avgVram = $derived(gpus.length > 0 ? gpus.reduce((sum, gpu) => sum + (gpu.memory_used ?? 0), 0) / gpus.length / 1024 : 0);
  let totalMemory = $derived(gpus.reduce((sum, gpu) => sum + (gpu.memory_used ?? 0), 0));
  let maxMemory = $derived(gpus.reduce((sum, gpu) => sum + (gpu.memory_total ?? 0), 0));
  let serverGroups = $derived(buildServerGroups(gpus, processes));
  let chartPanels = $derived([buildChartData(chartSamples, 'vram'), buildChartData(chartSamples, 'power')]);
  let chartGpuCount = $derived(new Set(chartSamples.map(sampleKey)).size);
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
        <div>Weekly GPU Metrics Trend</div>
        <div class="text-xs font-normal mt-0.5" style="color: var(--tblr-muted);">
          {chartSamples.length > 0 ? `${chartPanels[0].startLabel} - ${chartPanels[0].endLabel}` : 'Loading 7-day history'}
          {#if chartLastUpdate}
            · Updated {chartLastUpdate.toLocaleTimeString()}
          {/if}
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2 text-xs">
        <span class="badge badge-success">{chartGpuCount} GPU{chartGpuCount === 1 ? '' : 's'}</span>
        <span class="badge" style="background: rgb(32 107 196 / 0.1); color: var(--tblr-primary);">
          7 days · {WEEKLY_CHART_SAMPLE_BY} sample
        </span>
      </div>
    </div>
    <div class="card-body">
      {#if chartError}
        <div class="mb-4 rounded border px-3 py-2 text-sm" style="border-color: var(--tblr-danger); color: var(--tblr-danger);">
          {chartError}
        </div>
      {/if}
      {#if chartPanels.some((panel) => panel.lines.length > 0)}
        <div class="grid gap-5 2xl:grid-cols-2">
          {#each chartPanels as panel (panel.kind)}
            <section class="rounded border p-4" style="border-color: var(--tblr-card-border);">
              <div class="mb-3 flex items-start justify-between gap-4">
                <div>
                  <h2 class="text-sm font-semibold">{panel.title}</h2>
                  <p class="mt-0.5 text-xs" style="color: var(--tblr-muted);">
                    7-day sampled trend by server GPU
                  </p>
                </div>
                <div class="text-right">
                  <div class="stat-label">Scale</div>
                  <div class="text-sm font-semibold">{formatChartValue(panel.maxValue, panel.unit)}</div>
                </div>
              </div>

              <div class="overflow-x-auto">
                <svg
                  viewBox="0 0 {CHART.width} {CHART.height}"
                  class="min-w-[760px] w-full"
                  role="img"
                  aria-label="{panel.title} chart for all GPUs"
                >
                  <defs>
                    <linearGradient id="chart-fill-{panel.kind}" x1="0" x2="0" y1="0" y2="1">
                      <stop offset="0%" stop-color={panel.kind === 'vram' ? '#2563eb' : '#f97316'} stop-opacity="0.12" />
                      <stop offset="100%" stop-color={panel.kind === 'vram' ? '#2563eb' : '#f97316'} stop-opacity="0" />
                    </linearGradient>
                  </defs>

                  <rect x={CHART.left} y={CHART.top} width={CHART.width - CHART.left - CHART.right} height={CHART.height - CHART.top - CHART.bottom} rx="8" fill="url(#chart-fill-{panel.kind})" />

                  {#each panel.ticks as tick}
                    {@const y = CHART.top + (1 - tick / panel.maxValue) * (CHART.height - CHART.top - CHART.bottom)}
                    <line x1={CHART.left} x2={CHART.width - CHART.right} y1={y} y2={y} stroke="var(--tblr-card-border)" stroke-width="1" />
                    <text x={CHART.left - 10} y={y + 4} text-anchor="end" class="fill-current text-[11px]" style="color: var(--tblr-muted);">
                      {formatChartValue(tick, panel.unit)}
                    </text>
                  {/each}

                  <line x1={CHART.left} x2={CHART.left} y1={CHART.top} y2={CHART.height - CHART.bottom} stroke="var(--tblr-card-border)" stroke-width="1.2" />
                  <line x1={CHART.left} x2={CHART.width - CHART.right} y1={CHART.height - CHART.bottom} y2={CHART.height - CHART.bottom} stroke="var(--tblr-card-border)" stroke-width="1.2" />

                  <text x={CHART.left} y={CHART.height - 12} class="fill-current text-[11px]" style="color: var(--tblr-muted);">
                    {panel.startLabel}
                  </text>
                  <text x={CHART.width - CHART.right} y={CHART.height - 12} text-anchor="end" class="fill-current text-[11px]" style="color: var(--tblr-muted);">
                    {panel.endLabel}
                  </text>

                  {#each panel.lines as line (line.id)}
                    <path
                      d={line.path}
                      fill="none"
                      stroke={line.color}
                      stroke-width="2.6"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      opacity="0.9"
                      vector-effect="non-scaling-stroke"
                    />
                    <line
                      x1={line.latestX + 7}
                      x2={CHART.width - CHART.right + 8}
                      y1={line.latestY}
                      y2={line.labelY}
                      stroke={line.color}
                      stroke-width="1"
                      opacity="0.35"
                      vector-effect="non-scaling-stroke"
                    />
                    <circle cx={line.latestX} cy={line.latestY} r="4" fill={line.color} stroke="var(--tblr-card-bg)" stroke-width="2" />
                    <text x={CHART.width - CHART.right + 12} y={line.labelY + 4} class="fill-current text-[10px]">
                      {line.label} · {formatChartValue(line.latestValue, panel.unit)}
                    </text>
                  {/each}
                </svg>
              </div>
            </section>
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
      <span class="text-xs" style="color: var(--tblr-muted);">{processes.length} processes · highest VRAM first</span>
    </div>
    <div class="card-body p-0">
      <ProcessTable {processes} paginated sortByMemoryDesc pageSize={10} />
    </div>
  </div>
</div>
