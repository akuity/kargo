---
sidebar_label: helm-update-image
description: Updates the values of specified keys in a specified Helm values file to reflect a new version of a container image.
---

# `helm-update-image`

:::warning
**Deprecated:** Use the generic [`yaml-update` step](yaml-update.md) instead.
Will be removed in v1.3.0.
:::

`helm-update-image` updates the values of specified keys in a specified Helm
values file (e.g. `values.yaml`) to reflect a new version of a container image.
This step is useful for the common scenario of updating such a `values.yaml`
file with new version information which is referenced by the Freight being
promoted. This step is commonly followed by a
[`helm-template` step](helm-template.md).

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to Helm values file (e.g. `values.yaml`). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `images` | `[]object` | Y | The details of changes to be applied to the values file. At least one must be specified. |
| `images[].image` | `string` | Y | Name/URL of the image being updated. The Freight being promoted presumably contains a reference to a revision of this image. |
| `images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins) |
| `images[].key` | `string` | Y | The key to update within the values file. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `images[].value` | `string` | Y | Specifies how the value of `key` is to be updated. Possible values for this field are limited to:<ul><li>`ImageAndTag`: Replaces the value of `key` with a string in form `<image url>:<tag>`</li><li>`Tag`: Replaces the value of `key` with the image's tag</li><li>`ImageAndDigest`: Replaces the value of `key` with a string in form `<image url>@<digest>`</li><li>`Digest`: Replaces the value of `key` with the image's digest</li></ul> |

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
- uses: helm-update-image
  config:
    path: ./src/charts/my-chart/values.yaml
    images:
    - image: my/image
      key: image.tag
      value: Tag
# Render manifests to ./out, commit, push, etc...
```
