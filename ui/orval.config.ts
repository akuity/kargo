import { defineConfig } from 'orval';

export default defineConfig({
  kargo: {
    input: {
      target: '../swagger.json'
    },
    output: {
      mode: 'tags-split',
      target: './src/gen/api/v2',
      schemas: './src/gen/api/v2/models',
      client: 'react-query',
      httpClient: 'fetch',
      clean: true,
      prettier: true,
      override: {
        mutator: {
          path: './src/lib/api/custom-fetch.ts',
          name: 'customFetch'
        },
        query: {
          useQuery: true,
          useMutation: true,
          signal: false
        }
      }
    },
    hooks: {
      afterAllFilesWrite: 'prettier --write'
    }
  }
});
