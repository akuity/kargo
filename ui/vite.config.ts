/* eslint-disable @typescript-eslint/no-explicit-any */

import react from '@vitejs/plugin-react';
import { theme } from 'antd';
import { defineConfig } from 'vite';
import monacoEditorPlugin from 'vite-plugin-monaco-editor';
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
  plugins: [
    tsConfigPaths(),
    react(),
    monacoEditorPlugin({
      customWorkers: [
        {
          label: 'yaml',
          entry: 'monaco-yaml/yaml.worker'
        }
      ]
    })
  ],
  server: {
    proxy: {
      '/akuity.io.kargo.service.v1alpha1.KargoService': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    },
    port: 3333
  }
});
