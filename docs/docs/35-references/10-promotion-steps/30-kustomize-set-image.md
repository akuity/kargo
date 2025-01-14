---
sidebar_label: kustomize-set-image
description: Updates the `kustomization.yaml` file in a specified directory to reflect a different revision of a container image.
---

# `kustomize-set-image`

`kustomize-set-image` updates the `kustomization.yaml` file in a specified
directory to reflect a different revision of a container image. It is equivalent
to executing `kustomize edit set image`. This step is commonly followed by a
[`kustomize-build` step](39-kustomize-build.md).

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a directory containing a `kustomization.yaml` file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `images` | `[]object` | N | The details of changes to be applied to the `kustomization.yaml` file. When left unspecified, all images from the Freight collection will be set in the Kustomization file. Unless there is an ambiguous image name (for example, due to two Warehouses subscribing to the same repository), which requires manual configuration. |
| `images[].image` | `string` | Y | Name/URL of the image being updated. |
| `images[].tag` | `string` | N | A tag naming a specific revision of `image`. Mutually exclusive with `digest` and `useDigest=true`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of this field. |
| `images[].digest` | `string` | N | A digest naming a specific revision of `image`. Mutually exclusive with `tag` and `useDigest=true`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of `tag`. |
| `images[].useDigest` | `boolean` | N | Whether to update the `kustomization.yaml` file using the container image's digest instead of its tag. Mutually exclusive with `digest` and `tag`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of `tag`. <br/><br/>__Deprecated: Use `digest` with an expression instead. Will be removed in v1.3.0.__ |
| `images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Use `digest` or `tag` with an expression instead. Will be removed in v1.3.0.__ |
| `images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |


## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit` step](15-git-commit.md) will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |


## Examples

### Common Usage

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: imageRepo
  value: my/image
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
- uses: kustomize-set-image
  config:
    path: ./src/base
    images:
    - image: ${{ vars.imageRepo }}
      tag: ${{ imageFrom(vars.imageRepo).Tag }}
# Render manifests to ./out, commit, push, etc...
```

### Changing an Image Name

For this example, consider the promotion of Freight containing a reference to
some revision of the container image
`123456789012.dkr.ecr.us-east-1.amazonaws.com/my-image`. This image exists in the
`us-east-1` region of Amazon's Elastic Container Registry. However, assuming the
Stage targeted by the promotion is backed by environments in the `us-west-2`
region, it will be necessary to make a substitution when updating the
`kustomization.yaml` file. This can be accomplished like so:

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
- uses: kustomize-set-image
  config:
    path: ./src/base
    images:
    - image: 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-image
      newName: 123456789012.dkr.ecr.us-west-2.amazonaws.com/my-image
# Render manifests to ./out, commit, push, etc...
```
