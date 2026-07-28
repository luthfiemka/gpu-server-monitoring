<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  import type { LayoutData } from './$types';
  import { theme } from '$lib/stores/theme';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import {
    LayoutDashboard,
    Users,
    Container,
    Clock,
    Sun,
    Moon,
    LogOut,
    Cpu
  } from '@lucide/svelte';

  let { data, children }: { data: LayoutData; children: Snippet } = $props();

  const navItems = [
    { href: '/', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/users', label: 'Users', icon: Users },
    { href: '/containers', label: 'Containers', icon: Container },
    { href: '/history', label: 'History', icon: Clock }
  ];

  async function logout() {
    await fetch('/api/auth', { method: 'DELETE' });
    goto('/login');
  }

  let showMobileMenu = $state(false);
</script>

{#if !data.user || page.url.pathname === '/login'}
  {@render children()}
{:else}
  <div class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="px-4 py-3 flex items-center gap-2.5 border-b border-white/10">
        <div class="w-8 h-8 rounded bg-blue-600 flex items-center justify-center">
          <Cpu class="h-5 w-5 text-white" />
        </div>
        <span class="font-semibold text-white text-sm">GPU Dashboard</span>
      </div>

      <nav class="sidebar-nav">
        {#each navItems as item}
          {@const active = page.url.pathname === item.href}
          <a
            href={item.href}
            class="sidebar-link"
            class:active
          >
            <item.icon class="h-5 w-5" />
            {item.label}
          </a>
        {/each}
      </nav>

      <div class="px-2 py-3 border-t border-white/10">
        <button
          onclick={() => theme.toggle()}
          class="sidebar-link w-full"
        >
          {#if $theme === 'dark'}
            <Sun class="h-5 w-5" />
            Light Mode
          {:else}
            <Moon class="h-5 w-5" />
            Dark Mode
          {/if}
        </button>
        <button
          onclick={logout}
          class="sidebar-link w-full"
        >
          <LogOut class="h-5 w-5" />
          Logout
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <div class="flex flex-col flex-1 overflow-hidden">
      <!-- Navbar -->
      <header class="navbar flex items-center justify-between px-4">
        <div class="flex items-center gap-3">
          <button
            onclick={() => (showMobileMenu = !showMobileMenu)}
            class="md:hidden p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-800"
          >
            <LayoutDashboard class="h-5 w-5" />
          </button>
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {data.user?.username || 'User'}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs text-gray-400">v0.1.0</span>
        </div>
      </header>

      <!-- Mobile menu -->
      {#if showMobileMenu}
        <div class="md:hidden border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-2 py-1">
          {#each navItems as item}
            {@const active = page.url.pathname === item.href}
            <a
              href={item.href}
              onclick={() => (showMobileMenu = false)}
              class="sidebar-link"
              class:active
            >
              <item.icon class="h-5 w-5" />
              {item.label}
            </a>
          {/each}
          <button
            onclick={logout}
            class="sidebar-link w-full"
          >
            <LogOut class="h-5 w-5" />
            Logout
          </button>
        </div>
      {/if}

      <main class="flex-1 overflow-y-auto p-6">
        {@render children()}
      </main>
    </div>
  </div>
{/if}
