import { writable } from 'svelte/store';
import { browser } from '$app/environment';

type ThemeMode = 'system' | 'light' | 'dark';
type AppliedTheme = 'light' | 'dark';

function getSystemTheme(): AppliedTheme {
  return browser && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(mode: ThemeMode) {
  if (!browser) return getSystemTheme();

  const applied = mode === 'system' ? getSystemTheme() : mode;
  document.documentElement.classList.toggle('dark', applied === 'dark');

  if (mode === 'system') {
    localStorage.removeItem('theme-mode');
    localStorage.removeItem('theme');
  } else {
    localStorage.setItem('theme-mode', mode);
  }

  return applied;
}

function createThemeStore() {
  const stored = browser ? localStorage.getItem('theme-mode') : null;
  const initial: ThemeMode = stored === 'dark' || stored === 'light' ? stored : 'system';

  const { subscribe, set, update } = writable<ThemeMode>(initial);

  if (browser) {
    const media = window.matchMedia('(prefers-color-scheme: dark)');

    subscribe((mode) => {
      applyTheme(mode);
    });
    applyTheme(initial);

    media.addEventListener('change', () => {
      update((mode) => {
        if (mode === 'system') applyTheme(mode);
        return mode;
      });
    });
  }

  return {
    subscribe,
    toggle: () => update((mode) => {
      const applied = applyTheme(mode);
      return applied === 'dark' ? 'light' : 'dark';
    }),
    useSystem: () => set('system')
  };
}

export const theme = createThemeStore();
