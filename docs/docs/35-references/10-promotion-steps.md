---
sidebar_label: Promotion Steps Reference
description: Learn about all of Kargo's built-in promotion steps
---

# Promotion Steps Reference

This reference document describes the promotion steps that are built directly
into Kargo. Steps are presented roughly in the order in which they might appear
in a typical promotion process. Similarly, configuration options for each step
are laid out in order of their applicability to typical use cases.

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
| `checkout[].branch` | `string` | N | A branch to check out. Mutually exclusive with `tag` and `fromFreight=true`. If none of these is specified, the default branch will be checked out. |
| `checkout[].create` | `boolean` | N | In the event `branch` does not already exist on the remote, whether a new, empty, orphaned branch should be created. Default is `false`, but should commonly be set to `true` for Stage-specific branches, which may not exist yet at the time of a Stage's first promotion. |
| `checkout[].tag` | `string` | N | A tag to check out. Mutually exclusive with `branch` and `fromFreight=true`. If none of these is specified, the default branch will be checked out. |
| `checkout[].fromFreight` | `boolean` | N | Whether a commit to check out should be obtained from the Freight being promoted. A value of `true` is mutually exclusive with `branch` and `tag`. If none of these is specified, the default branch will be checked out. Default is `false`, but is often set to `true`. |
| `checkout[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). |
| `checkout[].path` | `string` | Y | The path for a working tree that will be created from the checked out revision. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |

### `git-clone` Examples

<Tabs groupId="git-clone-examples">

<TabItem value="common" label="Common Usage" default>

The most common usage of this step is to check out a commit specified by the
Freight being promoted as well as a Stage-specific branch. Subsequent steps are
likely to perform actions that revise the contents of the Stage-specific branch
using the commit from the Freight as input.

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: base
      path: ./src
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: test-overlay
      path: ./overlay
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/test/kustomization.yaml
    outPath: ./src/stages/test/kustomization.yaml
- uses: kustomize-build
  config:
    path: ./src/stages/test
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
    - branch: stage/test
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: base
      path: ./src
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: test-overlay
      path: ./overlay
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/test/kustomization.yaml
    outPath: ./src/stages/test/kustomization.yaml
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
| `images[].image` | `string` | Y | Name/URL of the image being updated. The Freight being promoted presumably contains a reference to a revision of this image. |
| `images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). |
| `images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |
| `images[].useDigest` | `boolean` | N | Whether to update the `kustomization.yaml` file using the container image's digest instead of its tag. |

### `kustomize-set-image` Examples

<Tabs groupId="kustomize-set-image">

<TabItem value="common" label="Common Usage" default>

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: base
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-set-image
  config:
    path: ./src/base
    images:
    - image: my/image
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: base
      path: ./src
    - branch: stage/test
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

### `kustomize-build` Examples

<Tabs groupId="kustomize-build">

<TabItem value="file" label="Rendering to a File" default>

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-build
  config:
    path: ./src/stages/test
    outPath: ./out/manifests.yaml
# Commit, push, etc...
```

</TabItem>

<TabItem value="dir" label="Rendering to a Directory" default>

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: kustomize-build
  config:
    path: ./src/stages/test
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
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
| `charts[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins) |

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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: https://example-chart-repo
      name: some-chart
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-update-chart
  config:
    path: ./src/charts/my-chart
    charts:
    - repository: oci://example-chart-registry
      name: some-chart
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-template
  config:
    path: ./src/charts/my-chart
    valuesFiles:
    - ./src/charts/my-chart/test-values.yaml
    outPath: ./out/manifests.yaml
# Commit, push, etc...
```

</TabItem>

<TabItem value="dir" label="Rendering to a Directory">

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: helm-template
  config:
    path: ./src/charts/my-chart
    valuesFiles:
    - ./src/charts/my-chart/test-values.yaml
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
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      path: ./src
    - branch: stage/test
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
    path: ./src/stages/test
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
| `sourceBranchFromStep` | `string` | N | Indicates the source branch should be determined by the `branch` key in the output of a previous promotion step with the specified alias. Mutually exclusive with `sourceBranch`. |
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
    sourceBranchFromStep: push
    targetBranch: stage/prod
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
| `prNumberFromStep` | `string` | N | References the `prNumber` output from a previous step. Mutually exclusive with `prNumber`. |

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
    sourceBranchFromStep: push
    targetBranch: stage/prod
- uses: git-wait-for-pr
  as: wait-for-pr
  config:
    repoURL: https://github.com/example/repo.git
    prNumberFromStep: open-pr
```

## `argocd-update`

`argocd-update` updates one or more Argo CD `Application` resources in various
ways. Among other scenarios, this step is useful for the common one of forcing
an Argo CD `Application` to sync after previous steps have updated a remote
branch referenced by the `Application`. This step is commonly the last step in a
promotion process.

### `argocd-update` Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apps` | `[]object` | Y | Describes Argo CD `Application` resources to update and how to update them. At least one must be specified.  |
| `apps[].name` | `string` | Y | The name of the Argo CD `Application`. |
| `apps[].namespace` | `string` | N | The namespace of the Argo CD `Application` resource to be updated. If left unspecified, the namespace will be the Kargo controller's configured default -- typically `argocd`. |
| `apps[].sources` | `[]object` | N | Describes Argo CD `ApplicationSource`s to update and how to update them. |
| `apps[].sources[].repoURL` | `string` | Y | The value of the target `ApplicationSource`'s  own `repoURL` field. This must match exactly. |
| `apps[].sources[].chart` | `string` | N | Applicable only when the target `ApplicationSource` references a Helm chart repository, the value of the target `ApplicationSource`'s  own `chart` field. This must match exactly. |
| `apps[].sources[].desiredCommitFromStep` | `string` | N | Applicable only when `repoURL` references a Git repository, this field references the `commit` output from a previous step and uses it as the desired revision for the source. If this is left undefined, the desired revision will be determined by Freight (if possible). Note that the source's `targetRevision` will not be updated to this commit unless `updateTargetRevision=true` is set. The utility of this field is to ensure that health checks on Argo CD `ApplicationSource`s can account for scenarios where the desired revision differs from what may be found in Freight, likely due to the use of rendered branches and/or PR-based promotion workflows. |
| `apps[].sources[].updateTargetRevision` | `boolean` | Y | Indicates whether the target `ApplicationSource` should be updated such that its `targetRevision` field points at the most recently Git commit (if `repoURL` references a Git repository) or chart version (if `repoURL` references a chart repository). |
| `apps[].sources[].kustomize` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Kustomize-specific properties. |
| `apps[].sources[].kustomize.images` | `[]object` | Y | Describes how to update an Argo CD `ApplicationSource`'s Kustomize-specific properties to reference specific versions of container images. |
| `apps[].sources[].kustomize.images[].repoURL` | `string` | Y | URL of the image being updated. The Freight being promoted must contain a reference to a revision of this image. |
| `apps[].sources[].kustomize.images[].newName` | `string` | N | A substitution for the name/URL of the image being updated. This is useful when different Stages have access to different container image repositories (assuming those different repositories contain equivalent images that are tagged identically). This may be a frequent consideration for users of Amazon's Elastic Container Registry. |
| `apps[].sources[].kustomize.images[].useDigest` | `boolean` | N | Whether to use the container image's digest instead of its tag. |
| `apps[].sources[].kustomize.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].kustomize.fromOrigin`. |
| `apps[].sources[].kustomize.fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].fromOrigin`.  |
| `apps[].sources[].helm` | `object` | N | Describes updates to an Argo CD `ApplicationSource`'s Helm parameters. |
| `apps[].sources[].helm.images` | `[]object` | Y | Describes how to update  an Argo CD `ApplicationSource`'s Helm parameters to reference specific versions of container images. |
| `apps[].sources[].helm.images[].repoURL` | `string` | Y | URL of the image being updated. The Freight being promoted must contain a reference to a revision of this image. |
| `apps[].sources[].helm.images[].key` | `string` | Y | The key to update within the target `ApplicationSource`'s `helm.parameters` map. See Helm documentation on the [format and limitations](https://helm.sh/docs/intro/using_helm/#the-format-and-limitations-of---set) of the notation used in this field. |
| `apps[].sources[].helm.images[].value` | `string` | Y | Specifies how the value of `key` is to be updated. Possible values for this field are limited to:<ul><li>`ImageAndTag`: Replaces the value of `key` with a string in form `<image url>:<tag>`</li><li>`Tag`: Replaces the value of `key` with the image's tag</li><li>`ImageAndDigest`: Replaces the value of `key` with a string in form `<image url>@<digest>`</li><li>`Digest`: Replaces the value of `key` with the image's digest</li></ul> |
| `apps[].sources[].helm.images[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].sources[].helm.fromOrigin` |
| `apps[].sources[].helm.fromOrigin` | `object` | N | See [specifying origins].(#specifying-origins). If not specified, may inherit a value from `apps[].sources[]`. |
| `apps[].sources[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `apps[].fromOrigin`. |
| `apps[].fromOrigin` | `object` | N | See [specifying origins](#specifying-origins). If not specified, may inherit a value from `fromOrigin`. |
| `fromOrigin` | `object` | N | See [specifying origins](#specifying-origins) |

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
        desiredCommitFromStep: commit
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
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://example-chart-repo
        chart: my-chart
        updateTargetRevision: true
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
steps:
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        kustomize:
          images:
          - repoURL: my/image
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
          - repoURL: my/image
            key: image.tag
            value: Tag
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

## Specifying Origins

Many promotion steps, or parts of those steps, will (whether optionally or
unconditionally) attempt to learn the desired revision(s) of some artifact(s) by
consulting the revisions of those artifact(s) references by the Freight being
promoted.

By way of example, this `kustomize-set-image` step will consult the Freight
being promoted to learn the desired revision of the `my/image` container image:

```yaml
- uses: kustomize-set-image
  config:
    path: ./src/base
    images:
    - image: my/image
```

In some _advanced_ uses cases, Stages may request Freight from multiple origins
(Warehouses). In such scenarios, it is possible (although somewhat rare) that
the multiple Freight being promoted may collectively reference multiple distinct
revisions of the same artifact. In such as case, it can become ambiguous which
revision of an artifact referenced by a promotion step should be used.

To permit disambiguation in cases such as those described above, all promotion
steps that have the potential to reference Freight from multiple origins support
a `fromOrigin` option that can be used to clarify which piece of Freight's
reference to the artifact should be used by identifying the origin (Warehouse)
from which the Freight should have originated.

The best way to illustrate this involves a complex example wherein a Stage
requests Freight from two Warehouses. Both Warehouses subscribe to the same Git
repository, with one watching for changes to a Kustomize "base" configuration
and the other watching for changes to a Stage-specific Kustomize overlay.
Rendering the manifests intended for such a Stage will require combining the
base and overlay configurations with the help of a [`copy`](#copy) step. For
this case, a `git-clone` step may be configured similarly to the following:

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: base
      path: ./src
    - fromFreight: true
      fromOrigin:
        kind: Warehouse
        name: test-overlay
      path: ./overlay
    - branch: stage/test
      create: true
      path: ./out
- uses: git-clear
  config:
    path: ./out
- uses: copy
  config:
    inPath: ./overlay/stages/test/kustomization.yaml
    outPath: ./src/stages/test/kustomization.yaml
- uses: kustomize-build
  config:
    path: ./src/stages/test
    outPath: ./out
# Commit, push, etc...
```

Note that when checking out specific revisions of the
`https://github.com/example/repo.git` repository to different working trees, the
`git-clone` step has twice utilized `fromOrigin` to clarify which of the Freight
being promoted should be used to determine the revision to check out.

:::info
`fromOrigin` never needs to be specified in the majority of use cases wherein
there is no inherent ambiguity. Kargo will automatically select the correct
revision of an artifact when there is only one possibility. When Kargo detects
that there may be multiple possibilities, it will fail and raise an error
indicating that the user must disambiguate by specifying `fromOrigin` in
applicable steps.
:::
