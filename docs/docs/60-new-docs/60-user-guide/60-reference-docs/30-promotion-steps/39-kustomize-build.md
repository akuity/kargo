---
sidebar_label: kustomize-build
description: Renders manifests from a specified directory containing a `kustomization.yaml` file to a specified file or to many files in a specified directory.
---

# `kustomize-build`

`kustomize-build` renders manifests from a specified directory containing a
`kustomization.yaml` file to a specified file or to many files in a specified
directory. This step is useful for the common scenario of rendering
Stage-specific manifests to a Stage-specific branch. This step is commonly
preceded by a [`git-clear`](11-git-clear.md) step and followed by
[`git-commit`](15-git-commit.md) and [`git-push`](16-git-push.md) steps.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a directory containing a `kustomization.yaml` file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `plugin.helm.apiVersions` | `[]string` | N | Optionally specifies a list of supported API versions to be used when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes API versions. |
| `plugin.helm.kubeVersion` | `string` | N | Optionally specifies a Kubernetes version to be assumed when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes versions. |

## Examples

### Rendering to a File

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out/manifests.yaml
# Commit, push, etc...
```

### Rendering to a Directory

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out
# Commit, push, etc...
```
