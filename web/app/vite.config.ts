import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      manifestFilename: 'manifest.json',
      devOptions: {
        enabled: true,
      },
      includeAssets: ['favicon.svg', 'apple-touch-icon.png'],
      manifest: {
        name: 'Expenses',
        short_name: 'Expenses',
        description: 'Personal expense tracker',
        start_url: '/',
        display: 'standalone',
        theme_color: '#F4F1EA',
        background_color: '#F4F1EA',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          { src: '/icons/icon-maskable-192.png', sizes: '192x192', type: 'image/png', purpose: 'maskable' },
          { src: '/icons/icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//],
        // Read-only offline: NetworkFirst lets the SW serve the last-seen
        // expenses payload when the user is offline (subway, airplane mode)
        // while always preferring fresh data when the network is available.
        // Writes (POST/PATCH/DELETE) have no rule and pass through to the
        // network — failing them is preferable to faking success.
        runtimeCaching: [
          {
            urlPattern: /\/api\/expenses(\?.*)?$/,
            method: 'GET',
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-expenses',
              expiration: { maxEntries: 50 },
            },
          },
          {
            urlPattern: /\/api\/expenses\/\d+$/,
            method: 'GET',
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-expense-detail',
              expiration: { maxEntries: 100 },
            },
          },
        ],
      },
    }),
  ],
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
