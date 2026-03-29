---
sidebar_label: kcl-run
description: Renders manifests from a specified KCL file or package directory to a specified file or to many files in a specified directory.
---

# `kcl-run`

<span class="tag beta"></span>

`kcl-run` renders manifests from a specified KCL file or package directory to a
specified file or to many files in a specified directory. This step is useful
for the common scenario of rendering Stage-specific manifests to a Stage-
specific branch. This step is commonly preceded by a [`git-clear`](git-clear.md)
step and followed by [`git-commit`](git-commit.md) and [`git-push`](git-push.md)
steps.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a KCL file, `kcl.yaml`, or package directory. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outputFormat` | `string` | N | Specifies the naming convention for output files when writing to a directory. `kargo` (default) uses `[namespace-]kind-name.yaml` format (e.g., `deployment-myapp.yaml` or `default-deployment-myapp.yaml`). `kustomize` matches the naming convention of `kustomize build -o dir/`, using `[namespace_]group_version_kind_name.yaml` format (e.g., `apps_v1_deployment_myapp.yaml` or `default_apps_v1_deployment_myapp.yaml`). |
| `arguments` | `[]object` | N | Top-level KCL arguments, equivalent to the `kcl run -D name=value` flag. |
| `arguments[].name` | `string` | Y | The name of the top-level KCL argument. |
| `arguments[].value` | `string` | Y | The value of the top-level KCL argument. |

## Examples

### Rendering to a File

In this example, KCL manifests are rendered to a single output file. After
cloning the repository and clearing the output directory, the `kcl-run` step
evaluates a stage-specific KCL file and writes the rendered manifests to a
single file at `./out/manifests.yaml`.

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
- uses: kcl-run
  config:
    path: ./src/stages/${{ ctx.stage }}/main.k
    outPath: ./out/manifests.yaml
    arguments:
    - name: stage
      value: ${{ ctx.stage }}
# Commit, push, etc...
```

### Rendering to a Directory

In this example, KCL manifests are rendered with output split across multiple
files in a directory. Similar to the [previous example](#rendering-to-a-file),
it clones the repository and clears the output directory, but instead of
specifying a single output file, it directs the `kcl-run` step to write to the
`./out` directory.

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
- uses: kcl-run
  config:
    path: ./src/stages/${{ ctx.stage }}/main.k
    outPath: ./out
    outputFormat: kustomize
    arguments:
    - name: stage
      value: ${{ ctx.stage }}
# Commit, push, etc...
```