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
        // Read-only offline with a split read strategy:
        //   - List (/api/expenses): StaleWhileRevalidate. Reload / PWA
        //     relaunch paints the Hero off the cached payload in a
        //     microtask instead of waiting on the network, collapsing the
        //     "0 → real total" flash to one frame. The background
        //     revalidate keeps the cache warm; multi-device deltas land
        //     via the separate /api/expenses/changes call fired from
        //     useSyncOnVisible on every Feed visibility change.
        //   - Detail (/api/expenses/:id): NetworkFirst. The EntryForm
        //     should always open against the latest server-side row when
        //     online so a cross-device edit doesn't overwrite fresher
        //     data; the cache is purely the offline fallback.
        // Writes (POST/PATCH/DELETE) have no rule and pass through to the
        // network — failing them is preferable to faking success.
        runtimeCaching: [
          {
            urlPattern: /\/api\/expenses(\?.*)?$/,
            method: 'GET',
            handler: 'StaleWhileRevalidate',
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
