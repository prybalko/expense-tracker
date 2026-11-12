import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

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
        // List is StaleWhileRevalidate for instant paint; ResumeSync's delta
        // catches up. Detail is NetworkFirst so an edit opens the latest row.
        // Writes have no rule and must fail offline rather than fake success.
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
