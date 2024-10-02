export const promoStepsExample = `- uses: git-clone
  config:
    repoURL: https://github.com/<github-username>/kargo-demo-gitops.git
    checkout:
        - fromFreight: true
        path: ./main
        - branch: 09/stage/test
        create: true
        path: ./out
- uses: git-overwrite
  config:
    inPath: ./main
    outPath: ./out`;
