import 'monaco-editor/esm/vs/editor/editor.all.js';
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution.js';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.main.js';

// Fixes issues with monaco-yaml web worker
// Work around https://github.com/remcohaszing/monaco-yaml/issues/272
const { createWebWorker: oldCreateWebWorker } = monaco.editor;
monaco.editor.createWebWorker = (
  opts: monaco.IWebWorkerOptions | monaco.editor.IInternalWebWorkerOptions
) => {
  if ('worker' in opts) {
    return oldCreateWebWorker(opts);
  }

  return monaco.createWebWorker(opts);
};
