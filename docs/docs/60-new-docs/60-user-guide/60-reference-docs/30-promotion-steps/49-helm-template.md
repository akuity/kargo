---
sidebar_label: helm-template
description: Renders a specified Helm chart to one or more files in a specified directory.
---

# `helm-template`

`helm-template` renders a specified Helm chart to a specified directory or to
many files in a specified directory. This step is useful for the common scenario
of rendering Stage-specific manifests to a Stage-specific branch. This step is
commonly preceded by a [`git-clear` step](11-git-clear.md) and followed by
[`git-commit`](15-git-commit.md) and [`git-push`](16-git-push.md) steps.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Helm chart (i.e. to a directory containing a `Chart.yaml` file). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. |
| `releaseName` | `string` | N | Optional release name to use when rendering the manifests. This is commonly omitted. |
| `namespace` | `string` | N | Optional namespace to use when rendering the manifests. This is commonly omitted. GitOps agents such as Argo CD will generally ensure the installation of manifests into the namespace specified by their own configuration. |
| `valuesFiles` | `[]string` | N | Helm values files (apart from the chart's default `values.yaml`) to be used when rendering the manifests.  |
| `includeCRDs` | `boolean` | N | Whether to include CRDs in the rendered manifests. This is `false` by default. |
| `kubeVersion` | `string` | N | Optionally specifies a Kubernetes version to be assumed when rendering manifests. This is useful for charts that may contain logic specific to different Kubernetes versions. |
| `apiVersions` | `[]string` | N | Allows a manual set of supported API versions to be specified. |

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
- uses: helm-template
  config:
    path: ./src/charts/my-chart
    valuesFiles:
    - ./src/charts/my-chart/${{ ctx.stage }}-values.yaml
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
- uses: helm-template
  config:
    path: ./src/charts/my-chart
    valuesFiles:
    - ./src/charts/my-chart/${{ ctx.stage }}-values.yaml
    outPath: ./out
# Commit, push, etc...
```
