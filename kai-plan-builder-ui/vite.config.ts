import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5175,
    proxy: {
      '/api/v1/plan-builder': { target: 'http://localhost:8083', changeOrigin: true },
      '/api': { target: 'http://localhost:8080', changeOrigin: true }
    }
  },
  preview: {
    port: 4175,
    proxy: {
      '/api/v1/plan-builder': { target: 'http://localhost:8083', changeOrigin: true },
      '/api': { target: 'http://localhost:8080', changeOrigin: true }
    }
  },
  build: {
    outDir: 'dist'
  }
})
