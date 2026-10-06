# git_kustomize_commit_n_image

Git driven, Kustomize commit-and-image. `prod` is promoted via a pull request.

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `kargo_demo_gitops_repo` | HTTPS URL of a fork of the `kargo-demo-gitops` repository. Substituted into the fixtures at runtime (the Warehouse subscription, the promotion's `gitRepo` var, and/or the Argo CD `ApplicationSet` source). |
| `git_pat` | GitHub personal access token with **write** access to that fork. The promotion pushes stage-specific branches and, for the `prod` stage, opens and merges a pull request. |

Example:

```yaml
context:
  kargo_demo_gitops_repo: https://github.com/<you>/kargo-demo-gitops.git
  git_pat: <github-personal-access-token>
```
