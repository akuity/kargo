# argocd_wait

Exercises the `argocd-wait` promotion step. A branch is created uniquely per test
run in setup (a copy of the `kustomize` branch) and deleted in teardown; the Argo
CD `Application` tracks that branch with **auto-sync** enabled. The promotion
checks out the existing branch, updates the image, commits and pushes, then uses
`argocd-wait` to block until Argo CD reconciles the change (Healthy + Synced).
Argo CD's own auto-sync triggers the sync -- not an `argocd-update` step -- which
is the scenario `argocd-wait` exists for.

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `kargo_demo_gitops_repo` | HTTPS URL of a fork of the `kargo-demo-gitops` repository. Substituted into the fixtures at runtime (the git credentials Secret, the promotion's `gitRepo` var, and the Argo CD `ApplicationSet` source). |
| `git_pat` | GitHub personal access token with **write** access to that fork. The promotion pushes the per-run branch that Argo CD tracks. |

Example:

```yaml
context:
  kargo_demo_gitops_repo: https://github.com/<you>/kargo-demo-gitops.git
  git_pat: <github-personal-access-token>
```

## Note

Each run creates a branch named `argocd-wait/e2e/<timestamp>` on the fork in
setup, pointing it at the head of the `kustomize` branch via the go-github API
(no clone, no `git` binary). The branch is left behind, like the other git-driven
suites' branches.
