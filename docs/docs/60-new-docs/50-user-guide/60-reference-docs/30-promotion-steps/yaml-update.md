---
sidebar_label: yaml-update
description: Updates the values of specified keys in any YAML file.
---

# `yaml-update`

`yaml-update` updates the values of specified keys in any YAML file. This step
most often used to update image tags or digests in a Helm values and is commonly
followed by a [`helm-template` step](helm-template.md).

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a YAML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use dots to delimit key parts. e.g. `image.tag`. The syntax is identical to that supported by the `json-update` step and is documented in more detail [here](https://github.com/tidwall/sjson?tab=readme-ov-file#path-syntax). |
| `updates[].value` | `string` | Y | The new value for the key. Typically specified using an expression. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## Examples

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
- uses: yaml-update
  config:
    path: ./src/charts/my-chart/values.yaml
    updates:
    - key: image.tag
      value: ${{ imageFrom("my/image").Tag }}
# Render manifests to ./out, commit, push, etc...
```
