---
sidebar_label: Promotion Steps Reference
description: Learn about all of Kargo's built-in promotion steps
---

# Promotion Steps Reference

This reference document describes the promotion steps that are built directly
into Kargo. Steps are presented roughly in the order in which they might appear
in a typical promotion process. Similarly, configuration options for each step
are laid out in order of their applicability to typical use cases.

:::info
Promotion steps support the use of [expr-lang] expressions in their
configuration. Many examples in this reference document will include expressions
to demonstrate their use. For more information on expressions, refer to our
[Expression Language Reference](./20-expression-language.md).
:::

## `git-clone`

`git-clone` is often the first step in a promotion process. It creates a
[bare clone](https://git-scm.com/docs/git-clone#Documentation/git-clone.txt-code--barecode)
of a remote Git repository, then checks out one or more branches, tags, or
commits to working trees at specified paths. Checking out different revisions to
different paths is useful for the common scenarios of combining content from
multiple sources or rendering Stage-specific manifests to a Stage-specific
branch.

:::note
It is a noteworthy limitation of Git that one branch cannot be checked out in
multiple working trees.
:::

### `git-clone` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository to clone. |
| `insecureSkipTLSVerify` | `boolean` | N | Whether to bypass TLS certificate verification when cloning (and for all subsequent operations involving this clone). Setting this to `true` is highly discouraged in production. |
| `checkout` | `[]object` | Y | The commits, branches, or tags to check out from the repository and the paths where they should be checked out. At least one must be specified. |
| `checkout[].branch` | `string` | N | A branch to check out. Mutually exclusive with `commit`, `tag`, and `fromFreight=true`. If none of these is specified, the default branch will be checked out. |
| `checkout[].create` | `boolean` | N | In the event `branch` does not already exist on the remote, whether a new, empty, orphaned branch should be created. Default is `false`, but should commonly be set to `true` for Stage-specific branches, which may not exist yet at the time of a Stage's first promotion. |
| `checkout[].commit` | `string` | N | A specific commit to check out. Mutually exclusive with `branch`, `tag`, and `fromFreight=true`. If none of these is specified, the default branch will be checked out. |
| `checkout[].tag` | `string` | N | A tag to check out. Mutually exclusive with `branch`, `commit`, and `fromFreight=true`. If none of these is specified, the default branch will be checked out. |
| `checkout[].fromFreight` | `boolean` | N | Whether a commit to check out should be obtained from the Freight being promoted. A value of `true` is mutually exclusive with `branch`, `commit`, and `tag`. If none of these is specified, the default branch will be checked out. Default is `false`, but is often set to `true`. <br/><br/>__Deprecated: Use `commit` with an expression instead. Will be removed in v1.3.0.__ |
| `checkout[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Use `commit` with an expression instead. Will be removed in v1.3.0.__ |
| `checkout[].path` | `string` | Y | The path for a working tree that will be created from the checked out revision. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |

### `git-clone` Examples

<Tabs groupId="git-clone-examples">

<TabItem value="common" label="Common Usage" default>

The most common usage of this step is to check out a commit specified by the
Freight being promoted as well as a Stage-specific branch. Subsequent steps are
likely to perform actions that revise the contents of the Stage-specific branch
using the commit from the Freight as input.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo) }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
# Prepare the contents of ./out ...
# Commit, push, etc...
```

</TabItem>

<TabItem value="multiple-sources" label="Combining Multiple Sources">

For this more advanced example, consider a Stage that requests Freight from two
Warehouses, where one provides Kustomize "base" configuration, while the other
provides a Stage-specific Kustomize overlay. Rendering the manifests intended
for such a Stage will require combining the base and overlay configurations
with the help of a [`copy`](#copy) step. For this case, a `git-clone` step may be
configured similarly to the following:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo, warehouse("base")).ID }}
      path: ./src
    - commit: ${{ commitFrom(vars.gitRepo, warehouse(ctx.stage + "-overlay")).ID }}
      path: ./overlay
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/${{ ctx.stage }}/kustomization.yaml
    outPath: ./src/stages/${{ ctx.stage }}/kustomization.yaml
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out
# Commit, push, etc...
```

</TabItem>

</Tabs>

## `git-clear`

`git-clear` deletes the _the entire contents_ of a specified Git working tree
(except for the `.git` file). It is equivalent to executing
`git add . && git rm -rf --ignore-unmatch .`. This step is useful for the common
scenario where the entire content of a Stage-specific branch is to be replaced
with content from another branch or with content rendered using some
configuration management tool.

### `git-clear` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree whose entire contents are to be deleted. |

### `git-clear` Example

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
# Prepare the contents of ./out ...
# Commit, push, etc...
```

## `copy`

`copy` copies files or the contents of entire directories from one specified
location to another.

### `copy` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | Path to the file or directory to be copied. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the destination. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |

### `copy` Example

The most common (though still advanced) usage of this step is to combine content
from two working trees to use as input to a subsequent step, such as one that
renders Stage-specific manifests.

Consider a Stage that requests Freight from two Warehouses, where one provides
Kustomize "base" configuration, while the other provides a Stage-specific
Kustomize overlay. Rendering the manifests intended for such a Stage will
require combining the base and overlay configurations:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo, warehouse("base")).ID }}
      path: ./src
    - commit: ${{ commitFrom(vars.gitRepo, warehouse(ctx.stage + "-overlay")).ID }}
      path: ./overlay
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/${{ ctx.stage }}/kustomization.yaml
    outPath: ./src/stages/${{ ctx.stage }}/kustomization.yaml
# Render manifests to ./out, commit, push, etc...
```

## `kustomize-set-image`

`kustomize-set-image` updates the `kustomization.yaml` file in a specified
directory to reflect a different revision of a container image. It is equivalent
to executing `kustomize edit set image`. This step is commonly followed by a
[`kustomize-build`](#kustomize-build) step.

### `kustomize-set-image` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a directory containing a `kustomization.yaml` file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `images` | `[]object` | Y | The details of changes to be applied to the `kustomization.yaml` file. At least one must be specified. |
| `images[].image` | `string` | Y | Name/URL of the image being updated. |
| `images[].tag` | `string` | N | A tag naming a specific revision of `image`. Mutually exclusive with `digest` and `useDigest=true`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of this field. |
| `images[].digest` | `string` | N | A digest naming a specific revision of `image`. Mutually exclusive with `tag` and `useDigest=true`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of `tag`. |
| `images[].useDigest` | `boolean` | N | Whether to update the `kustomization.yaml` file using the container image's digest instead of its tag. Mutually exclusive with `digest` and `tag`. If none of these are specified, the tag specified by a piece of Freight referencing `image` will be used as the value of `tag`. <br/><br/>__Deprecated: Use `digest` with an expression instead. Will be removed in v1.3.0.__ |
| `images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Use `digest` or `tag` with an expression instead. Will be removed in v1.3.0.__ |
| `images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |

### `kustomize-set-image` Examples

<Tabs groupId="kustomize-set-image">

<TabItem value="common" label="Common Usage" default>

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
      tag: ${{ imageFrom(vars.imageRepo).tag }}
# Render manifests to ./out, commit, push, etc...
```

</TabItem>

<TabItem value="name-change" label="Changing an Image Name">

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

</TabItem>

</Tabs>

### `kustomize-set-image` Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit`](#git-commit) step will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## `kustomize-build`

`kustomize-build` renders manifests from a specified directory containing a
`kustomization.yaml` file to a specified file or to many files in a specified
directory. This step is useful for the common scenario of rendering
Stage-specific manifests to a Stage-specific branch. This step is commonly
preceded by a [`git-clear`](#git-clear) step and followed by
[`git-commit`](#git-commit) and [`git-push`](#git-push) steps.

### `kustomize-build` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a directory containing a `kustomization.yaml` file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to the file or directory where rendered manifests are to be written. If the path ends with `.yaml` or `.yml` it is presumed to indicate a file and is otherwise presumed to indicate a directory. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `plugin.helm.apiVersions` | `[]string` | N | Optionally specifies a list of supported API versions to be used when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes API versions. |
| `plugin.helm.kubeVersion` | `string` | N | Optionally specifies a Kubernetes version to be assumed when rendering manifests using Kustomize's Helm chart plugin. This is useful for charts that may contain logic specific to different Kubernetes versions. |

### `kustomize-build` Examples

<Tabs groupId="kustomize-build">

<TabItem value="file" label="Rendering to a File" default>

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

</TabItem>

<TabItem value="dir" label="Rendering to a Directory" default>

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

</TabItem>

</Tabs>

## `helm-update-image`

`helm-update-image` updates the values of specified keys in a specified Helm
values file (e.g. `values.yaml`) to reflect a new version of a container image.
This step is useful for the common scenario of updating such a `values.yaml`
file with new version information which is referenced by the Freight being
promoted. This step is commonly followed by a [`helm-template`](#helm-template)
step.

__Deprecated: Use the generic `yaml-update` step instead. Will be removed in v1.3.0.__

### `helm-update-image` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to Helm values file (e.g. `values.yaml`). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `images` | `[]object` | Y | The details of changes to be applied to the values file. At least one must be specified. |
| `images[].image` | `string` | Y | Name/URL of the image being updated. The Freight being promoted presumably contains a reference to a revision of this image. |
| `images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins) |
| `images[].key` | `string` | Y | The key to update within the values file. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `images[].value` | `string` | Y | Specifies how the value of `key` is to be updated. Possible values for this field are limited to:<ul><li>`ImageAndTag`: Replaces the value of `key` with a string in form `<image url>:<tag>`</li><li>`Tag`: Replaces the value of `key` with the image's tag</li><li>`ImageAndDigest`: Replaces the value of `key` with a string in form `<image url>@<digest>`</li><li>`Digest`: Replaces the value of `key` with the image's digest</li></ul> |

### `helm-update-image` Example

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

### `helm-update-image` Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit`](#git-commit) step will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## `yaml-update`

`yaml-update` updates the values of specified keys in any YAML file. This step
most often used to update image tags or digests in a Helm values and is commonly
followed by a [`helm-template`](#helm-template) step.

### `yaml-update` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a YAML file. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `updates` | `[]object` | Y | The details of changes to be applied to the file. At least one must be specified. |
| `updates[].key` | `string` | Y | The key to update within the file. For nested values, use a YAML dot notation path. |
| `updates[].value` | `string` | Y | The new value for the key. Typically specified using an expression. |

### `yaml-update` Example

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
      value: ${{ imageFrom("my/image").tag }}
# Render manifests to ./out, commit, push, etc...
```

### `yaml-update` Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit`](#git-commit) step will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## `helm-update-chart`

`helm-update-chart` performs specified updates on the `dependencies` section of
a specified Helm chart's `Chart.yaml` file. This step is useful for the common
scenario of updating a chart's dependencies to reflect new versions of charts
referenced by the Freight being promoted. This step is commonly followed by a
[`helm-template`](#helm-template) step.

### `helm-update-chart` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Helm chart (i.e. to a directory containing a `Chart.yaml` file). This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `charts` | `[]string` | Y | The details of dependency (subschart) updates to be applied to the chart's `Chart.yaml` file. |
| `charts[].repository` | `string` | Y | The URL of the Helm chart repository in the `dependencies` entry whose `version` field is to be updated. Must _exactly_ match the `repository` field of that entry. |
| `charts[].name` | `string` | Y | The name of the chart in in the `dependencies` entry whose `version` field is to be updated. Must exactly match the `name` field of that entry. |
| `charts[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Use `version` with an expression instead. Will be removed in v1.3.0.__ |
| `charts[].version` | `string` | N | The version to which the dependency should be updated. If left unspecified, the version specified by a piece of Freight referencing this chart will be used. |

### `helm-update-chart` Examples

<Tabs groupId="helm-update-chart">

<TabItem value="classic" label="Classic Chart Repository" default>

Given a `Chart.yaml` file such as the following:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: https://example-chart-repo
  name: some-chart
  version: 1.2.3
```

The `dependencies` can be updated to reflect the version of `some-chart`
referenced by the Freight being promoted like so:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: chartRepo
  value: https://example-chart-repo
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
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: ${{ chartRepo }}
      name: some-chart
      version: ${{ chartFrom(chartRepo).Version }}
# Render manifests to ./out, commit, push, etc...
```

</TabItem>

<TabItem value="oci" label="OCI Chart Repository" default>

:::caution
Classic (HTTP/HTTPS) Helm chart repositories can contain many differently named
charts. A specific chart, therefore, can be identified by a repository URL and
a chart name.

OCI repositories, on the other hand, are organizational constructs within OCI
_registries._ Each OCI repository is presumed to contain versions of only a
single chart. As such, a specific chart can be identified by a repository URL
alone.

Kargo Warehouses understand this distinction well, so a subscription to an OCI
chart repository will utilize its URL only, _without_ specifying a chart name.
For example:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - chart:
      repoURL: oci://example-chart-registry/some-chart
      semverConstraint: ^1.0.0
```

Helm deals with this difference somewhat more awkwardly, however. When a Helm
chart references a chart in an OCI repository, it must reference the _registry_
by URL in the `repository` field and _still_ specify a chart name in the name
field. For example:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: oci://example-chart-registry
  name: some-chart
  version: 1.2.3
```

__When using `helm-update-chart` to update the dependencies in a `Chart.yaml`
file, you must play by Helm's rules and use the _registry_ URL in the
`repository` field and the _repository_ name (chart name) in the `name` field.__
:::

:::info
As a general rule, when configuring Kargo to update something, observe the
conventions of the thing being updated, even if those conventions differ from
Kargo's own. Kargo is aware of such differences and will adapt accordingly.
:::

Given a `Chart.yaml` file such as the following:

```yaml
apiVersion: v2
name: example
type: application
version: 0.1.0
appVersion: 0.1.0
dependencies:
- repository: oci://example-chart-registry
  name: some-chart
  version: 1.2.3
```

The `dependencies` can be updated to reflect the version of
`oci://example-chart-registry/some-chart` referenced by the Freight being
promoted like so:

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: chartReg
  value: oci://example-chart-registry
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
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: ${{ chartReg }}
      name: some-chart
      version: ${{ chartFrom(chartReg + "/some-chart").Version }}
# Render manifests to ./out, commit, push, etc...
```

</TabItem>

</Tabs>

### `helm-update-chart` Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the change(s) applied by this step. Typically, a subsequent [`git-commit`](#git-commit) step will reference this output and aggregate this commit message fragment with other like it to build a comprehensive commit message that describes all changes. |

## `helm-template`

`helm-template` renders a specified Helm chart to a specified directory or to
many files in a specified directory. This step is useful for the common scenario
of rendering Stage-specific manifests to a Stage-specific branch. This step is
commonly preceded by a [`git-clear`](#git-clear) step and followed by
[`git-commit`](#git-commit) and [`git-push`](#git-push) steps.

### `helm-template` Configuration

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

### `helm-template` Examples

<Tabs groupId="kustomize-build">

<TabItem value="file" label="Rendering to a File" default>

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

</TabItem>

<TabItem value="dir" label="Rendering to a Directory">

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

</TabItem>

</Tabs>

## `git-commit`

`git-commit` commits all changes in a working tree to its checked out branch.
This step is often used after previous steps have put the working tree into the
desired state and is commonly followed by a [`git-push`](#git-push) step.

### `git-commit` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing changes to be committed. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `message` | `string` | N | The commit message. Mutually exclusive with `messageFromSteps`. |
| `messageFromSteps` | `[]string` | N | References the `commitMessage` output of previous steps. When one or more are specified, the commit message will be constructed by concatenating the messages from individual steps. Mutually exclusive with `message`. |
| `author` | `[]object` | N | Optionally provider authorship information for the commit. |
| `author.name` | `string` | N | The committer's name. |
| `author.email` | `string` | N | The committer's email address. |

### `git-commit` Example

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
  as: update-image
  config:
    images:
    - image: my/image
- uses: kustomize-build
  config:
    path: ./src/stages/${{ ctx.stage }}
    outPath: ./out
- uses: git-commit
  config:
    path: ./out
    messageFromSteps:
    - update-image
# Push, etc...
```

### `git-commit` Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the commit created by this step. If the step short-circuited and did not create a new commit because there were no differences from the current head of the branch, this value will be the ID of the existing commit at the head of the branch instead. Typically, a subsequent [`argocd-update`](#argocd-update) step will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

## `git-push`

`git-push` pushes the committed changes in a specified working tree to a
specified branch in the remote repository. This step typically follows a `git-commit` step and is often followed by a `git-open-pr` step.

### `git-push` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to a Git working tree containing committed changes. |
| `targetBranch` | `string` | N | The branch to push to in the remote repository. Mutually exclusive with `generateTargetBranch=true`. If neither of these is provided, the target branch will be the same as the branch currently checked out in the working tree. |
| `generateTargetBranch` | `boolean` | N | Whether to push to a remote branch named like `kargo/<project>/<stage>/promotion`. If such a branch does not already exist, it will be created. A value of 'true' is mutually exclusive with `targetBranch`. If neither of these is provided, the target branch will be the currently checked out branch. This option is useful when a subsequent promotion step will open a pull request against a Stage-specific branch. In such a case, the generated target branch pushed to by the `git-push` step can later be utilized as the source branch of the pull request. |

### `git-push` Examples

<Tabs groupId="git-push">

<TabItem value="common" label="Common Usage" default>

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: git-push
  config:
    path: ./out
```

</TabItem>

<TabItem value="pr" label="For Use With a PR">

```yaml
steps:
# Clone, prepare the contents of ./out, etc...
- uses: git-commit
  config:
    path: ./out
    message: rendered updated manifests
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
# Open a PR and wait for it to be merged or closed...
```

</TabItem>

</Tabs>

### `git-push` Output

| Name | Type | Description |
|------|------|-------------|
| `branch` | `string` | The name of the remote branch pushed to by this step. This is especially useful when the `generateTargetBranch=true` option has been used, in which case a subsequent [`git-open-pr`](#git-open-pr) will typically reference this output to learn what branch to use as the head branch of a new pull request. |
| `commit` | `string` | The ID (SHA) of the commit pushed by this step. |

## `git-open-pr`

`git-open-pr` opens a pull request in a specified remote repository using
specified source and target branches. This step is often used after a `git-push`
and is commonly followed by a `git-wait-for-pr` step.

At present, this feature only supports GitHub pull requests and GitLab merge
requests.

### `git-open-pr` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github` and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.  |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `sourceBranch` | `string` | N | Specifies the source branch for the pull request. Mutually exclusive with `sourceBranchFromStep`. |
| `sourceBranchFromStep` | `string` | N | Indicates the source branch should be determined by the `branch` key in the output of a previous promotion step with the specified alias. Mutually exclusive with `sourceBranch`.<br/><br/>__Deprecated: Use `sourceBranch` with an expression instead. Will be removed in v1.3.0.__  |
| `targetBranch` | `string` | N | The branch to which the changes should be merged. |
| `createTargetBranch` | `boolean` | N | Indicates whether a new, empty orphaned branch should be created and pushed to the remote if the target branch does not already exist there. Default is `false`. |

### `git-open-pr`  Example

```yaml
steps:
# Clone, prepare the contents of ./out, commit, etc...
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: https://github.com/example/repo.git
    createTargetBranch: true
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
# Wait for the PR to be merged or closed...
```

### `git-open-pr` Output

| Name | Type | Description |
|------|------|-------------|
| `prNumber` | `number` | The numeric identifier of the pull request opened by this step. Typically, a subsequent [`git-wait-for-pr`](#git-wait-for-pr) step will reference this output to learn what pull request to monitor. |

## `git-wait-for-pr`

`git-wait-for-pr` waits for a specified open pull request to be merged or
closed. This step commonly follows a `git-open-pr` step and is commonly followed
by an `argocd-update` step.

### `git-wait-for-pr` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repoURL` | `string` | Y | The URL of a remote Git repository. |
| `provider` | `string` | N | The name of the Git provider to use. Currently only `github` and `gitlab` are supported. Kargo will try to infer the provider if it is not explicitly specified.  |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when interfacing with the Git provider. Setting this to `true` is highly discouraged in production. |
| `prNumber` | `string` | N | The number of the pull request to wait for. Mutually exclusive with `prNumberFromStep`. |
| `prNumberFromStep` | `string` | N | References the `prNumber` output from a previous step. Mutually exclusive with `prNumber`.<br/><br/>__Deprecated: Use `prNumber` with an expression instead. Will be removed in v1.3.0.__ |

### `git-wait-for-pr` Output

| Name | Type | Description |
|------|------|-------------|
| `commit` | `string` | The ID (SHA) of the new commit at the head of the target branch after merge. Typically, a subsequent [`argocd-update`](#argocd-update) step will reference this output to learn the ID of the commit that an applicable Argo CD `ApplicationSource` should be observably synced to under healthy conditions. |

### `git-wait-for-pr` Example

```yaml
steps:
# Clone, prepare the contents of ./out, commit, etc...
- uses: git-push
  as: push
  config:
    path: ./out
    generateTargetBranch: true
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: https://github.com/example/repo.git
    createTargetBranch: true
    sourceBranch: ${{ outputs.push.branch }}
    targetBranch: stage/${{ ctx.stage }}
- uses: git-wait-for-pr
  as: wait-for-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumber: ${{ outputs['open-pr'].prNumber }}
```

## `argocd-update`

`argocd-update` updates one or more Argo CD `Application` resources in various
ways. Among other scenarios, this step is useful for the common one of forcing
an Argo CD `Application` to sync after previous steps have updated a remote
branch referenced by the `Application`. This step is commonly the last step in a
promotion process.

:::note
For an Argo CD `Application` resource to be managed by a Kargo `Stage`,
the `Application` _must_ have an annotation of the following form:

```yaml
kargo.akuity.io/authorized-stage: "<project-name>:<stage-name>"
```

Such an annotation offers proof that a user who is themselves authorized
to update the `Application` in question has consented to a specific
`Stage` updating the `Application` as well.

The following example shows how to configure an Argo CD `Application`
manifest to authorize the `test` `Stage` of the `kargo-demo` `Project`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kargo-demo-test
  namespace: argocd
  annotations:
    kargo.akuity.io/authorized-stage: kargo-demo:test
spec:
  # Application specifications go here
```
:::

### `argocd-update` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apps` | `[]object` | Y | Describes Argo CD `Application` resources to update and how to update them. At least one must be specified.  |
| `apps[].name` | `string` | Y | The name of the Argo CD `Application`. __Note:__ A small technical restriction on this field is that any [expressions](./20-expression-language.md) used therein are limited to accessing `ctx` and `vars` and may not access `secrets` or any Freight. This is because templates in this field are, at times, evaluated outside the context of an actual `Promotion` for the purposes of building an index. In practice, this restriction does not prove to be especially limiting. |
| `apps[].namespace` | `string` | N | The namespace of the Argo CD `Application` resource to be updated. If left unspecified, the namespace will be the Kargo controller's configured default -- typically `argocd`. __Note:__ This field is subject to the same restrictions as the `name` field. See above. |
| `apps[].sources` | `[]object` | N | Describes Argo CD `ApplicationSource`s to update and how to update them. |
| `apps[].sources[].repoURL` | `string` | Y | The value of the target `ApplicationSource`'s  own `repoURL` field. This must match exactly. |
| `apps[].sources[].chart` | `string` | N | Applicable only when the target `ApplicationSource` references a Helm chart repository, the value of the target `ApplicationSource`'s  own `chart` field. This must match exactly. |
| `apps[].sources[].desiredRevision` | `string` | N | Specifies the desired revision for the source. i.e. The revision to which the source must be observably synced when performing a health check. This field is mutually exclusive with `desiredCommitFromStep`. Prior to v1.1.0, if both were left undefined, the desired revision was determined by Freight (if possible). Beginning with v1.1.0, if both are left undefined, Kargo will not require the source to be observably synced to any particular source to be considered healthy. Note that the source's `targetRevision` will not be updated to this revision unless `updateTargetRevision=true` is also set. |
| `apps[].sources[].desiredCommitFromStep` | `string` | N | Applicable only when `repoURL` references a Git repository, this field references the `commit` output from a previous step and uses it as the desired revision for the source. i.e. The revision to which the source must be observably synced when performing a health check. This field is mutually exclusive with `desiredRevisionFromStep`. Prior to v1.1.0, if both were left undefined, the desired revision was determined by Freight (if possible). Beginning with v1.1.0, if both are left undefined, Kargo will not require the source to be observably synced to any particular source to be considered healthy. Note that the source's `targetRevision` will not be updated to this commit unless `updateTargetRevision=true` is also set.<br/><br/>__Deprecated: Use `desiredRevision` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].updateTargetRevision` | `boolean` | Y | Indicates whether the target `ApplicationSource` should be updated such that its `targetRevision` field points directly at the desired revision. A `true` value in this field has no effect if `desiredRevision` and `desiredCommitFromStep` are both undefined. |
| `apps[].sources[].kustomize` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Kustomize-specific properties. |
| `apps[].sources[].kustomize.images` | `[]object` | Y | Describes how to update an Argo CD `ApplicationSource`'s Kustomize-specific properties to reference specific versions of container images. |
| `apps[].sources[].kustomize.images[].repoURL` | `string` | Y | URL of the image being updated. |
| `apps[].sources[].kustomize.images[].tag` | `string` | N | A tag naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `digest` and `useDigest=true`. One of `digest`, `tag`, or `useDigest=true` must be specified. |
| `apps[].sources[].kustomize.images[].digest` | `string` | N | A digest naming a specific revision of the image specified by `repoURL`. Mutually exclusive with `tag` and `useDigest=true`. One of `digest`, `tag`, or `useDigest=true` must be specified. |
| `apps[].sources[].kustomize.images[].useDigest` | `boolean` | N | Whether to use the container image's digest instead of its tag. |
| `apps[].sources[].kustomize.images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |
| `apps[].sources[].kustomize.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].kustomize.fromOrigin`. <br/><br/>__Deprecated: Use `digest` or `tag` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].kustomize.fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].fromOrigin`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].sources[].helm` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Helm parameters. |
| `apps[].sources[].helm.images` | `[]object` | Y | Describes how to update  an Argo CD `ApplicationSource`'s Helm parameters to reference specific versions of container images. |
| `apps[].sources[].helm.images[].repoURL` | `string` | N | URL of the image being updated. __Deprecated: Use `value` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].helm.images[].key` | `string` | Y | The key to update within the target `ApplicationSource`'s `helm.parameters` map. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `apps[].sources[].helm.images[].value` | `string` | Y | Specifies how the value of `key` is to be updated. When `repoURL` is non-empty, possible values for this field are limited to:<ul><li>`ImageAndTag`: Replaces the value of `key` with a string in form `<image url>:<tag>`</li><li>`Tag`: Replaces the value of `key` with the image's tag</li><li>`ImageAndDigest`: Replaces the value of `key` with a string in form `<image url>@<digest>`</li><li>`Digest`: Replaces the value of `key` with the image's digest</li></ul> When `repoURL` is empty, use an expression in this field to describe the new value.  |
| `apps[].sources[].helm.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].helm.fromOrigin`. <br/><br/>__Deprecated: Use `value` with an expression instead. Will be removed in v1.3.0.__ |
| `apps[].sources[].helm.fromOrigin` | `object` | N | See [specifying origins].(#specifying-origins). If not specified, may inherit a value from `apps[].sources[]`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].sources[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].fromOrigin`. <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `apps[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `fromOrigin`.  <br/><br/>__Deprecated: Will be removed in v1.3.0.__ |
| `fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). <br/><br/>__Deprecated: Will be removed in v1.3.0.__  |

### `argocd-update` Examples

<Tabs groupId="argocd-update">

<TabItem value="common" label="Common Usage" default>

```yaml
steps:
# Clone, render manifests, commit, push, etc...
- uses: git-commit
  as: commit
  config:
    path: ./out
    messageFromSteps:
    - update-image
- uses: git-push
  config:
    path: ./out
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredCommit: ${{ outputs.commit.commit }}
```

</TabItem>

<TabItem value="target-revision" label="Updating Target Revision">

:::caution
Without making any modifications to a Git repository, this example simply
updates a "live" Argo CD `Application` resource to point its `targetRevision`
field at a specific version of a Helm chart, which Argo CD will pull directly
from the the chart repository.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed
up in a Git repository. If the `Application` resource were deleted, there would
be no remaining record of its desired state.
:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: ${{ chartRepo }}
        chart: my-chart
        targetRevision: ${{ chartFrom(chartRepo, "my-chart").Version }}
```

</TabItem>

<TabItem value="updating-image-kustomize" label="Updating an Image with Kustomize">

:::caution
Without making any modifications to a Git repository, this example simply
updates Kustomize-specific properties of a "live" Argo CD `Application`
resource.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed up
in a Git repository. If the `Application` resource were deleted, there would be
no remaining record of its desired state.
:::

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        kustomize:
          images:
          - repoURL: ${{ vars.imageRepo }}
            tag: ${{ imageFrom(vars.imageRepo).Tag }}
```

</TabItem>

<TabItem value="updating-image-helm" label="Updating an Image with Helm">

:::caution
Without making any modifications to a Git repository, this example simply
updates Helm-specific properties of a "live" Argo CD `Application` resource.

While this can be a useful technique, it should be used with caution. This is
not "real GitOps" since the state of the `Application` resource is not backed
up in a Git repository. If the `Application` resource were deleted, there would
be no remaining record of its desired state.
:::

```yaml
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        helm:
          images:
          - key: image.tag
            value: ${{ imageFrom("my/image").Tag }}
```

</TabItem>

</Tabs>

### `argocd-update` Health Checks

The `argocd-update` step is unique among all other built-in promotion steps in
that, on successful completion, it will register health checks to be performed
upon the target Stage on an ongoing basis. This health check configuration is
_opaque_ to the rest of Kargo and is understood only by health check
functionality built into the step. This permits Kargo to factor the health and
sync state of Argo CD `Application` resources into the overall health of a Stage
without requiring Kargo to understand `Application` health directly.

:::info
Although the `argocd-update` step is the only promotion step to currently
utilize this health check framework, we anticipate that future built-in and
third-party promotion steps will take advantage of it as well.
:::

## `http`

`http` is a generic step that makes an HTTP/S request to enable basic integration
with a wide variety of external services.

### `http` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `method` | `string` | Y | The HTTP method to use. |
| `url` | `string` | Y | The URL to which the request should be made. |
| `headers` | `[]object` | N | A list of headers to include in the request. |
| `headers[].name` | `string` | Y | The name of the header. |
| `headers[].value` | `string` | Y | The value of the header. |
| `queryParams` | `[]object` | N | A list of query parameters to include in the request. |
| `queryParams[].name` | `string` | Y | The name of the query parameter. |
| `queryParams[].value` | `string` | Y | The value of the query parameter. The provided value will automatically be URL-encoded if necessary. |
| `body` | `string` | N | The body of the request. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when making the request. Setting this to `true` is highly discouraged. |
| `timeoutSeconds` | `number` | N | The maximum number of seconds to wait for a request to complete. If this is not specified, the default timeout is 10 seconds. The minimum permissible value is `1` and the maximum is `60`. _This is the timeout for an individual HTTP request. If a request is retried, each attempt is independently subject to this timeout._ |
| `successExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine success. If this is left undefined and `failureExpression` _is_ defined, the default success criteria will be the inverse of the specified failure criteria. If both are left undefined, success is `true` when the HTTP status code is `2xx`. If `successExpression` and `failureExpression` are both defined and both evaluate to `true`, the failure takes precedence. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `failureExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine failure. If this is left undefined and `successExpression` _is_ defined, the default failure criteria will be the inverse of the specified success criteria. If both are left undefined, failure is `true` when the HTTP status code is _not_ `2xx`. If `successExpression` and `failureExpression` are both defined and both evaluate to `true`, the failure takes precedence. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `outputs` | `[]object` | N | A list of rules for extracting outputs from the HTTP response. These are only applied to responses deemed successful. |
| `outputs[].name` | `string` | Y | The name of the output. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract a value from the HTTP response. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |

:::note
An HTTP response that is not conclusively determined to have succeeded or failed
will result in the step reporting a result of `Running`. Kargo will retry such
a step on its next attempt at reconciling the `Promotion` resource. This will
continue until the step succeeds, fails, exhausts the configured maximum number
of retries, or a configured timeout has elapsed.
:::

### `http` Expressions

The `successExpression`, `failureExpression`, and `outputs[].fromExpression`
fields all support [expr-lang] expressions.

:::note
The expressions included in the `successExpression`, `failureExpression`, and
`outputs[].fromExpression` fields should _not_ be offset by `${{` and `}}`. This
is to prevent the expressions from being evaluated by Kargo during
pre-processing of step configurations. The `http` step itself will evaluate
these expressions.
:::

A `response` object (a `map[string]any`) is available to these expressions. It
is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `status` | `int` | The HTTP status code of the response. |
| `headers` | `http.Header` | The headers of the response. See applicable [Go documentation](https://pkg.go.dev/net/http#Header). |
| `header` | `func(string) string` | `headers` can be inconvenient to work with directly. This function allows you to access a header by name. |
| `body` | `map[string]any` | The body of the response, if any, unmarshaled into a map. If the response body is empty, this map will also be empty. |

### `http` Examples

<Tabs groupId="http">

<TabItem value="basic" label="Basic Usage" default>

This examples configuration makes a `GET` request to the
[Cat Facts API.](https://www.catfacts.net/api/) and uses the default
success/failure criteria.

```yaml
steps:
# ...
- uses: http
  as: cat-facts
  config:
    method: GET
    url: https://www.catfacts.net/api/
    outputs:
    - name: status
      fromExpression: response.status
    - name: fact1
      fromExpression: response.body.facts[0]
    - name: fact2
      fromExpression: response.body.facts[1]
```

Assuming a `200` response with the following JSON body:

```json
{
    "facts": [
        {
            "fact_number": 1,
            "fact": "Kittens have baby teeth, which are replaced by permanent teeth around the age of 7 months."
        },
        {
            "fact_number": 2,
            "fact": "Each day in the US, animal shelters are forced to destroy 30,000 dogs and cats."
        }
    ]
}
```

The step would succeed and produce the following outputs:

```yaml
| Name     | Type | Value |
|----------|------|-------|
| `status` | `int` | `200` |
| `fact1` | `string` | `Kittens have baby teeth, which are replaced by permanent teeth around the age of 7 months.` |
| `fact2` | `string` | `Each day in the US, animal shelters are forced to destroy 30,000 dogs and cats.` |
```

</TabItem>

<TabItem value="polling" label="Polling">

Building on the basic example, this configuration defines explicit success and
failure criteria. Any response meeting neither of these criteria will result in
the step reporting a result of `Running` and being retried.

```yaml
steps:
# ...
- uses: http
  as: cat-facts
  config:
    method: GET
    url: https://www.catfacts.net/api/
    successExpression: response.status == 200
    failureExpression: response.status == 404
    outputs:
    - name: status
      fromExpression: response.status
    - name: fact1
      fromExpression: response.body.facts[0]
    - name: fact2
      fromExpression: response.body.facts[1]
```

Our request is considered:

- Successful if the response status is `200`.
- A failure if the response status is `404`.
- Running if the response status is anything else. i.e. Any other status code
  will result in a retry.

</TabItem>

<TabItem value="slack" label="Posting to Slack">

This examples is adapted from
[Slack's own documentation](https://api.slack.com/tutorials/tracks/posting-messages-with-curl):

```yaml
steps:
# ...
- uses: http
  config:
    method: POST
    url: https://slack.com/api/chat.postMessage
    headers:
    - name: Authorization
      value: Bearer ${{ secrets.slack.token }}
    - name: Content-Type
      value: application/json
    body: |
      {
        "channel": ${{ vars.slackChannel }},
        "blocks": [
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": "Hi I am a bot that can post *_fancy_* messages to any public channel."
            }
          }
        ]
      }
```

</TabItem>

</Tabs>

### `http` Outputs

The `http` step only produces the outputs described by the `outputs` field of
its configuration.

[expr-lang]: https://expr-lang.org/
