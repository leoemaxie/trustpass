import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 5174,
    proxy: {
      '/api/schemas': {
        target: 'http://localhost:8081',
        rewrite: (path) => path.replace(/^\/api\/schemas/, ''),
      },
      '/api/issuer': {
        target: 'http://localhost:8082',
        rewrite: (path) => path.replace(/^\/api\/issuer/, ''),
      },
      '/api/verifier': {
        target: 'http://localhost:8083',
        rewrite: (path) => path.replace(/^\/api\/verifier/, ''),
      },
      '/api/receipts': {
        target: 'http://localhost:8084',
        rewrite: (path) => path.replace(/^\/api\/receipts/, ''),
      },
    },
  },
  build: { target: 'es2020' },
});
