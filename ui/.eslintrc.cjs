module.exports = {
    root: true,
    parser: '@typescript-eslint/parser',
    extends: ['eslint:recommended', 'plugin:@typescript-eslint/recommended', 'prettier'],
    plugins: ['@typescript-eslint', 'prettier', 'import', 'react', 'react-refresh'],
    ignorePatterns: ['*.cjs'],
    parserOptions: {
      sourceType: 'module',
      ecmaVersion: 2019
    },
    env: {
      browser: true,
      es2017: true,
      node: true
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
      '@typescript-eslint/no-unused-vars': 'error',
      'react-refresh/only-export-components': 'warn'
    },
    'overrides': [
      {
        files: ['*.test.ts'],
        rules: {
          '@typescript-eslint/ban-ts-comment': 'off'
        }
      }
    ]
  };
  