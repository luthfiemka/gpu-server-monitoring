<script lang="ts">
  import type { GpuProcessRow } from '$lib/server/questdb';

  let { processes }: { processes: GpuProcessRow[] } = $props();
</script>

<div class="overflow-x-auto">
  <table class="table">
    <thead>
      <tr>
        <th>Server</th>
        <th>GPU</th>
        <th>PID</th>
        <th>Process</th>
        <th>User</th>
        <th>Container</th>
        <th class="text-end">VRAM</th>
      </tr>
    </thead>
    <tbody>
      {#each processes as proc}
        <tr>
          <td class="font-mono text-xs">{proc.hostname}</td>
          <td>
            <a href="/gpus/{encodeURIComponent(proc.hostname)}/{proc.gpu_id}" class="font-medium" style="color: var(--tblr-primary);">
              {proc.gpu_id}
            </a>
          </td>
          <td class="font-mono text-xs">{proc.pid}</td>
          <td>{proc.process_name}</td>
          <td>
            <span class="badge badge-success">{proc.username}</span>
          </td>
          <td class="font-mono text-xs">
            {#if proc.container_id}
              {proc.container_id}
            {:else}
              <span style="color: var(--tblr-muted);">-</span>
            {/if}
          </td>
          <td class="text-end font-medium">{(proc.used_memory ?? 0).toFixed(0)} MB</td>
        </tr>
      {/each}
      {#if processes.length === 0}
        <tr>
          <td colspan="7" class="text-center py-8" style="color: var(--tblr-muted);">
            No running GPU processes
          </td>
        </tr>
      {/if}
    </tbody>
  </table>
</div>
