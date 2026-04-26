import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:18880',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://127.0.0.1:18880',
        changeOrigin: true,
      },
      '/readyz': {
        target: 'http://127.0.0.1:18880',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
