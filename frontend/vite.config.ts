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
    chunkSizeWarningLimit: 700,
    terserOptions: {
      compress: {
        drop_console: false,
      },
    },
    modulePreload: { polyfill: false },
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('/antd/')) return 'vendor-antd';
            if (id.includes('@ant-design/icons')) return 'vendor-icons';
            if (id.includes('react-router')) return 'vendor-router';
            if (id.includes('react-dom') || id.includes('scheduler')) return 'vendor-react-dom';
            if (id.includes('/react/')) return 'vendor-react';
            if (id.includes('recharts') || id.includes('klinecharts') || id.includes('d3-')) return 'vendor-charts';
            if (id.includes('@bufbuild/protobuf') || id.includes('@connectrpc/connect')) return 'vendor-protobuf';
            if (id.includes('i18next') || id.includes('react-i18next')) return 'vendor-i18n';
            if (id.includes('dayjs') || id.includes('date-fns')) return 'vendor-date';
            if (id.includes('monaco-editor') || id.includes('@monaco-editor')) return 'vendor-monaco';
            if (id.includes('zustand')) return 'vendor-zustand';
            if (id.includes('react-syntax-highlighter') || id.includes('refractor') || id.includes('prismjs')) return 'vendor-prism';
            if (id.includes('highlight.js') || id.includes('lowlight')) return 'vendor-highlight';
            if (id.includes('react-markdown') || id.includes('remark') || id.includes('micromark') || id.includes('mdast')) return 'vendor-markdown';
            if (id.includes('@codemirror')) return 'vendor-codemirror';
            if (id.includes('@sentry')) return 'vendor-sentry';
            if (id.includes('@emotion') || id.includes('@rc-component') || id.includes('clsx') || id.includes('decimal.js')) return 'vendor-ui-utils';
            return 'vendor-misc';
          }
          if (id.includes('/src/gen/')) return 'gen-proto';
          if (id.includes('/src/i18n/resources/zh-cn/')) return 'i18n-zh-cn';
          if (id.includes('/src/i18n/resources/en/')) return 'i18n-en';
          if (id.includes('/src/i18n/resources/zh-tw/')) return 'i18n-zh-tw';
          if (id.includes('/src/i18n/resources/ja/')) return 'i18n-ja';
          if (id.includes('/src/i18n/resources/vi/')) return 'i18n-vi';
          if (id.includes('/src/i18n/resources/')) return 'i18n-other';
        },
      },
    },
  },
})
