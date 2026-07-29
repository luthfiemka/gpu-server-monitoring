<script lang="ts">
  import { onMount } from 'svelte';
  import { Settings, Users, Save, Plus, Trash2, Image, Cpu } from '@lucide/svelte';

  type Tab = 'general' | 'users';

  interface AppUser {
    username: string;
    display_name: string;
    role: string;
    created_at: string;
  }

  let activeTab: Tab = $state('general');
  let saving = $state(false);
  let error = $state('');
  let success = $state('');

  // Brand settings
  let brandName = $state('GPU Dashboard');
  let logoUrl = $state('');
  let logoPreview = $state('');

  // Users
  let users = $state<AppUser[]>([]);
  let showUserForm = $state(false);
  let editingUser = $state<string | null>(null);
  let userForm = $state({ username: '', password: '', display_name: '' });

  onMount(() => {
    loadSettings();
    loadUsers();
  });

  async function loadSettings() {
    try {
      const res = await fetch('/api/settings');
      if (!res.ok) return;
      const data = await res.json();
      brandName = data.brand_name || 'GPU Dashboard';
      logoUrl = data.logo_url || '';
      logoPreview = logoUrl;
    } catch {}
  }

  async function loadUsers() {
    try {
      const res = await fetch('/api/users/manage');
      if (!res.ok) return;
      users = await res.json();
    } catch {}
  }

  async function saveSettings() {
    saving = true;
    error = '';
    success = '';
    try {
      const res = await fetch('/api/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ brand_name: brandName, logo_url: logoUrl })
      });
      if (!res.ok) {
        const data = await res.json();
        error = data.error || 'Failed to save';
      } else {
        logoPreview = logoUrl;
        success = 'Settings saved';
        setTimeout(() => success = '', 3000);
      }
    } catch {
      error = 'Network error';
    } finally {
      saving = false;
    }
  }

  function openAddUser() {
    editingUser = null;
    userForm = { username: '', password: '', display_name: '' };
    showUserForm = true;
  }

  function openEditUser(u: AppUser) {
    editingUser = u.username;
    userForm = { username: u.username, password: '', display_name: u.display_name };
    showUserForm = true;
  }

  async function submitUser() {
    error = '';
    success = '';
    if (editingUser) {
      const body: Record<string, string> = { username: editingUser, display_name: userForm.display_name };
      if (userForm.password) body.password = userForm.password;
      const res = await fetch('/api/users/manage', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      if (!res.ok) {
        const data = await res.json();
        error = data.error || 'Failed to update';
        return;
      }
    } else {
      if (!userForm.username || !userForm.password) {
        error = 'Username and password required';
        return;
      }
      const res = await fetch('/api/users/manage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(userForm)
      });
      if (!res.ok) {
        const data = await res.json();
        error = data.error || 'Failed to create';
        return;
      }
    }
    showUserForm = false;
    success = editingUser ? 'User updated' : 'User created';
    setTimeout(() => success = '', 3000);
    loadUsers();
  }

  async function deleteUser(username: string) {
    if (!confirm(`Delete user "${username}"?`)) return;
    error = '';
    success = '';
    const res = await fetch('/api/users/manage', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username })
    });
    if (!res.ok) {
      const data = await res.json();
      error = data.error || 'Failed to delete';
      return;
    }
    success = `User "${username}" deleted`;
    setTimeout(() => success = '', 3000);
    loadUsers();
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString();
  }
</script>

<svelte:head>
  <title>Settings - GPU Dashboard</title>
</svelte:head>

<div class="space-y-6">
  <div>
    <h1 class="text-xl font-bold" style="color: var(--tblr-body-color);">Settings</h1>
    <p class="text-sm mt-0.5" style="color: var(--tblr-muted);">Manage brand and users</p>
  </div>

  {#if error}
    <div class="card" style="border-color: var(--tblr-danger);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-danger);">{error}</div>
    </div>
  {/if}

  {#if success}
    <div class="card" style="border-color: var(--tblr-success);">
      <div class="card-body py-3 text-sm" style="color: var(--tblr-success);">{success}</div>
    </div>
  {/if}

  <!-- Tabs -->
  <div class="flex gap-1 border-b" style="border-color: var(--tblr-card-border);">
    <button
      onclick={() => activeTab = 'general'}
      class="btn btn-ghost rounded-none border-0 pb-3 text-sm"
      class:active-tab={activeTab === 'general'}
      style={activeTab === 'general' ? 'color: var(--tblr-primary); border-bottom: 2px solid var(--tblr-primary); margin-bottom: -1px;' : ''}
    >
      <Settings class="h-4 w-4" />
      General
    </button>
    <button
      onclick={() => activeTab = 'users'}
      class="btn btn-ghost rounded-none border-0 pb-3 text-sm"
      class:active-tab={activeTab === 'users'}
      style={activeTab === 'users' ? 'color: var(--tblr-primary); border-bottom: 2px solid var(--tblr-primary); margin-bottom: -1px;' : ''}
    >
      <Users class="h-4 w-4" />
      Users
    </button>
  </div>

  {#if activeTab === 'general'}
    <div class="card">
      <div class="card-header">Brand</div>
      <div class="card-body space-y-4">
        <div>
          <label for="brand-name" class="form-label">Brand Name</label>
          <input
            id="brand-name"
            type="text"
            bind:value={brandName}
            class="form-control"
            placeholder="GPU Dashboard"
          />
        </div>
        <div>
          <label for="logo-url" class="form-label">Logo URL</label>
          <input
            id="logo-url"
            type="url"
            bind:value={logoUrl}
            class="form-control"
            placeholder="https://example.com/logo.png"
          />
          <p class="text-xs mt-1" style="color: var(--tblr-muted);">URL gambar untuk sidebar. Kosongkan untuk icon default.</p>
        </div>
        <div>
          <span class="form-label">Preview</span>
          <div class="flex items-center gap-3 p-3 rounded" style="background: var(--tblr-sidebar-bg); max-width: 300px;">
            {#if logoPreview}
              <img src={logoPreview} alt="Logo" class="w-8 h-8 rounded object-contain" style="background: var(--tblr-card-bg);" onerror={() => logoPreview = ''} />
            {:else}
              <div class="w-8 h-8 rounded bg-blue-600 flex items-center justify-center">
                <Cpu class="h-5 w-5 text-white" />
              </div>
            {/if}
            <span class="text-sm font-semibold text-white">{brandName || 'GPU Dashboard'}</span>
          </div>
        </div>
        <div class="flex justify-end">
          <button onclick={saveSettings} disabled={saving} class="btn btn-primary">
            <Save class="h-4 w-4" />
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  {/if}

  {#if activeTab === 'users'}
    <div class="card">
      <div class="card-header flex items-center justify-between">
        <span>Users ({users.length})</span>
        <button onclick={openAddUser} class="btn btn-primary btn-sm flex items-center gap-1.5" style="padding: 0.375rem 0.75rem; font-size: 0.8125rem;">
          <Plus class="h-3.5 w-3.5" />
          Add User
        </button>
      </div>
      <div class="card-body p-0">
        {#if users.length === 0}
          <div class="text-center py-8 text-sm" style="color: var(--tblr-muted);">No users yet</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="table">
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Display Name</th>
                  <th>Role</th>
                  <th>Created</th>
                  <th class="text-end">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each users as u (u.username)}
                  <tr>
                    <td class="font-mono text-xs">{u.username}</td>
                    <td>{u.display_name}</td>
                    <td><span class="badge badge-success">{u.role}</span></td>
                    <td class="text-xs" style="color: var(--tblr-muted);">{formatDate(u.created_at)}</td>
                    <td class="text-end">
                      <button onclick={() => openEditUser(u)} class="btn btn-ghost btn-sm" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem;">
                        Edit
                      </button>
                      <button onclick={() => deleteUser(u.username)} class="btn btn-ghost btn-sm" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem; color: var(--tblr-danger);">
                        <Trash2 class="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>

    <!-- User form modal -->
    {#if showUserForm}
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
        <div class="card w-full max-w-md mx-4">
          <div class="card-header">{editingUser ? 'Edit User' : 'Add User'}</div>
          <div class="card-body space-y-4">
            <div>
              <label for="uf-username" class="form-label">Username</label>
              <input
                id="uf-username"
                type="text"
                bind:value={userForm.username}
                class="form-control"
                disabled={!!editingUser}
                placeholder="username"
              />
            </div>
            <div>
              <label for="uf-display" class="form-label">Display Name</label>
              <input
                id="uf-display"
                type="text"
                bind:value={userForm.display_name}
                class="form-control"
                placeholder="Display Name"
              />
            </div>
            <div>
              <label for="uf-password" class="form-label">{editingUser ? 'New Password (leave blank to keep)' : 'Password'}</label>
              <input
                id="uf-password"
                type="password"
                bind:value={userForm.password}
                class="form-control"
                placeholder={editingUser ? 'Leave blank to keep current' : 'Password'}
              />
            </div>
            <div class="flex justify-end gap-2">
              <button onclick={() => showUserForm = false} class="btn btn-ghost">Cancel</button>
              <button onclick={submitUser} class="btn btn-primary">
                <Save class="h-4 w-4" />
                {editingUser ? 'Update' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .btn-sm {
    padding: 0.25rem 0.5rem;
    font-size: 0.8125rem;
  }
</style>
