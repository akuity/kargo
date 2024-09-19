export const testDirectives = [
  {
    step: 'fake-custom-directive',
    config: {
      str: 'Hello world!',
      ok: true,
      list: ['Item 1', 'Item 2'],
      nested: {
        enabled: 1
      }
    }
  },
  {
    step: 'git-clone',
    config: {
      url: 'https://example.com/org/repo.git',
      checkout: [
        {
          branch: 'stage/test',
          path: './stage/test'
        },
        {
          tag: 'v1.2.3',
          path: './release/v1.2.3'
        }
      ]
    }
  },
  {
    step: 'kargo-render',
    config: {
      inPath: './src',
      outPath: './build',
      targetBranch: 'stage/test'
    }
  },
  {
    step: 'kustomize-set-image',
    config: {
      path: './build',
      images: [
        {
          image: 'nginx',
          fromOrigin: {
            kind: 'Warehouse',
            name: 'demo'
          },
          useDigest: true
        }
      ]
    }
  },
  {
    step: 'kustomize-build',
    config: {
      path: './stage/test/app/test',
      outPath: './build/manifests.yaml'
    }
  },
  {
    step: 'helm-update-image',
    config: {
      path: './chart/values.yaml',
      images: [
        {
          image: 'nginx',
          key: 'nginx.image.tag',
          value: 'Tag'
        }
      ]
    }
  },
  {
    step: 'helm-update-chart',
    config: {
      path: './chart',
      charts: [
        {
          repository: 'https://helm.example.com',
          origin: {
            kind: 'Warehouse',
            name: 'demo'
          }
        }
      ]
    }
  },
  {
    step: 'helm-template',
    config: {
      path: './chart',
      outPath: './build/manifets.yaml',
      releaseName: 'my-app',
      valuesFiles: ['./env/test/values.yaml']
    }
  },
  {
    step: 'argocd-health',
    config: {
      name: 'fake-app',
      namespace: 'fake-namespace'
    }
  },
  {
    step: 'git-commit',
    config: {
      path: './stage/test',
      author: {
        name: 'Kargo',
        email: 'kargo@example.com'
      },
      message: 'Automated commit by Kargo'
    }
  },
  {
    step: 'git-push',
    config: {
      path: './stage/test'
    }
  },
  {
    step: 'pr-open',
    config: {
      path: './stage/test',
      targetBranch: 'stage/test',
      provider: 'github'
    }
  },
  {
    step: 'pr-wait',
    config: {
      fromStep: 'my-alias',
      timeout: '1h'
    }
  },
  {
    step: 'git-overwrite',
    config: {
      inPath: './build',
      outPath: './stage/test'
    }
  },
  {
    step: 'copy',
    config: {
      inPath: './build/file.yaml',
      outPath: './stage/test/file.yaml',
      recursive: false,
      overwrite: true
    }
  }
];
