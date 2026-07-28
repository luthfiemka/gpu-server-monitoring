<script lang="ts">
  import { onMount } from 'svelte';
  import { Container } from '@lucide/svelte';

  interface ContainerSummary {
    container_id: string;
    container_name: string;
    hostname: string;
    gpu_id: string;
    total_memory: number;
    process_count: number;
  }

  let containers = $state<ContainerSummary[]>([]);
  let error = $state('');

  async function fetchContainers() {
    try {
      const res = await fetch('/api/containers');
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        containers = data.containers;
      }
    } catch {
      error = 'Failed to fetch container data';
    }
  }

  onMount(fetchContainers);

  let groupedContainers = $derived(
    containers.reduce(
      (acc, c) => {
        if (!acc[c.container_id]) acc[c.container_id] = { info: c, gpus: [] };
        acc[c.container_id].gpus.push(c);
        return acc;
      },
      {} as Record<string, { info: ContainerSummary; gpus: ContainerSummary[] }>
    )
  );
</script>

<svelte:head>
  <title>Containers - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div>
    <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Containers</h1>
    <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">GPU usage grouped by Docker container</p>
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  <div class="space-y-4">
    {#each Object.entries(groupedContainers) as [cid, data]}
      <div class="card">
        <div class="card-header flex items-center justify-between">
          <div>
            <span class="font-mono text-sm">{cid}</span>
            <span class="text-xs ml-2" style="color: var(--tblr-muted);">{data.info.container_name}</span>
          </div>
          <span class="text-xs" style="color: var(--tblr-muted);">
            {data.gpus.reduce((s, g) => s + g.process_count, 0)} processes
          </span>
        </div>
        <div class="card-body p-0">
          <table class="table">
            <thead>
              <tr>
                <th>Server</th>
                <th>GPU</th>
                <th class="text-end">VRAM Used</th>
                <th class="text-end">Processes</th>
              </tr>
            </thead>
            <tbody>
              {#each data.gpus as g}
                <tr>
                  <td class="font-mono text-xs">{g.hostname}</td>
                  <td>
                    <a href="/gpus/{encodeURIComponent(g.hostname)}/{g.gpu_id}" style="color: var(--tblr-primary);">GPU {g.gpu_id}</a>
                  </td>
                  <td class="text-end">{g.total_memory.toFixed(0)} MB</td>
                  <td class="text-end">{g.process_count}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/each}

    {#if Object.keys(groupedContainers).length === 0 && !error}
      <div class="card">
        <div class="card-body text-center py-8" style="color: var(--tblr-muted);">
          No containers using GPU resources
        </div>
      </div>
    {/if}
  </div>
</div>
