---
sidebar_label: helm-template
description: Renders a specified Helm chart to one or more files in a specified directory.
---

# `helm-template`

`helm-template` renders a specified Helm chart to a specified directory or to
many files in a specified directory. This step is useful for the common scenario
of rendering Stage-specific manifests to a Stage-specific branch. This step is
commonly preceded by a [`git-clear` step](git-clear.md) and followed by
[`git-commit`](git-commit.md) and [`git-push`](git-push.md) steps.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Helm chart (i.e. to a directory containing a `Chart.yaml` file). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. |
| `releaseName` | `string` | Y | Release name to use when rendering the manifests. |
| `useReleaseName` | `boolean` | N | Whether to use the release name in the output path (instead of the chart name). This is `false` by default, and only has an effect when `outPath` is set to a directory. |
| `namespace` | `string` | N | Optional namespace to use when rendering the manifests. This is commonly omitted. GitOps agents such as Argo CD will generally ensure the installation of manifests into the namespace specified by their own configuration. |
| `valuesFiles` | `[]string` | N | Helm values files (apart from the chart's default `values.yaml`) to be used when rendering the manifests.  |
| `includeCRDs` | `boolean` | N | Whether to include CRDs in the rendered manifests. This is `false` by default. |
| `disableHooks` | `boolean` | N | Whether to disable hooks in the rendered manifests. This is `false` by default. |
| `skipTests` | `boolean` | N | Whether to skip tests when rendering the manifests. This is `false` by default. |
| `kubeVersion` | `string` | N | Optionally specifies a Kubernetes version to be assumed when rendering manifests. This is useful for charts that may contain logic specific to different Kubernetes versions. |
| `apiVersions` | `[]string` | N | Allows a manual set of supported API versions to be specified. |
| `setValues` | `[]object` | N | Allows for amending chart configuration inline as one would with the `helm template` command's `--set` flag. |
| `setValues[].key` | `string` | N | The key whose value should be set. For nested values, use dots to delimit key parts. e.g. `image.tag`. |
| `setValues[].value` | `string` | N | The new value for the key. |

## Examples

### Rendering to a File

In this example, a Helm chart is rendered to a single output file. After
cloning the repository and clearing the output directory, the `helm-template`
step reads the chart from `./src/charts/my-chart` and uses stage-specific
values from `${{ ctx.stage }}-values.yaml` to render the manifests. The
rendered output is written to a single file at `./out/manifests.yaml`
rather than being split across multiple files.

This approach is useful when you want to maintain all manifests in a single
file for easier tracking or when working with tools that expect a single
input file.

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

In this example, a Helm chart is rendered with its output split across multiple
files in a directory. Similar to the [previous example](#rendering-to-a-file),
it clones the repository and clears the output directory, but instead of
specifying a single output file, it directs the `helm-template` step to write
to the `./out` directory. This means each Kubernetes resource will be written
to its own file, maintaining the traditional Helm output structure (i.e. the
structure of the Helm chart).

This approach is useful when you want to maintain separation between different
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
- uses: helm-template
  config:
    path: ./src/charts/my-chart
    valuesFiles:
    - ./src/charts/my-chart/${{ ctx.stage }}-values.yaml
    outPath: ./out
# Commit, push, etc...
```
