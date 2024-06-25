export const getStageYAMLExample = (namespace: string) =>
  `apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: prod
  namespace: ${namespace}
spec:
  subscriptions:
    upstreamStages:
    - name: uat
  promotionMechanisms:
    gitRepoUpdates:
    - repoURL: https://github.com/akuity/kargo-demo.git
      writeBranch: main
      kustomize:
        images:
        - image: public.ecr.aws/nginx/nginx
          path: stages/prod
    argoCDAppUpdates:
    - appName: kargo-demo-prod
      appNamespace: argocd`;
