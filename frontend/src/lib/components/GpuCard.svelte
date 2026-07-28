<script lang="ts">
  import type { GpuMetricsRow } from '$lib/server/questdb';

  let { gpu }: { gpu: GpuMetricsRow } = $props();

  let memPercent = $derived(
    gpu.memory_total > 0 ? (gpu.memory_used / gpu.memory_total) * 100 : 0
  );

  let tempClass = $derived(
    (gpu.temperature ?? 0) >= 85 ? 'temp-high' : (gpu.temperature ?? 0) >= 70 ? 'temp-medium' : 'temp-low'
  );

  let utilClass = $derived(
    (gpu.utilization_gpu ?? 0) >= 80 ? 'util-bar-high' : (gpu.utilization_gpu ?? 0) >= 50 ? 'util-bar-medium' : 'util-bar-low'
  );

  let memBadge = $derived(
    memPercent >= 90 ? 'badge-danger' : memPercent >= 70 ? 'badge-warning' : 'badge-success'
  );
</script>

<a
  href="/gpus/{encodeURIComponent(gpu.hostname)}/{gpu.gpu_id}"
  class="block card hover:shadow-md transition-shadow"
>
  <div class="card-body">
    <!-- Header -->
    <div class="flex items-start justify-between mb-3">
      <div>
        <div class="flex items-center gap-2">
          <span class="font-semibold" style="color: var(--tblr-body-color);">GPU {gpu.gpu_id}</span>
          <span class="badge {memBadge}">{memPercent.toFixed(0)}%</span>
        </div>
        <div class="text-xs mt-0.5" style="color: var(--tblr-muted);">{gpu.gpu_name}</div>
        <div class="text-xs font-mono mt-0.5" style="color: var(--tblr-muted);">{gpu.hostname}</div>
      </div>
      <span class="text-xs font-mono" style="color: var(--tblr-muted);">{gpu.gpu_uuid?.slice(0, 8)}</span>
    </div>

    <!-- Stats grid -->
    <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm mb-3">
      <div class="flex justify-between">
        <span style="color: var(--tblr-muted);">Util</span>
        <span class="font-medium">{(gpu.utilization_gpu ?? 0).toFixed(0)}%</span>
      </div>
      <div class="flex justify-between">
        <span style="color: var(--tblr-muted);">Temp</span>
        <span class="font-medium {tempClass}">{(gpu.temperature ?? 0).toFixed(0)}C</span>
      </div>
      <div class="flex justify-between">
        <span style="color: var(--tblr-muted);">Memory</span>
        <span class="font-medium">{(gpu.memory_used ?? 0).toFixed(0)} MB</span>
      </div>
      <div class="flex justify-between">
        <span style="color: var(--tblr-muted);">Power</span>
        <span class="font-medium">{(gpu.power_draw ?? 0).toFixed(0)} W</span>
      </div>
    </div>

    <!-- Utilization bar -->
    <div class="mb-2">
      <div class="flex justify-between text-xs mb-1">
        <span style="color: var(--tblr-muted);">GPU Utilization</span>
        <span>{(gpu.utilization_gpu ?? 0).toFixed(0)}%</span>
      </div>
      <div class="progress">
        <div class="progress-bar {utilClass}" style="width: {gpu.utilization_gpu ?? 0}%"></div>
      </div>
    </div>

    <!-- Memory bar -->
    <div>
      <div class="flex justify-between text-xs mb-1">
        <span style="color: var(--tblr-muted);">Memory</span>
        <span>{(gpu.memory_used ?? 0).toFixed(0)} / {(gpu.memory_total ?? 0).toFixed(0)} MB</span>
      </div>
      <div class="progress">
        <div class="progress-bar {memPercent >= 90 ? 'util-bar-high' : memPercent >= 70 ? 'util-bar-medium' : 'util-bar-low'}" style="width: {memPercent}%"></div>
      </div>
    </div>
  </div>
</a>
