---
description: Learn how to work effectively with Warehouses
sidebar_label: Working with Warehouses
---

# Working with Warehouses

Kargo `Warehouse` resources each manage subscriptions to one or more of various types of artifact sources, including:

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

A `Warehouse` resource's most important field is its `spec.subscriptions` field,
which is used to subscribe to one or more artifact sources.

Here's an example of a `Warehouse` subscribing to both a
container image repository and a Git repository:

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

:::info
Kargo uses [semver](https://github.com/masterminds/semver#checking-version-constraints) to handle semantic versioning constraints.
:::

### Container Image Subscriptions

For subscriptions to container image repositories, the `imageSelectionStrategy`
field specifies a method for selecting the desired image. The available
strategies are:

- `Digest`: This strategy is used when subscribing to a specific mutable tag, such as `latest`, which is generally
    discouraged due to best practices favoring immutable tags. Users must supply a value in the `constraint` field,
    specifying the mutable tag they wish to track. The strategy will retrieve the latest details for the image
    tagged in this way, including any new or updated digest.

- `Lexical`: This strategy selects the image with the lexicographically greatest tag, making it suitable
    for scenarios where tags incorporate date/time stamps in formats like `yyyymmdd`. When using this
    strategy, it's recommended to pair it with a regular expression in the `allowTags` field to limit
    eligibility to tags that match the expected format, ensuring the correct selection of tags.

- `NewestBuild`: This strategy selects the image with the most recent build time.

    :::warning
    `NewestBuild` requires retrieving metadata for every eligible tag, which can be slow and is likely to
    exceed the registry's rate limits. It's advisable to use the `allowTags` field to limit
    the number of tags for which metadata is retrieved, thereby reducing the risk of hitting rate limits.
    :::

- `SemVer`: This strategy selects the image that best matches a semantic versioning constraint.

    :::info
    Kargo uses the [semver](https://github.com/masterminds/semver) package for
    parsing and comparing semantic versions and semantic version constraints.
    Refer to
    [these docs](https://github.com/masterminds/semver#checking-version-constraints)
    for detailed information on version constraint syntax.
    :::

### Git Repository Subscriptions

For subscriptions to Git repositories, the `commitSelectionStrategy`
field specifies a method for selecting the desired commit. The available
strategies are:

The available strategies for subscribing to a Git repository are:

- `Lexical`: Selects the commit referenced by the lexicographically greatest tag.
    It is particularly useful in scenarios where commit references, such as tags or branches,
    incorporate date/time stamps in formats like `yyyymmdd`.
    To ensure the correct selection, it's advisable to use regular expressions in the
    `allowTags` or `allowBranches` field, which limit the acceptable format of the references,
    preventing the selection of undesired tags like `zzz-custom` over something like `nightly-20241211`.

- `NewestFromBranch`: Selects the most recent commit from a specified branch. It's useful when tracking the latest changes in a branch that receives regular updates.

- `NewestTag`: Selects the most recent commit associated with a tag. Since tags are typically immutable,
    there should be only one commit per tag.
    To optimize this strategy, it's recommended to constrain the eligible tags using regular expressions or specific patterns,
    ensuring the selection is limited to tags that follow a consistent naming convention.

- `SemVer`: Selects the commit referenced by a *tag* that best matches the constraint.

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
