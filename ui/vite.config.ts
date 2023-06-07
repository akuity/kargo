/* eslint-disable @typescript-eslint/no-explicit-any */

import react from '@vitejs/plugin-react';
import { theme } from 'antd';
import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';

import { token } from './src/config/theme';

const { defaultAlgorithm, defaultSeed } = theme;

const mapToken = defaultAlgorithm(defaultSeed);

export const UI_VERSION = process.env.VERSION || 'development';

// https://vitejs.dev/config/
export default defineConfig({
  build: {
    outDir: 'build',
    sourcemap: false
  },
  css: {
    preprocessorOptions: {
      less: {
        javascriptEnabled: true,
        modifyVars: { ...mapToken, ...token }
      }
    }
  },
  plugins: [tsConfigPaths(), react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:30081',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '')
      }
    },
    port: 3333
  }
});
