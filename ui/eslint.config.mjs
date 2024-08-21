import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { fixupPluginRules, includeIgnoreFile } from '@eslint/compat';
import { FlatCompat } from '@eslint/eslintrc';
import js from '@eslint/js';
import typescriptEslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import _import from 'eslint-plugin-import';
import prettier from 'eslint-plugin-prettier';
import react from 'eslint-plugin-react';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all
});
const gitignorePath = path.resolve(__dirname, '.gitignore');

export default [
  includeIgnoreFile(gitignorePath),
  {
    ignores: ['**/*.cjs']
  },
  ...compat.extends('eslint:recommended', 'plugin:@typescript-eslint/recommended', 'prettier'),
  {
    plugins: {
      '@typescript-eslint': typescriptEslint,
      prettier,
      import: fixupPluginRules(_import),
      react: fixupPluginRules(react),
      'react-refresh': reactRefresh
    },

    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      },

      parser: tsParser,
      ecmaVersion: 2019,
      sourceType: 'module'
    },

    rules: {
      'prettier/prettier': 'error',
      '@typescript-eslint/explicit-module-boundary-types': 'off',

      'import/order': [
        'error',
        {
          groups: ['builtin', 'external', ['internal', 'unknown'], 'parent', 'sibling', 'index'],

          pathGroups: [
            {
              pattern: '@ui/**',
              group: 'internal'
            }
          ],

          pathGroupsExcludedImportTypes: [],
          'newlines-between': 'always',

          alphabetize: {
            order: 'asc'
          }
        }
      ],

      '@typescript-eslint/ban-ts-comment': 'warn',
      'no-console': 'error',
      'react/jsx-key': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { caughtErrors: 'none' }],
      'react-refresh/only-export-components': 'warn'
    }
  },
  {
    files: ['**/*.test.ts'],

    rules: {
      '@typescript-eslint/ban-ts-comment': 'off'
    }
  }
];
