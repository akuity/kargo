---
sidebar_label: kustomize-build
description: Renders manifests from a specified directory containing a `kustomization.yaml` file to a specified file or to many files in a specified directory.
---

# `kustomize-build`

`kustomize-build` renders manifests from a specified directory containing a
`kustomization.yaml` file to a specified file or to many files in a specified
directory. This step is useful for the common scenario of rendering
Stage-specific manifests to a Stage-specific branch. This step is commonly
preceded by a [`git-clear`](git-clear.md) step and followed by
[`git-commit`](git-commit.md) and [`git-push`](git-push.md) steps.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a directory containing a `kustomization.yaml` file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `plugin.helm.apiVersions` | `[]string` | N | Optionally specifies a list of supported API versions to be used when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes API versions. |
| `plugin.helm.kubeVersion` | `string` | N | Optionally specifies a Kubernetes version to be assumed when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes versions. |

## Examples

### Rendering to a File

In this example, Kustomize manifests are rendered to a single output file. After
cloning the repository and clearing the output directory, the `kustomize-build`
step processes the Kustomize configuration from the stage-specific directory
(`./src/stages/${{ ctx.stage }}`) and writes all rendered manifests to a single
file at `./out/manifests.yaml`.

This approach is useful when you want to maintain all manifests in a single file
for easier tracking or when working with tools that expect a consolidated input
file.

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

In this example, Kustomize manifests are rendered with output split across
multiple files in a directory. Similar to the
[previous example](#rendering-to-a-file), it clones the repository and
clears the output directory, but instead of specifying a single output file,
it directs the `kustomize-build` step to write to the `./out` directory. This
means each Kubernetes resource will be written to its own file, maintaining a
clear separation between resources.

This pattern is useful when you want to maintain distinct files for different
resources or when working with tools that expect resources to be in separate
files.

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
