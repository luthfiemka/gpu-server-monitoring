<script lang="ts">
  import { onMount } from 'svelte';
  import GpuCard from '$lib/components/GpuCard.svelte';
  import ProcessTable from '$lib/components/ProcessTable.svelte';
  import type { GpuMetricsRow, GpuProcessRow } from '$lib/server/questdb';

  let gpus = $state<GpuMetricsRow[]>([]);
  let processes = $state<GpuProcessRow[]>([]);
  let error = $state('');
  let lastUpdate = $state<Date | null>(null);
  let interval: ReturnType<typeof setInterval>;

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

  onMount(() => {
    fetchData();
    interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  });

  let totalMemory = $derived(
    gpus.reduce((sum, g) => sum + (g.memory_used ?? 0), 0)
  );

  let maxMemory = $derived(
    gpus.reduce((sum, g) => sum + (g.memory_total ?? 0), 0)
  );

  let avgTemp = $derived(
    gpus.length > 0
      ? gpus.reduce((sum, g) => sum + (g.temperature ?? 0), 0) / gpus.length
      : 0
  );

  let avgUtil = $derived(
    gpus.length > 0
      ? gpus.reduce((sum, g) => sum + (g.utilization_gpu ?? 0), 0) / gpus.length
      : 0
  );
</script>

<svelte:head>
  <title>Dashboard - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <!-- Page header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Dashboard</h1>
      <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">Real-time GPU monitoring</p>
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

  <!-- Stats row -->
  <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
    <div class="stat-card">
      <div class="stat-label">GPUs Online</div>
      <div class="stat-value" style="color: var(--tblr-primary);">{gpus.length}</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Memory Used</div>
      <div class="stat-value">
        {(totalMemory / 1024).toFixed(1)}
        <span class="text-sm font-normal" style="color: var(--tblr-muted);">/ {(maxMemory / 1024).toFixed(1)} GB</span>
      </div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Avg Temperature</div>
      <div class="stat-value {avgTemp >= 85 ? 'temp-high' : avgTemp >= 70 ? 'temp-medium' : 'temp-low'}">
        {avgTemp.toFixed(0)}<span class="text-sm font-normal" style="color: var(--tblr-muted);"> C</span>
      </div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Avg Utilization</div>
      <div class="stat-value" style="color: {avgUtil >= 80 ? 'var(--tblr-danger)' : avgUtil >= 50 ? 'var(--tblr-warning)' : 'var(--tblr-success)'};">
        {avgUtil.toFixed(0)}<span class="text-sm font-normal" style="color: var(--tblr-muted);">%</span>
      </div>
    </div>
  </div>

  <!-- GPU cards -->
  <div>
    <div class="card-header flex items-center justify-between">
      <span>GPUs</span>
      <span class="text-xs" style="color: var(--tblr-muted);">{gpus.length} devices</span>
    </div>
    <div class="card-body">
      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4">
        {#each gpus as gpu (gpu.gpu_id)}
          <GpuCard {gpu} />
        {/each}
        {#if gpus.length === 0 && !error}
          <p class="text-sm" style="color: var(--tblr-muted);">No GPU data available</p>
        {/if}
      </div>
    </div>
  </div>

  <!-- Process table -->
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
