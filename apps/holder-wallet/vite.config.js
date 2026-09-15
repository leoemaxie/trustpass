import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 5173,
    proxy: {
      '/api/issuer':   'http://localhost:8081',
      '/api/verifier': 'http://localhost:8082',
    },
  },
  build: {
    target: 'es2020',
  },
});
