# argocd_kustomize_commit_n_image

Argo CD driven, Kustomize. Kargo advances new commits together with new image versions into each stage.

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `kargo_demo_gitops_repo` | HTTPS URL of a fork of the `kargo-demo-gitops` repository. Substituted into the fixtures at runtime (the Warehouse subscription, the promotion's `gitRepo` var, and/or the Argo CD `ApplicationSet` source). |

Example:

```yaml
context:
  kargo_demo_gitops_repo: https://github.com/<you>/kargo-demo-gitops.git
```
