<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, Database, Gauge, Users } from '@lucide/svelte';

  interface UserSummary {
    username: string;
    hostname: string;
    gpu_id: string;
    total_memory: number;
    memory_total: number;
    process_count: number;
  }

  interface UserGroup {
    username: string;
    gpus: UserSummary[];
    totalMemory: number;
    totalCapacity: number;
    processCount: number;
    percentOfUsed: number;
    capacityPercent: number;
  }

  interface DonutSegment {
    username: string;
    memory: number;
    percent: number;
    color: string;
    dashArray: string;
    dashOffset: number;
  }

  const COLORS = ['#2563eb', '#16a34a', '#f97316', '#dc2626', '#7c3aed', '#0891b2', '#ca8a04', '#db2777'];
  const DONUT_RADIUS = 44;
  const DONUT_CIRCUMFERENCE = 2 * Math.PI * DONUT_RADIUS;

  let users = $state<UserSummary[]>([]);
  let error = $state('');
  let lastUpdate = $state<Date | null>(null);

  async function fetchUsers() {
    try {
      const res = await fetch('/api/users');
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        users = data.users;
        error = '';
        lastUpdate = new Date();
      }
    } catch {
      error = 'Failed to fetch user data';
    }
  }

  onMount(fetchUsers);

  function formatMemory(mb: number) {
    const gb = mb / 1024;
    return gb >= 10 ? `${gb.toFixed(0)} GB` : `${gb.toFixed(1)} GB`;
  }

  function safePercent(value: number, total: number) {
    return total > 0 ? Math.min(100, Math.max(0, (value / total) * 100)) : 0;
  }

  function buildUserGroups(rows: UserSummary[]): UserGroup[] {
    const groups = new Map<string, UserSummary[]>();
    for (const row of rows) {
      const list = groups.get(row.username) ?? [];
      list.push(row);
      groups.set(row.username, list);
    }

    const totalUsed = rows.reduce((sum, row) => sum + (row.total_memory ?? 0), 0);

    return [...groups.entries()]
      .map(([username, gpus]) => {
        const sortedGpus = gpus.toSorted(
          (a, b) =>
            (b.total_memory ?? 0) - (a.total_memory ?? 0) ||
            a.hostname.localeCompare(b.hostname) ||
            a.gpu_id.localeCompare(b.gpu_id)
        );
        const totalMemory = sortedGpus.reduce((sum, row) => sum + (row.total_memory ?? 0), 0);
        const totalCapacity = sortedGpus.reduce((sum, row) => sum + (row.memory_total ?? 0), 0);
        const processCount = sortedGpus.reduce((sum, row) => sum + (row.process_count ?? 0), 0);

        return {
          username,
          gpus: sortedGpus,
          totalMemory,
          totalCapacity,
          processCount,
          percentOfUsed: safePercent(totalMemory, totalUsed),
          capacityPercent: safePercent(totalMemory, totalCapacity)
        };
      })
      .sort((a, b) => b.totalMemory - a.totalMemory || a.username.localeCompare(b.username));
  }

  function buildDonutSegments(groups: UserGroup[]): DonutSegment[] {
    const total = groups.reduce((sum, group) => sum + group.totalMemory, 0);
    const visibleGroups = groups.length > 8
      ? [
          ...groups.slice(0, 7),
          {
            username: 'Others',
            gpus: [],
            totalMemory: groups.slice(7).reduce((sum, group) => sum + group.totalMemory, 0),
            totalCapacity: 0,
            processCount: groups.slice(7).reduce((sum, group) => sum + group.processCount, 0),
            percentOfUsed: groups.slice(7).reduce((sum, group) => sum + group.percentOfUsed, 0),
            capacityPercent: 0
          }
        ]
      : groups;
    let offset = 0;

    return visibleGroups.map((group, index) => {
      const percent = safePercent(group.totalMemory, total);
      const length = total > 0 ? (percent / 100) * DONUT_CIRCUMFERENCE : 0;
      const segment = {
        username: group.username,
        memory: group.totalMemory,
        percent,
        color: COLORS[index % COLORS.length],
        dashArray: `${length} ${DONUT_CIRCUMFERENCE - length}`,
        dashOffset: -offset
      };
      offset += length;
      return segment;
    });
  }

  function sumDistinctGpuCapacity(rows: UserSummary[]) {
    const totals = new Map<string, number>();
    for (const row of rows) {
      const key = `${row.hostname}|${row.gpu_id}`;
      totals.set(key, Math.max(totals.get(key) ?? 0, row.memory_total ?? 0));
    }
    return [...totals.values()].reduce((sum, value) => sum + value, 0);
  }

  let userGroups = $derived(buildUserGroups(users));
  let totalUsedMemory = $derived(users.reduce((sum, row) => sum + (row.total_memory ?? 0), 0));
  let totalCapacity = $derived(sumDistinctGpuCapacity(users));
  let totalProcesses = $derived(users.reduce((sum, row) => sum + (row.process_count ?? 0), 0));
  let topUser = $derived(userGroups[0]);
  let donutSegments = $derived(buildDonutSegments(userGroups));
</script>

<svelte:head>
  <title>Users - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Users</h1>
      <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">GPU usage grouped by Linux user</p>
    </div>
    {#if lastUpdate}
      <span class="text-xs" style="color: var(--tblr-muted);">Updated {lastUpdate.toLocaleTimeString()}</span>
    {/if}
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  {#if userGroups.length > 0}
    <div class="grid gap-4 xl:grid-cols-[minmax(320px,420px)_1fr]">
      <section class="card">
        <div class="card-header flex items-center justify-between">
          <span>VRAM Leaders</span>
          <span class="text-xs" style="color: var(--tblr-muted);">{userGroups.length} users</span>
        </div>
        <div class="card-body">
          <div class="flex flex-col items-center gap-5 sm:flex-row xl:flex-col 2xl:flex-row">
            <div class="relative h-52 w-52 shrink-0">
              <svg viewBox="0 0 120 120" class="h-full w-full -rotate-90" role="img" aria-label="VRAM usage by user donut chart">
                <circle cx="60" cy="60" r={DONUT_RADIUS} fill="none" stroke="var(--tblr-card-border)" stroke-width="16" />
                {#each donutSegments as segment (segment.username)}
                  <circle
                    cx="60"
                    cy="60"
                    r={DONUT_RADIUS}
                    fill="none"
                    stroke={segment.color}
                    stroke-width="16"
                    stroke-linecap="round"
                    stroke-dasharray={segment.dashArray}
                    stroke-dashoffset={segment.dashOffset}
                  />
                {/each}
              </svg>
              <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
                <span class="text-xs uppercase" style="color: var(--tblr-muted);">Top user</span>
                <span class="mt-1 max-w-32 truncate text-xl font-bold">{topUser.username}</span>
                <span class="text-xs" style="color: var(--tblr-muted);">{topUser.percentOfUsed.toFixed(0)}% of used</span>
              </div>
            </div>

            <div class="w-full space-y-3">
              {#each donutSegments.slice(0, 5) as segment}
                <div>
                  <div class="mb-1 flex items-center justify-between gap-3 text-xs">
                    <span class="inline-flex min-w-0 items-center gap-2">
                      <span class="h-2.5 w-2.5 shrink-0 rounded-full" style="background: {segment.color};"></span>
                      <span class="truncate font-medium">{segment.username}</span>
                    </span>
                    <span style="color: var(--tblr-muted);">{segment.percent.toFixed(1)}%</span>
                  </div>
                  <div class="progress">
                    <div class="progress-bar" style="width: {segment.percent}%; background: {segment.color};"></div>
                  </div>
                  <div class="mt-1 text-xs" style="color: var(--tblr-muted);">{formatMemory(segment.memory)}</div>
                </div>
              {/each}
            </div>
          </div>
        </div>
      </section>

      <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div class="stat-card">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="stat-label">Active Users</div>
              <div class="stat-value">{userGroups.length}</div>
            </div>
            <Users class="h-5 w-5" style="color: var(--tblr-primary);" />
          </div>
        </div>
        <div class="stat-card">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="stat-label">VRAM Used</div>
              <div class="stat-value">{formatMemory(totalUsedMemory)}</div>
            </div>
            <Database class="h-5 w-5" style="color: #7c3aed;" />
          </div>
        </div>
        <div class="stat-card">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="stat-label">VRAM %</div>
              <div class="stat-value">{safePercent(totalUsedMemory, totalCapacity).toFixed(0)}%</div>
            </div>
            <Gauge class="h-5 w-5" style="color: var(--tblr-success);" />
          </div>
        </div>
        <div class="stat-card">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="stat-label">Processes</div>
              <div class="stat-value">{totalProcesses}</div>
            </div>
            <Activity class="h-5 w-5" style="color: var(--tblr-warning);" />
          </div>
        </div>
      </section>
    </div>

    <div class="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
      {#each userGroups as group, index (group.username)}
        <section class="card">
          <div class="card-header flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="h-2.5 w-2.5 shrink-0 rounded-full" style="background: {COLORS[index % COLORS.length]};"></span>
                <span class="truncate font-semibold">{group.username}</span>
              </div>
              <div class="mt-0.5 text-xs font-normal" style="color: var(--tblr-muted);">
                {group.gpus.length} GPU{group.gpus.length === 1 ? '' : 's'} · {group.processCount} process{group.processCount === 1 ? '' : 'es'}
              </div>
            </div>
            <div class="text-right">
              <div class="text-base font-bold">{formatMemory(group.totalMemory)}</div>
              <div class="text-xs" style="color: var(--tblr-muted);">{group.percentOfUsed.toFixed(1)}% of active VRAM</div>
            </div>
          </div>
          <div class="card-body space-y-4">
            <div>
              <div class="mb-1 flex justify-between text-xs">
                <span style="color: var(--tblr-muted);">User VRAM allocation</span>
                <span>{group.capacityPercent.toFixed(1)}%</span>
              </div>
              <div class="progress">
                <div
                  class="progress-bar {group.capacityPercent >= 90 ? 'util-bar-high' : group.capacityPercent >= 70 ? 'util-bar-medium' : 'util-bar-low'}"
                  style="width: {group.capacityPercent}%"
                ></div>
              </div>
            </div>

            <div class="overflow-x-auto">
              <table class="table">
                <thead>
                  <tr>
                    <th>Server</th>
                    <th>GPU</th>
                    <th class="text-end">VRAM</th>
                    <th class="text-end">%</th>
                  </tr>
                </thead>
                <tbody>
                  {#each group.gpus as gpu}
                    {@const gpuPercent = safePercent(gpu.total_memory, gpu.memory_total)}
                    <tr>
                      <td class="font-mono text-xs">{gpu.hostname}</td>
                      <td>
                        <a href="/gpus/{encodeURIComponent(gpu.hostname)}/{gpu.gpu_id}" style="color: var(--tblr-primary);">GPU {gpu.gpu_id}</a>
                      </td>
                      <td class="text-end">{formatMemory(gpu.total_memory)}</td>
                      <td class="text-end">
                        <span class="badge {gpuPercent >= 90 ? 'badge-danger' : gpuPercent >= 70 ? 'badge-warning' : 'badge-success'}">
                          {gpuPercent.toFixed(1)}%
                        </span>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      {/each}
    </div>
  {:else if !error}
    <div class="card">
      <div class="card-body text-center py-8" style="color: var(--tblr-muted);">
        No GPU processes running
      </div>
    </div>
  {/if}
</div>
