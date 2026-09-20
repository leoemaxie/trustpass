import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 5173,
    proxy: {
      '/api/issuer': {
        target: 'http://localhost:8082',
        rewrite: (path) => path.replace(/^\/api\/issuer/, ''),
      },
      '/api/verifier': {
        target: 'http://localhost:8083',
        rewrite: (path) => path.replace(/^\/api\/verifier/, ''),
      },
    },
  },
  build: {
    target: 'es2020',
  },
});
