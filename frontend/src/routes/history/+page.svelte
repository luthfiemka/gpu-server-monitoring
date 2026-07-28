<script lang="ts">
  import { onMount } from 'svelte';
  import { Clock } from '@lucide/svelte';
  import type { GpuMetricsRow } from '$lib/server/questdb';

  let history = $state<GpuMetricsRow[]>([]);
  let error = $state('');
  let loading = $state(false);

  const now = new Date();
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);

  let from = $state(oneHourAgo.toISOString().slice(0, 16));
  let to = $state(now.toISOString().slice(0, 16));
  let sampleBy = $state('5m');

  async function fetchHistory() {
    loading = true;
    error = '';
    try {
      const params = new URLSearchParams({
        from: from + ':00',
        to: to + ':00',
        sample_by: sampleBy
      });
      const res = await fetch(`/api/history?${params}`);
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        history = data.history;
      }
    } catch {
      error = 'Failed to fetch history';
    } finally {
      loading = false;
    }
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    fetchHistory();
  }

  onMount(fetchHistory);
</script>

<svelte:head>
  <title>History - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div>
    <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">History</h1>
    <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">Query historical GPU metrics</p>
  </div>

  <div class="card">
    <div class="card-body">
      <form onsubmit={handleSubmit} class="flex flex-wrap items-end gap-4">
        <div>
          <label class="form-label">From</label>
          <input
            type="datetime-local"
            bind:value={from}
            class="form-control"
          />
        </div>
        <div>
          <label class="form-label">To</label>
          <input
            type="datetime-local"
            bind:value={to}
            class="form-control"
          />
        </div>
        <div>
          <label class="form-label">Sample By</label>
          <select bind:value={sampleBy} class="form-control">
            <option value="1m">1 minute</option>
            <option value="5m">5 minutes</option>
            <option value="15m">15 minutes</option>
            <option value="1h">1 hour</option>
            <option value="1d">1 day</option>
          </select>
        </div>
        <button
          type="submit"
          disabled={loading}
          class="btn btn-primary"
        >
          {loading ? 'Loading...' : 'Query'}
        </button>
      </form>
    </div>
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  {#if history.length > 0}
    <div class="card">
      <div class="card-header flex items-center justify-between">
        <span>Results</span>
        <span class="text-xs" style="color: var(--tblr-muted);">{history.length} data points</span>
      </div>
      <div class="card-body p-0">
        <div class="overflow-x-auto">
          <table class="table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Server</th>
                <th>GPU</th>
                <th>Name</th>
                <th class="text-end">Util%</th>
                <th class="text-end">Mem Used</th>
                <th class="text-end">Mem Total</th>
                <th class="text-end">Temp C</th>
                <th class="text-end">Power W</th>
                <th class="text-end">Fan%</th>
              </tr>
            </thead>
            <tbody>
              {#each history as row}
                <tr>
                  <td class="font-mono text-xs">{new Date(row.timestamp).toLocaleString()}</td>
                  <td class="font-mono text-xs">{row.hostname}</td>
                  <td>
                    <a href="/gpus/{encodeURIComponent(row.hostname)}/{row.gpu_id}" style="color: var(--tblr-primary);">{row.gpu_id}</a>
                  </td>
                  <td style="color: var(--tblr-muted);">{row.gpu_name}</td>
                  <td class="text-end">{(row.utilization_gpu ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.memory_used ?? 0).toFixed(0)} MB</td>
                  <td class="text-end">{(row.memory_total ?? 0).toFixed(0)} MB</td>
                  <td class="text-end">{(row.temperature ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.power_draw ?? 0).toFixed(0)}</td>
                  <td class="text-end">{(row.fan_speed ?? 0).toFixed(0)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  {:else if !loading && !error}
    <div class="card">
      <div class="card-body text-center py-8" style="color: var(--tblr-muted);">
        No data for selected range
      </div>
    </div>
  {/if}
</div>
