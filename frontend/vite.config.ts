import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8989',
      '/signalr': { target: 'http://localhost:8989', ws: true },
      '/ping': 'http://localhost:8989',
    },
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
    sourcemap: true,
  },
})
