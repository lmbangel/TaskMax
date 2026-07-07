import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Single-page Wails app: build to frontend/dist which main.go embeds.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true
  },
  // Fixed dev-server port so Wails can connect directly (see wails.json
  // "frontend:dev:serverUrl"). This avoids the "timed out waiting for Vite
  // to output a URL" race, which is common when the project lives on the
  // slow /mnt/c Windows filesystem under WSL.
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true
  }
})
