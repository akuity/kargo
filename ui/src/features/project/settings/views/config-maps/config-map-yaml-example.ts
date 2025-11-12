export const configMapYAMLExample = (project?: string) => ({
  apiVersion: 'v1',
  kind: 'ConfigMap',
  metadata: {
    name: 'cm-1',
    namespace: project || 'kargo-demo'
  },
  data: {
    foo: 'bar'
  }
});
