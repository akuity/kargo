---
sidebar_label: yaml-merge
description: Merge multiple YAML file into a single file.
---

# `yaml-merge`

`yaml-merge` merges multiple YAML files into a single file. This step
most often used to merge multiple Helm values.yaml files into a single
file and is commonly followed by a [`helm-template` step](helm-template.md).
YAML files are merged in order, so the first one is the base, and all
subsequent files are "overlays", modifying the default values.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPaths` | `[]string` | Y | Paths to a YAML files. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | The path to the output file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `strict` | `bool` | N | Strict will cause the directive to fail if an input path does not exist. Defaults to `false`. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In this example, two Helm values, one global, one more specific, are merged
into a new single file, that is then commited.

This pattern is commonly used when you need to merge global values
into a final `values.yaml` file, to be used by Helm.

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
- uses: yaml-merge
  config:
    inPaths:
      - ./src/charts/my-chart/values.yaml
      - ./src/charts/qa/values.yaml
      - ./src/charts/qa/cluster-a/values.yaml
    outPath: ./out/charts/my-chart/values.yaml
# Render manifests to ./out, commit, push, etc...
```
