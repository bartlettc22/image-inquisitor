import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In dev, /api is proxied to the daemon (override with API_UPSTREAM)
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    proxy: {
      '/api': process.env.API_UPSTREAM ?? 'http://localhost:8080',
    },
  },
});
