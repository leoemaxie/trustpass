import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 5174,
    proxy: {
      '/api/issuer':   'http://localhost:8081',
      '/api/verifier': 'http://localhost:8082',
      '/api/receipts': 'http://localhost:8083',
      '/api/schemas':  'http://localhost:8084',
    },
  },
  build: { target: 'es2020' },
});
