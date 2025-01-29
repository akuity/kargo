---
description: Learn how to work effectively with Warehouses
sidebar_label: Working with Warehouses
---

# Working with Warehouses

Kargo `Warehouse` resources each manage subscriptions to one or more of various
types of artifact sources, including:

- Container image repositories
- Git repositories
- Helm chart repositories

When a `Warehouse` observes a new revision of any artifact to which it
subscribes, it creates a new `Freight` resource representing a specific
collection of artifact revisions that can be promoted from `Stage` to `Stage`
_as a unit_.

:::info
For a broader, conceptual understanding of warehouses and their relation
to other Kargo concepts, refer to 
[Core Concepts](./../10-core-concepts/index.md).
:::

## The `Warehouse` Resource Type

A `Warehouse`'s subscriptions are all defined within its `spec.subscriptions`
field.

In this example, a `Warehouse` subscribes to both a container image repository
and a Git repository:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - image:
      repoURL: public.ecr.aws/nginx/nginx
      semverConstraint: ^1.26.0
  - git:
      repoURL: https://github.com/example/kargo-demo.git
```

The remainder of this document focuses on the configuration of the individual
subscription types.

### Container Image Subscriptions

#### Image Selection Strategies

For subscriptions to container image repositories, the `imageSelectionStrategy`
field specifies a method for selecting the desired image. The available
strategies are:

- `SemVer`: Selects the image with the tag that best matches a semantic
  versioning constraint. All tags that are not valid semantic versions are
  ignored. With no constraint specified, the strategy simply selects the image
  with the semantically greatest tag.

    __`SemVer`__ is the default strategy if one is not specified. 

    :::info
    Kargo uses the [semver](https://github.com/masterminds/semver) package for
    parsing and comparing semantic versions and semantic version constraints.
    Refer to
    [these docs](https://github.com/masterminds/semver#checking-version-constraints)
    for detailed information on version constraint syntax.
    :::

    :::warning
    TODO: Add an example
    :::

- `Lexical`: This strategy selects the image with the lexicographically greatest
   tag.

   This is useful in scenarios wherein tags incorporate date/time stamps in
   formats such as `yyyymmdd` and you wish to select the tag with the latest
   stamp. When using this strategy, it is recommended to use the `allowTags`
   field to limit eligibility to tags that match the expected format.

    :::warning
    TODO: Add an example
    :::

- `Digest`: This selects the image _currently_ referenced by some "mutable tag,"
   such as `latest`. The tag name must be specified using the `constraint`
   field. Importantly, the _digest_ will change every time the tag is updated.

    :::warning
    "Mutable tags": Tags like `latest` that are sometimes, perhaps frequently,
    updated to point to a different, presumably newer image.

    "Immutable tags": Tags that have version or date information embedded within
    them, along with an expectation of never being updated to reference a
    different image.

    Using mutable tags like `latest` _is a widely discouraged practice._
    Whenever possible, it is recommended to use immutable tags.
    :::

    :::warning
    TODO: Add an example
    :::

- `NewestBuild`: This strategy selects the image with the most recent build
  time.

    :::warning
    `NewestBuild` requires retrieving metadata for every eligible tag, which can
    be slow and is likely to exceed the registry's rate limits. __This can
    result in system-wide performance degradation.__

    If using this strategy is unavoidable, it is recommended to use the
    `allowTags` field to limit the number of tags for which metadata is
    retrieved to reduce the risk of encountering rate limits. `allowTags` may
    require periodic adjustment as a repository grows.
    :::

    :::warning
    TODO: Add an example
    :::

### Git Repository Subscriptions

#### Commit Selection Strategies

For subscriptions to Git repositories, the `commitSelectionStrategy`
field specifies a method for selecting the desired commit. The available
strategies are:

- `NewestFromBranch`: Selects the most recent commit from a specified branch.
  
    This is useful for the average case, wherein you wish for the `Warehouse` to
    continuously discover the latest changes to a branch that receives regular
    updates.

    `NewestFromBranch` is the default strategy if one is not specified.

    :::warning
    TODO: Add an example
    :::

- `SemVer`: Selects the commit referenced by the tag that best matches a
  semantic versioning constraint. All tags that are not valid semantic versions
  are ignored. With no constraint specified, the strategy simply selects the
  commit referenced by the semantically greatest tag.

    This is useful in scenarios wherein you do not wish for the `Warehouse` to
    continuously discover _every new commit_ and would like limit selection to
    commits tagged with a semantic version, and possibly within a certain range.

    :::info
    Kargo uses the [semver](https://github.com/masterminds/semver) package for
    parsing and comparing semantic versions and semantic version constraints.
    Refer to
    [these docs](https://github.com/masterminds/semver#checking-version-constraints)
    for detailed information on version constraint syntax.
    :::

    :::warning
    TODO: Add an example
    :::

- `Lexical`: Selects the commit referenced by the lexicographically greatest
  tag.

    This is useful in scenarios wherein you do not wish for the `Warehouse` to
    discover _every new commit_ and tags incorporate date/time stamps in formats
    such as `yyyymmdd` and you wish to select the tag with the latest stamp.
    When using this strategy, it is recommended to use the `allowTags` field to
    limit eligibility to tags that match the expected format.

    :::warning
    TODO: Add an example
    :::

- `NewestTag`: Selects the commit referenced by the most recently created tag.
  
    When using this strategy, it is recommended to use the `allowTags` field to
    limit eligibility to tags that match the expected format.

    :::warning
    TODO: Add an example
    :::

#### Git Subscription Path Filtering

In some cases, it may be necessary to constrain the paths within a Git
repository that a `Warehouse` will consider as triggers for `Freight`
production. This is especially useful for GitOps repositories that are
"monorepos" containing configuration for multiple applications.

The paths that may or must not trigger `Freight` production may be specified
using a combination of the `includePaths` and `excludePaths` fields of a Git
repository subscription.

The following example demonstrates a `Warehouse` with a Git repository
subscription that will only produce new `Freight` when the latest commit
(selected by the applicable commit selection strategy) contains changes in the
`apps/guestbook` directory since the last piece of `Freight` produced by the
`Warehouse`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/kargo-demo.git
      includePaths:
      - apps/guestbook
```

The next example demonstrates the opposite: a `Warehouse` with a Git repository
subscription that will only produce new `Freight` when the latest commit
(selected by the applicable commit selection strategy) contains changes to paths
_other than_ the repository's `docs/` directory:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/kargo-demo.git
      excludePaths:
      - docs
```

`includePaths` and `excludePaths` may be combined to include a broad set of
paths and then exclude a subset of those. The following example demonstrates a
`Warehouse` with a Git repository subscription that will only produce new
`Freight` when the latest commit (selected by the applicable commit selection
strategy) contains changes _within_ the `apps/guestbook` directory _other than_
the `apps/guestbook/README.md`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/kargo-demo.git
      includePaths:
      - apps/guestbook
      excludePaths:
      - apps/guestbook/README.md
```

:::note
It is important to understand that new `Freight` will be produced when the
latest commit (selected by the applicable commit selection strategy) contains
_even a single change_ that is:

1. Implicitly included via undefined `includePaths`.

    OR

    Explicitly included via `includePaths`.

    AND

2. Not explicitly excluded via `excludePaths`.
:::

:::note
By default, the strings in the `includePaths` and `excludePaths` fields are
treated as exact paths to files or directories. (Selecting a directory will
implicitly select all paths within that directory.)

Paths may _also_ be specified using glob patterns (by prefixing the string with
`glob:`) or regular expressions (by prefixing the string with `regex:` or
`regexp:`).
:::

### Helm Chart Repository Subscriptions
