import { editor, languages } from 'monaco-editor';
import { useEffect } from 'react';

export const monacoEditorLogLanguage = 'logs';
export const monacoEditorLogLanguageTheme = 'logsTheme';
export const monacoEditorLogLanguageThemeDark = 'logsThemeDark';

export const useMonacoEditorLogLanguage = () => {
  useEffect(() => {
    languages.register({ id: monacoEditorLogLanguage });

    languages.setMonarchTokensProvider(monacoEditorLogLanguage, {
      tokenizer: {
        root: [
          [/\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})?\b/, 'time-format']
        ]
      }
    });

    editor.defineTheme(monacoEditorLogLanguageTheme, {
      base: 'vs',
      inherit: true,
      rules: [
        {
          token: 'time-format',
          foreground: '064497'
        }
      ],
      colors: {}
    });

    editor.defineTheme(monacoEditorLogLanguageThemeDark, {
      base: 'vs-dark',
      inherit: true,
      rules: [
        {
          token: 'time-format',
          foreground: '4aa3ff'
        }
      ],
      colors: { 'editor.background': '#0f1722' }
    });
  }, []);
};
