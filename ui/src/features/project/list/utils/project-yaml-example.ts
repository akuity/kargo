export const projectYAMLExample = {
  apiVersion: 'kargo.akuity.io/v1alpha1',
  kind: 'Project',
  metadata: {
    name: 'kargo-demo'
  }
};

export const projectConfigYAMLExample = {
  apiVersion: 'kargo.akuity.io/v1alpha1',
  kind: 'ProjectConfig',
  metadata: {
    name: 'kargo-demo',
    namespace: 'kargo-demo'
  }
};
