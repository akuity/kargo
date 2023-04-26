/* eslint-disable @typescript-eslint/no-explicit-any */

import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';

export const UI_VERSION = process.env.VERSION || 'development';

// https://vitejs.dev/config/
export default defineConfig({
  build: {
    outDir: 'build',
    sourcemap: false
  },
  plugins: [tsConfigPaths(), react()],
  server: {
    port: 3333
  },
  test: {}
});
