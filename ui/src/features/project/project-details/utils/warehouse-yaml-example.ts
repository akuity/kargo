export const getWarehouseYAMLExample = (namespace: string) =>
  `apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: kargo-demo
  namespace: ${namespace}
spec:
  subscriptions:
  - image:
      repoURL: nginx
      semverConstraint: ^1.24.0`;
