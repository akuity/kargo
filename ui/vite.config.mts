import react from '@vitejs/plugin-react';
import { theme } from 'antd';
import { defineConfig } from 'vite';
import viteCompression from 'vite-plugin-compression';
import monacoEditorPlugin from 'vite-plugin-monaco-editor';

import { token } from './src/config/themeConfig';

const { defaultAlgorithm, defaultSeed } = theme;

const mapToken = defaultAlgorithm(defaultSeed);

export const UI_VERSION = process.env.VERSION || 'development';
export const API_URL = process.env.API_URL || 'http://localhost:30081';
export const BUILD_TARGET_PATH = process.env.BUILD_TARGET_PATH || 'build';
// KARGO_BASE_PATH lets the dev server exercise a non-empty basePath end-to-end.
// Unset (or empty) means the dev server serves at the root, matching production
// without a basePath. When set (e.g. "/kargo"), the dev plugin below substitutes
// the index.html placeholders accordingly and the proxy paths get prefixed so
// API calls match what the backend expects when deployed at the same basePath.
export const KARGO_BASE_PATH = process.env.KARGO_BASE_PATH || '';

// https://vitejs.dev/config/
export default defineConfig({
  // Emit asset paths in built artifacts as document-relative ("./assets/foo.js")
  // rather than root-relative ("/assets/foo.js") so they resolve through the
  // <base href> the API server injects into the served index.html. This keeps
  // a single UI artifact compatible with any deployed basePath.
  base: './',
  define: {
    __UI_VERSION__: JSON.stringify(UI_VERSION)
  },
  resolve: {
    tsconfigPaths: true
  },
  build: {
    outDir: BUILD_TARGET_PATH,
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
    // In dev mode the API server isn't in front to substitute the
    // basePath placeholders in index.html, so do the substitution here
    // with empty-basePath values to keep dev-server HTML well-formed.
    // Build mode skips this hook so the placeholders survive into the
    // built artifact for the API server to substitute at serve time.
    {
      name: 'kargo-basepath-dev-substitute',
      apply: 'serve',
      transformIndexHtml(html: string) {
        const baseHref = KARGO_BASE_PATH ? `${KARGO_BASE_PATH}/` : '/';
        return html.replace(/__BASE_HREF__/g, baseHref).replace(/__BASE_PATH__/g, KARGO_BASE_PATH);
      }
    },
    viteCompression(),
    react({ exclude: [/\/node_modules\//] }),
    // https://github.com/vdesjs/vite-plugin-monaco-editor/issues/21
    (monacoEditorPlugin as unknown as { default: typeof monacoEditorPlugin }).default({
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
      [`${KARGO_BASE_PATH}/akuity.io.kargo.service.v1alpha1.KargoService`]: {
        target: API_URL,
        changeOrigin: true
      },
      [`${KARGO_BASE_PATH}/v1beta1`]: {
        target: API_URL,
        changeOrigin: true
      }
    },
    port: 3333
  }
});
