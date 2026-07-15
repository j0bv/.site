import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, '.'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // Same-origin dev ergonomics: frontend calls /api/*, Vite proxies to Axum.
      '/api': 'http://127.0.0.1:3001',
    },
  },
})

