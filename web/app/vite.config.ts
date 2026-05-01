import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg', 'apple-touch-icon.png'],
      manifest: {
        name: 'Expenses',
        short_name: 'Expenses',
        description: 'Personal expense tracker',
        start_url: '/',
        display: 'standalone',
        theme_color: '#F4F1EA',
        background_color: '#F4F1EA',
        icons: [],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//],
        runtimeCaching: [
          {
            urlPattern: /\/api\/categories$/,
            handler: 'CacheFirst',
            options: {
              cacheName: 'api-categories',
              expiration: { maxAgeSeconds: 60 * 60 * 24 * 30 },
            },
          },
          {
            urlPattern: /\/api\/expenses(\?.*)?$/,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'api-expenses',
              expiration: { maxEntries: 50 },
            },
          },
          {
            urlPattern: /\/api\/insights(\?.*)?$/,
            handler: 'StaleWhileRevalidate',
            options: { cacheName: 'api-insights' },
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
