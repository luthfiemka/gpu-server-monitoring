<script lang="ts">
  import { ChevronLeft, ChevronRight } from '@lucide/svelte';
  import type { GpuProcessRow } from '$lib/server/questdb';

  let {
    processes,
    paginated = false,
    pageSize = 10,
    sortByMemoryDesc = false
  }: {
    processes: GpuProcessRow[];
    paginated?: boolean;
    pageSize?: number;
    sortByMemoryDesc?: boolean;
  } = $props();

  let currentPage = $state(1);

  let sortedProcesses = $derived(
    sortByMemoryDesc
      ? processes.toSorted((a, b) =>
          (b.used_memory ?? 0) - (a.used_memory ?? 0) ||
          a.hostname.localeCompare(b.hostname) ||
          a.gpu_id.localeCompare(b.gpu_id) ||
          a.pid - b.pid
        )
      : processes
  );

  let totalPages = $derived(paginated ? Math.max(1, Math.ceil(sortedProcesses.length / pageSize)) : 1);
  let visibleProcesses = $derived(
    paginated
      ? sortedProcesses.slice((currentPage - 1) * pageSize, currentPage * pageSize)
      : sortedProcesses
  );
  let rangeStart = $derived(sortedProcesses.length === 0 ? 0 : (currentPage - 1) * pageSize + 1);
  let rangeEnd = $derived(Math.min(currentPage * pageSize, sortedProcesses.length));

  $effect(() => {
    if (currentPage > totalPages) {
      currentPage = totalPages;
    }
  });

  function formatOptionalMemory(value: number | null | undefined) {
    return value == null || Number.isNaN(value) ? '-' : `${value.toFixed(0)} MB`;
  }
</script>

<div>
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
          <th class="text-end">Mem Alloc</th>
          <th class="text-end">Shared</th>
        </tr>
      </thead>
      <tbody>
        {#each visibleProcesses as proc}
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
            <td class="text-end font-medium">{formatOptionalMemory(proc.used_memory)}</td>
            <td class="text-end">{formatOptionalMemory(proc.mem_alloc ?? proc.used_memory)}</td>
            <td class="text-end">{formatOptionalMemory(proc.shared_memory)}</td>
          </tr>
        {/each}
        {#if sortedProcesses.length === 0}
          <tr>
            <td colspan="9" class="text-center py-8" style="color: var(--tblr-muted);">
              No running GPU processes
            </td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>

  {#if paginated && sortedProcesses.length > pageSize}
    <div class="flex flex-col gap-3 border-t px-4 py-3 sm:flex-row sm:items-center sm:justify-between" style="border-color: var(--tblr-card-border);">
      <div class="text-xs" style="color: var(--tblr-muted);">
        Showing {rangeStart}-{rangeEnd} of {sortedProcesses.length} processes
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="btn btn-ghost px-2 py-1"
          disabled={currentPage === 1}
          onclick={() => (currentPage = Math.max(1, currentPage - 1))}
          aria-label="Previous page"
          title="Previous page"
        >
          <ChevronLeft class="h-4 w-4" />
        </button>
        <span class="text-xs" style="color: var(--tblr-muted);">
          Page {currentPage} / {totalPages}
        </span>
        <button
          type="button"
          class="btn btn-ghost px-2 py-1"
          disabled={currentPage === totalPages}
          onclick={() => (currentPage = Math.min(totalPages, currentPage + 1))}
          aria-label="Next page"
          title="Next page"
        >
          <ChevronRight class="h-4 w-4" />
        </button>
      </div>
    </div>
  {/if}
</div>
