import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function createThemeStore() {
  const stored = browser ? localStorage.getItem('theme') : null;
  const prefersDark = browser && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const initial = stored ?? (prefersDark ? 'dark' : 'light');

  const { subscribe, set, update } = writable<string>(initial);

  if (browser) {
    subscribe((v) => {
      localStorage.setItem('theme', v);
      document.documentElement.classList.toggle('dark', v === 'dark');
    });
    document.documentElement.classList.toggle('dark', initial === 'dark');
  }

  return {
    subscribe,
    toggle: () => update((v) => (v === 'dark' ? 'light' : 'dark'))
  };
}

export const theme = createThemeStore();
