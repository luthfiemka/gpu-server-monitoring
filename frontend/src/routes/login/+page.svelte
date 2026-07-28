<script lang="ts">
  import { goto } from '$app/navigation';
  import { Cpu } from '@lucide/svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      const res = await fetch('/api/auth', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });

      const data = await res.json();

      if (data.ok) {
        goto('/');
      } else {
        error = data.error || 'Login failed';
      }
    } catch {
      error = 'Network error';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen flex items-center justify-center px-4" style="background: var(--tblr-body-bg);">
  <div class="w-full max-w-sm">
    <div class="text-center mb-6">
      <div class="w-12 h-12 rounded-lg bg-blue-600 flex items-center justify-center mx-auto mb-4">
        <Cpu class="h-7 w-7 text-white" />
      </div>
      <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">GPU Dashboard</h1>
      <p class="text-sm mt-1" style="color: var(--tblr-muted);">Sign in to your account</p>
    </div>

    <form onsubmit={handleSubmit} class="card">
      <div class="card-body space-y-4">
        <div>
          <label for="username" class="form-label">Username</label>
          <input
            id="username"
            type="text"
            bind:value={username}
            required
            class="form-control"
            placeholder="admin"
          />
        </div>

        <div>
          <label for="password" class="form-label">Password</label>
          <input
            id="password"
            type="password"
            bind:value={password}
            required
            class="form-control"
            placeholder="password"
          />
        </div>

        {#if error}
          <div class="badge badge-danger" style="padding: 0.375rem 0.75rem;">
            {error}
          </div>
        {/if}

        <button
          type="submit"
          disabled={loading}
          class="btn btn-primary w-full"
        >
          {loading ? 'Signing in...' : 'Sign in'}
        </button>
      </div>
    </form>
  </div>
</div>
