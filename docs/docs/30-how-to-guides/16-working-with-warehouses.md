---
description: Learn how to work effectively with Warehouses
sidebar_label: Working with Warehouses
---

# Working with Warehouses

Each Kargo warehouse is represented by a Kubernetes resource of type
`Warehouse`.

## The `Warehouse` Resource Type

A `Warehouse` resource's most important field is its `spec.subscriptions` field,
which is used to subscribe to one or more:

* Container image repositories

* Git repositories

* Helm charts repositories

The following example shows a `Warehouse` resource that subscribes to a
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

### Image Subscription

For subscriptions to container image repositories, the `imageSelectionStrategy` field specifies the method for selecting
the desired image. The available strategies for subscribing to an image repository are:

- `Digest`: Selects the image with the specified digest.
- `Lexical`: Selects the image with the lexicographically greatest tag.
- `NewestBuild`: Selects the image with the most recent build time.
- `SemVer`: Selects the image that best matches a semantic versioning constraint.

### Git Subscription

In subscriptions to Git repositories, the `commitSelectionStrategy` field
specifies the method for selecting the desired commit.
The available strategies for subscribing to a git repository are:

- `Lexical`: Selects the commit with the lexicographically greatest reference.
- `NewestFromBranch`: Selects the most recent commit from a specified branch.
- `NewestTag`: Selects the most recent commit associated with a tag.
- `SemVer`: Selects the commit that best matches a semantic versioning constraint.

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

:::info
By default, the strings in the `includePaths` and `excludePaths` fields are
treated as exact paths to files or directories. (Selecting a directory will
implicitly select all paths within that directory.)

Paths may _also_ be specified using glob patterns (by prefixing the string with
`glob:`) or regular expressions (by prefixing the string with `regex:` or
`regexp:`).
:::
