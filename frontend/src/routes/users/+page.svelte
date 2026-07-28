<script lang="ts">
  import { onMount } from 'svelte';
  import { Users } from '@lucide/svelte';

  interface UserSummary {
    username: string;
    hostname: string;
    gpu_id: string;
    total_memory: number;
    process_count: number;
  }

  let users = $state<UserSummary[]>([]);
  let error = $state('');

  async function fetchUsers() {
    try {
      const res = await fetch('/api/users');
      const data = await res.json();
      if (data.error) {
        error = data.error;
      } else {
        users = data.users;
      }
    } catch {
      error = 'Failed to fetch user data';
    }
  }

  onMount(fetchUsers);

  let groupedUsers = $derived(
    users.reduce(
      (acc, u) => {
        if (!acc[u.username]) acc[u.username] = [];
        acc[u.username].push(u);
        return acc;
      },
      {} as Record<string, UserSummary[]>
    )
  );
</script>

<svelte:head>
  <title>Users - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div>
    <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Users</h1>
    <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">GPU usage grouped by Linux user</p>
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  <div class="space-y-4">
    {#each Object.entries(groupedUsers) as [username, gpuList]}
      <div class="card">
        <div class="card-header flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="badge badge-success">{username}</span>
          </div>
          <span class="text-xs" style="color: var(--tblr-muted);">
            {gpuList.reduce((s, u) => s + u.process_count, 0)} processes
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
              {#each gpuList as g}
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

    {#if Object.keys(groupedUsers).length === 0 && !error}
      <div class="card">
        <div class="card-body text-center py-8" style="color: var(--tblr-muted);">
          No GPU processes running
        </div>
      </div>
    {/if}
  </div>
</div>
