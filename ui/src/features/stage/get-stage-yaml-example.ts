export const getStageYAMLExample = (namespace: string) =>
  `apiVersion: kargo.akuity.io/v1alpha1
  kind: Stage
  metadata:
    name: test
    namespace: ${namespace}
  spec:
    requestedFreight:
      - origin:
          kind: Warehouse
          name: kargo-demo
        sources:
          direct: true
    promotionTemplate:
      spec:
        steps:
        - uses: git-clone
          config:
            repoURL: https://github.com/akuity/kargo-advanced
            checkout:
            - branch: main
              path: ./src
            - branch: stage/uat
              create: true
              path: ./out`;
