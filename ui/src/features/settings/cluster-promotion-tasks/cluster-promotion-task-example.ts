export const getClusterPromotionTaskYAMLExample = () =>
  `apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterPromotionTask
metadata:
    name: global-task
spec:
    # Task-wide input variables
    vars:
    - name: repoURL
    - name: sourceBranch
    - name: targetBranch
      value: main
  
    # Sequence of promotion steps
    steps:
    - uses: git-open-pr
      as: open-pr
      config:
        repoURL: \${{ vars.repoURL }}
        createTargetBranch: true
        sourceBranch: \${{ vars.sourceBranch }}
        targetBranch: \${{ vars.targetBranch }}`;
