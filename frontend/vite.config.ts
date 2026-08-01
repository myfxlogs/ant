import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { patchAntdDropdown } from './vite-plugin-patch-antd'

export default defineConfig({
  plugins: [react(), patchAntdDropdown()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    proxy: {
      '/alphaforge.': {
        target: process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/ant.v1.': {
        target: process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
    headers: {
      'Cache-Control': 'no-store, no-cache, must-revalidate',
      'Pragma': 'no-cache',
      'Expires': '0',
    },
  },
  build: {
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: false,
      },
    },
    modulePreload: { polyfill: false },
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-antd': ['antd'],
          'vendor-charts': ['recharts', 'klinecharts'],
          'vendor-protobuf': ['@bufbuild/protobuf', '@connectrpc/connect', '@connectrpc/connect-web'],
        },
      },
    },
  },
})
