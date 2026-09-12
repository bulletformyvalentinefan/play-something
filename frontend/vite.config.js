import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Go proxy (go-librespot) - auth/proxy/player -> :8081
      '/api/v1/spotify/auth': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api/v1/spotify/proxy': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api/v1/spotify/player': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})