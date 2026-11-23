import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { visualizer } from 'rollup-plugin-visualizer'

export default defineConfig({
  // Relative asset URLs, resolved against the <base> the server injects, so
  // the build works under any --prefix.
  base: './',
  plugins: [
    react(),
    tailwindcss(),
    visualizer({ filename: 'bundle-stats.html', gzipSize: true }),
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../static/dist',
    emptyOutDir: true,
  },
})
