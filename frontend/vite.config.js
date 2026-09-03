import path from 'path'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
      // Web shim for the Tauri API. The desktop build uses
      // the real @tauri-apps/api/core from node_modules.
      '@tauri-apps/api/core': path.resolve(import.meta.dirname, './src/lib/web/tauri-core.js'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    watch: {
      ignored: ['**/src-tauri/**'],
    },
    proxy: {
      '/api/chat': {
        target: 'http://127.0.0.1:8585',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/chat/, '/v0/chat'),
      },
      '/api/router': {
        target: 'http://127.0.0.1:8484',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/router/, ''),
      },
    },
  },
})
