import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    proxy: {
      '/api/questdb': {
        target: 'http://localhost:9000',
        rewrite: (path) => path.replace(/^\/api\/questdb/, ''),
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq) => {
            proxyReq.setHeader('Authorization', 'Basic ' + Buffer.from('admin:quest').toString('base64'));
          });
        }
      }
    }
  }
});
