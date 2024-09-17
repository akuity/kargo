export const promoMechanismExample = `gitRepoUpdates:
- repoURL: https://github.com/akuity/kargo-demo.git
  writeBranch: main
  kustomize:
    images:
    - image: public.ecr.aws/nginx/nginx
    path: stages/prod
argoCDAppUpdates:
- appName: kargo-demo-prod
  appNamespace: argocd`;
