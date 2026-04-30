import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    proxy: {
      '/health': 'http://localhost:8000',
      '/login': 'http://localhost:8000',
      '/me': 'http://localhost:8000',
      '/leads': 'http://localhost:8000',
    },
  },
  plugins: [
    tailwindcss(),
    react(),
  ],
})
