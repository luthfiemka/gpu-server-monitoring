<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  import type { LayoutData } from './$types';
  import { browser } from '$app/environment';
  import { theme } from '$lib/stores/theme';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import {
    LayoutDashboard,
    Users,
    Container,
    Clock,
    Settings,
    Sun,
    Moon,
    Monitor,
    PanelLeftClose,
    PanelLeftOpen,
    LogOut,
    Cpu
  } from '@lucide/svelte';

  interface BrandData { logo_url: string; brand_name: string; }
  let { data, children }: { data: LayoutData & { brand?: BrandData }; children: Snippet } = $props();

  const navItems = [
    { href: '/', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/users', label: 'Users', icon: Users },
    { href: '/containers', label: 'Containers', icon: Container },
    { href: '/history', label: 'History', icon: Clock },
    { href: '/settings', label: 'Settings', icon: Settings }
  ];

  async function logout() {
    await fetch('/api/auth', { method: 'DELETE' });
    goto('/login');
  }

  let showMobileMenu = $state(false);
  let sidebarCollapsed = $state(false);

  if (browser) {
    sidebarCollapsed = localStorage.getItem('sidebar-collapsed') === 'true';
  }

  $effect(() => {
    if (browser) {
      localStorage.setItem('sidebar-collapsed', String(sidebarCollapsed));
    }
  });
</script>

{#if !data.user || page.url.pathname === '/login'}
  {@render children()}
{:else}
  <div class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <aside class="sidebar" class:collapsed={sidebarCollapsed}>
      <div class="sidebar-brand border-b border-white/10">
        {#if data.brand?.logo_url}
          <img src={data.brand!.logo_url} alt="Logo" class="sidebar-brand-icon w-8 h-8 rounded object-contain" style="background: var(--tblr-card-bg);" onerror={() => { if (data.brand) data.brand.logo_url = ''; }} />
        {:else}
          <div class="sidebar-brand-icon w-8 h-8 rounded bg-blue-600 flex items-center justify-center">
            <Cpu class="h-5 w-5 text-white" />
          </div>
        {/if}
        <span class="sidebar-label font-semibold text-white text-sm">{data.brand?.brand_name || 'GPU Dashboard'}</span>
        <button
          type="button"
          onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
          class="sidebar-collapse-btn"
          title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {#if sidebarCollapsed}
            <PanelLeftOpen class="h-4 w-4" />
          {:else}
            <PanelLeftClose class="h-4 w-4" />
          {/if}
        </button>
      </div>

      <nav class="sidebar-nav">
        {#each navItems as item}
          {@const active = page.url.pathname === item.href}
          <a
            href={item.href}
            class="sidebar-link"
            class:active
            title={item.label}
          >
            <item.icon class="sidebar-link-icon h-5 w-5" />
            <span class="sidebar-label">{item.label}</span>
          </a>
        {/each}
      </nav>

      <div class="px-2 py-3 border-t border-white/10">
        <button
          onclick={() => theme.toggle()}
          class="sidebar-link w-full"
          title={$theme === 'system' ? 'System theme' : $theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {#if $theme === 'system'}
            <Monitor class="sidebar-link-icon h-5 w-5" />
            <span class="sidebar-label">System Theme</span>
          {:else if $theme === 'dark'}
            <Sun class="h-5 w-5" />
            <span class="sidebar-label">Light Mode</span>
          {:else}
            <Moon class="h-5 w-5" />
            <span class="sidebar-label">Dark Mode</span>
          {/if}
        </button>
        {#if $theme !== 'system'}
          <button
            onclick={() => theme.useSystem()}
            class="sidebar-link w-full"
            title="Follow system theme"
          >
            <Monitor class="sidebar-link-icon h-5 w-5" />
            <span class="sidebar-label">Auto Theme</span>
          </button>
        {/if}
        <button
          onclick={logout}
          class="sidebar-link w-full"
          title="Logout"
        >
          <LogOut class="sidebar-link-icon h-5 w-5" />
          <span class="sidebar-label">Logout</span>
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
