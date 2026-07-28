<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { ArrowLeft } from '@lucide/svelte';
  import ProcessTable from '$lib/components/ProcessTable.svelte';
  import type { GpuMetricsRow, GpuProcessRow } from '$lib/server/questdb';

  const gpuId = $derived(page.params.gpuId ?? '');
  const hostname = $derived(decodeURIComponent(page.params.hostname ?? ''));

  let gpus = $state<GpuMetricsRow[]>([]);
  let processes = $state<GpuProcessRow[]>([]);
  let history = $state<GpuMetricsRow[]>([]);
  let error = $state('');
  let from = $state('');
  let to = $state('');
  let sampleBy = $state('1m');

  const now = new Date();
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
  from = oneHourAgo.toISOString().slice(0, 19);
  to = now.toISOString().slice(0, 19);

  async function fetchLiveData() {
    try {
      const res = await fetch('/api/gpu');
      const data = await res.json();
      if (!data.error) {
        gpus = data.gpus;
        processes = data.processes.filter((p: GpuProcessRow) => p.hostname === hostname && p.gpu_id === gpuId);
      }
    } catch {
      error = 'Failed to fetch GPU data';
    }
  }

  async function fetchHistory() {
    try {
      const params = new URLSearchParams({
        hostname,
        gpu_id: gpuId,
        from,
        to,
        sample_by: sampleBy
      });
      const res = await fetch(`/api/gpu/history?${params}`);
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        history = data.history;
      }
    } catch {
      error = 'Failed to fetch history';
    }
  }

  onMount(() => {
    fetchLiveData();
    fetchHistory();
    const interval = setInterval(fetchLiveData, 5000);
    return () => clearInterval(interval);
  });

  let gpu = $derived(gpus.find((g) => g.hostname === hostname && g.gpu_id === gpuId));
</script>

<svelte:head>
  <title>GPU {gpuId} ({hostname}) - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div class="flex items-center gap-3">
    <a href="/" class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-800" style="color: var(--tblr-muted);">
      <ArrowLeft class="h-5 w-5" />
    </a>
    <div>
      <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">GPU {gpuId} — {hostname}</h1>
      {#if gpu}
        <p class="text-sm" style="color: var(--tblr-muted);">{gpu.gpu_name} — {gpu.gpu_uuid?.slice(0, 12)}</p>
      {/if}
    </div>
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  {#if gpu}
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="stat-card">
        <div class="stat-label">Utilization</div>
        <div class="stat-value">{(gpu.utilization_gpu ?? 0).toFixed(0)}%</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Temperature</div>
        <div class="stat-value {(gpu.temperature ?? 0) >= 85 ? 'temp-high' : (gpu.temperature ?? 0) >= 70 ? 'temp-medium' : 'temp-low'}">{(gpu.temperature ?? 0).toFixed(0)}C</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Memory</div>
        <div class="stat-value">
          {(gpu.memory_used ?? 0).toFixed(0)}
          <span class="text-sm font-normal" style="color: var(--tblr-muted);">/ {(gpu.memory_total ?? 0).toFixed(0)} MB</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">Power</div>
        <div class="stat-value">
          {(gpu.power_draw ?? 0).toFixed(0)}
          <span class="text-sm font-normal" style="color: var(--tblr-muted);">/ {(gpu.power_limit ?? 0).toFixed(0)} W</span>
        </div>
      </div>
    </div>
  {/if}

  <!-- History -->
  <div class="card">
    <div class="card-header flex items-center justify-between">
      <span>History</span>
      <div class="flex items-center gap-2">
        <input
          type="datetime-local"
          value={from}
          onchange={(e) => { from = (e.target as HTMLInputElement).value; fetchHistory(); }}
          class="form-control text-xs"
          style="width: auto;"
        />
        <input
          type="datetime-local"
          value={to}
          onchange={(e) => { to = (e.target as HTMLInputElement).value; fetchHistory(); }}
          class="form-control text-xs"
          style="width: auto;"
        />
        <select
          bind:value={sampleBy}
          onchange={() => fetchHistory()}
          class="form-control text-xs"
          style="width: auto;"
        >
          <option value="30s">30s</option>
          <option value="1m">1m</option>
          <option value="5m">5m</option>
          <option value="15m">15m</option>
          <option value="1h">1h</option>
        </select>
      </div>
    </div>
    <div class="card-body p-0">
      {#if history.length > 0}
        <div class="overflow-x-auto">
          <table class="table">
            <thead>
              <tr>
                <th>Time</th>
                <th class="text-end">Util%</th>
                <th class="text-end">Mem MB</th>
                <th class="text-end">Temp C</th>
                <th class="text-end">Power W</th>
                <th class="text-end">Fan%</th>
              </tr>
            </thead>
            <tbody>
              {#each history as row}
                <tr>
                  <td class="font-mono text-xs">{new Date(row.timestamp).toLocaleTimeString()}</td>
                  <td class="text-end">{(row.utilization_gpu ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.memory_used ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.temperature ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.power_draw ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.fan_speed ?? 0).toFixed(0)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {:else}
        <div class="text-center py-8 text-sm" style="color: var(--tblr-muted);">No history data for selected range</div>
      {/if}
    </div>
  </div>

  <!-- Processes -->
  <div class="card">
    <div class="card-header">Processes on GPU {gpuId} ({hostname})</div>
    <div class="card-body p-0">
      <ProcessTable {processes} />
    </div>
  </div>
</div>
