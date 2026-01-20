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
      constraint: ^1.26.0
  - git:
      repoURL: https://github.com/example/kargo-demo.git
```

The remainder of this section focuses on the configuration of the individual
subscription types.

### Container Image Subscriptions

Container image repository subscriptions can be defined using the following
fields:

- `repoURL`: The URL of the container image repository _without any tag_. This field is required.

- `imageSelectionStrategy`: One of four pre-defined strategies for selecting the
  desired image. (See next section.)

<a name="allow-tags-regexes-constraint"></a>

- `allowTagsRegexes`: An optional list of regular expressions that limit
  eligibility for selection to tags that match any of the patterns.

<a name="ignore-tags-regexes-constraint"></a>

- `ignoreTagsRegexes`: An optional list of regular expressions that limit
  eligibility for selection to tags that don't match any of the patterns.

<a name="platform-constraint"></a>

- `platform`: An optional identifier that constrains image selection to those
  images supporting the specified operating system and system architecture.
  e.g., `linux/amd64`.

  :::note

  It is seldom necessary to specify this field.
  :::

- `discoveryLimit`: Many selection strategies (see next section) do not actually
  select a _single_ image; rather they select the n best fits for the specified
  constraints. The _best_ fit is the zero element in the list of selected
  images. `discoveryLimit` specifies how many images to discover.

  The default is `20`.

  :::note

  For poorly performing `Warehouse`s -- for instance ones frequently
  encountering rate limits -- decreasing this limit may improve performance.
  :::

- `insecureSkipTLSVerify`: Set to `true` to disable validation of the
  repository's TLS certificate.

  :::warning

  This is a security risk and should only be used in development environments.
  :::

- `strictSemvers`: StrictSemvers specifies whether only "strict" semver tags
  should be considered. `StrictSemvers` specifies whether only "strict" semver
  tags should be considered. A "strict" semver tag is one containing ALL of
  major, minor, and patch version components. This is enabled by default, but
  only has any effect when the `ImageSelectionStrategy` is `SemVer`. This
  should be disabled cautiously, as it is not uncommon to tag container images
  with short Git commit hashes, which have the potential to contain numeric
  characters only and could be mistaken for a semver string containing the
  major version number only.

- `cacheByTag`: Set to `true` to enable more aggressive caching of image
  metadata using tags as keys. This can significantly reduce the number of API
  calls to the registry and improve performance.

  The default is `false`.

  :::warning[Use with caution!]

  This setting is safest if your tags are known to be "immutable" (i.e., tag
  always references the same image and is never updated to point to a different
  image).

  This setting does NOT apply to the `Digest` selection strategy, which assumes
  the one tag it subscribes to is a mutable one.

  :::

  :::warning

  Operators may also choose from a number of policies regarding the caching of
  image metadata using tags as keys. Some of these policies (`Forbid` and
  `Force`) can override an individual container image subscription's choice to
  cache metadata by tag or not. See
  [common configurations](../../40-operator-guide/20-advanced-installation/30-common-configurations.md) for further details.

  :::

#### Image Selection Strategies

For subscriptions to container image repositories, the `imageSelectionStrategy`
field specifies a method for selecting the desired image. The available
strategies are:

- `SemVer`: Selects the image with the tag that best matches a semantic
  versioning constraint specified by the `constraint` field. If no such
  constraint is specified, the strategy simply selects the image with the
  semantically greatest tag. All tags that are not valid semantic versions are
  ignored.

  The `strictSemvers` field defaults to `true`, meaning only tags containing
  all three parts of a semantic version (major, minor, and patch) are
  considered. Disabling this should be approached with caution because any
  image tagged only with decimal characters will be considered a valid semantic
  version (containing only the major element).

  **`SemVer` is the default strategy if one is not specified.**

  :::info

  Kargo uses the [semver](https://github.com/masterminds/semver) package for
  parsing and comparing semantic versions and semantic version constraints.
  Refer to
  [these docs](https://github.com/masterminds/semver#checking-version-constraints)
  for detailed information on version constraint syntax.
  :::

  Example:

  ```yaml
  spec:
    subscriptions:
    - image:
        repoURL: public.ecr.aws/nginx/nginx
        constraint: ^1.26.0
  ```

- `Lexical`: This strategy selects the image with the lexicographically greatest
  tag.

  This is useful in scenarios wherein tags incorporate date/time stamps in
  formats such as `yyyymmdd` and you wish to select the tag with the latest
  stamp. When using this strategy, it is recommended to use the
  `allowTagsRegexes` field to limit eligibility to tags matching specific
  patterns.

  Example:

  ```yaml
  spec:
    subscriptions:
    - image:
        repoURL: public.ecr.aws/nginx/nginx
        imageSelectionStrategy: Lexical
        allowTagsRegexes:
        - ^nightly-\d{8}$
  ```

- `Digest`: This selects the image _currently_ referenced by some "mutable tag"
  (such as `latest`) specified by the `constraint` field.

  :::warning

  "Mutable tags": Tags like `latest` that are sometimes, perhaps frequently,
  updated to point to a different, presumably newer image.

  "Immutable tags": Tags that have version or date information embedded within
  them, along with an expectation of never being updated to reference a
  different image.

  Using mutable tags like `latest` _is a widely discouraged practice._
  Whenever possible, it is recommended to use immutable tags.
  :::

  Example:

  ```yaml
  spec:
    subscriptions:
    - image:
        repoURL: public.ecr.aws/nginx/nginx
        imageSelectionStrategy: Digest
        constraint: latest
  ```

<a name="newest-build"></a>

- `NewestBuild`: This strategy selects the image with the most recent build
  time.

  The build time is evaluated using the labels
  `org.opencontainers.image.created` or `org.label-schema.build-date`. If
  neither label is set, Kargo will fall back to using the `config.Created` time
  of the image.

  :::warning

  `NewestBuild` requires retrieving metadata for every eligible tag, which can
  be slow and is likely to exceed the registry's rate limits. **This can
  result in system-wide performance degradation.**

  If using this strategy is unavoidable, it is recommended to use the
  `allowTagsRegexes` field to limit the number of tags for which metadata is
  retrieved to reduce the risk of encountering rate limits. `allowTagsRegexes`
  may require periodic adjustment as a repository grows.
  :::

  ```yaml
  spec:
    subscriptions:
    - image:
        repoURL: public.ecr.aws/nginx/nginx
        imageSelectionStrategy: NewestBuild
        allowTagsRegexes:
        - ^nightly
  ```

  :::tip

  If your tags are known to be **immutable** (i.e., a tag always references the
  same image and is never updated to point to a different image), you can use
  the `cacheByTag` field to enable more aggressive caching of image metadata by
  tag. This can significantly reduce the number of API calls to the registry and
  improve performance.

  ```yaml
  spec:
    subscriptions:
    - image:
        repoURL: public.ecr.aws/nginx/nginx
        imageSelectionStrategy: NewestBuild
        cacheByTag: true
        allowTagsRegexes:
        - ^nightly
  ```

  :::

  :::warning[Use with caution!]

  Only enable `cacheByTag` if you are certain that all relevant tags are
  **immutable**. Using this with mutable tags (like `latest`) can cause Kargo
  to select stale images indefinitely.

  :::

### Git Repository Subscriptions

Git repository subscriptions can be defined using the following fields:

- `repoURL`: The URL of the Git repository. This field is required.

- `commitSelectionStrategy`: One of four pre-defined strategies for selecting
  the desired commit. (See next section.)

- `allowTagsRegexes`: An optional list of regular expressions that limit
  eligibility for selection to tags that match any of the patterns. (This is not
  applicable to selection strategies that do not involve tags.)

- `ignoreTagsRegexes`: An optional list of regular expressions that limit
  eligibility for selection to tags that don't match any of the patterns. (This
  is not applicable to selection strategies that do not involve tags.)

- `expressionFilter`: An optional expression that filters commits and tags based
  on their metadata. See [Expression Filtering](#expression-filtering) for
  details.

- `includePaths`: See
  [Git Subscription Path Filtering](#git-subscription-path-filtering).

- `excludePaths`: See
  [Git Subscription Path Filtering](#git-subscription-path-filtering).

- `discoveryLimit`: Many selection strategies (see next section) do not actually
  select a _single_ commit; rather they select the n best fits for the specified
  constraints. The _best_ fit is the zero element in the list of selected
  commits. `discoveryLimit` specifies how many commits to discover.

  The default is `20`.

  :::note

  Lowering this limit for a Git repository subscription does not improve
  performance by the margins that it does for a container image repository
  subscription.
  :::

- `insecureSkipTLSVerify`: Set to `true` to disable validation of the
  repository's TLS certificate.

  :::warning

  This is a security risk and should only be used in development environments.
  :::

#### Commit Selection Strategies

For subscriptions to Git repositories, the `commitSelectionStrategy`
field specifies a method for selecting the desired commit. The available
strategies are:

- `NewestFromBranch`: Selects the most recent commit from a branch specified
  by the `branch` field. If a branch is not specified, the strategy selects
  commits from the repository's default branch (typically `main` or `master`).

  This is useful for the average case, wherein you wish for the `Warehouse` to
  continuously discover the latest changes to a branch that receives regular
  updates.

  **`NewestFromBranch` is the default selection strategy if one is not
  specified.**

  Example:

  ```yaml
  spec:
    subscriptions:
    - git:
        repoURL: https://github.com/example/repo.git
        branch: main
  ```

- `SemVer`: Selects the commit referenced by the tag that best matches a
  semantic versioning constraint. All tags that are not valid semantic versions
  are ignored. With no constraint specified, the strategy simply selects the
  commit referenced by the semantically greatest tag.

  This is useful in scenarios wherein you do not wish for the `Warehouse` to
  continuously discover _every new commit_ and would like limit selection to
  commits tagged with a semantic version, and possibly within a certain range.

  The `strictSemvers` field defaults to `true`, meaning only tags containing
  all three parts of a semantic version (major, minor, and patch) are
  considered. Disabling this should be approached with caution because any
  image tagged only with decimal characters will be considered a valid
  semantic version (containing only the major element).

  :::info

  Kargo uses the [semver](https://github.com/masterminds/semver) package for
  parsing and comparing semantic versions and semantic version constraints.
  Refer to
  [these docs](https://github.com/masterminds/semver#checking-version-constraints)
  for detailed information on version constraint syntax.
  :::

  Example:

  ```yaml
  spec:
    subscriptions:
    - git:
        repoURL: https://github.com/example/repo.git
        commitSelectionStrategy: SemVer
        semverConstraint: ^1.0.0
  ```

- `Lexical`: Selects the commit referenced by the lexicographically greatest
  tag.

  This is useful in scenarios wherein you do not wish for the `Warehouse` to
  discover _every new commit_ and tags incorporate date/time stamps in formats
  such as `yyyymmdd` and you wish to select the tag with the latest stamp. When
  using this strategy, it is recommended to use the `allowTagsRegexes` field to
  limit eligibility to tags matching specific patterns.

  Example:

  ```yaml
  spec:
    subscriptions:
    - git:
        repoURL: https://github.com/example/repo.git
        commitSelectionStrategy: Lexical
        allowTagsRegexes:
        - ^nightly-\d{8}$
  ```

- `NewestTag`: Selects the commit referenced by the most recently created tag.

  When using this strategy, it is recommended to use the `allowTagsRegexes`
  field to limit eligibility to tags matching specific patterns.

  Example:

  ```yaml
  spec:
    subscriptions:
    - git:
        repoURL: https://github.com/example/repo.git
        commitSelectionStrategy: NewestTag
        allowTagsRegexes:
        - ^nightly
  ```

#### Expression Filtering

Git repository subscriptions support advanced filtering using expressions. These
expressions allow you to filter commits and tags based on their metadata using
[expr-lang](https://expr-lang.org) syntax.

:::info

The expressions must evaluate to a boolean value (`true` or `false`). If an
expression evaluates to a non-boolean value, an attempt will be made to
convert it to a boolean (e.g., `0` to `false`, `1` to `true`).

:::

:::warning

Invalid expressions will cause the subscription to fail. Always test your
expressions to ensure they evaluate correctly with your repository's data.

:::

:::info

You can test your expressions using the
[expr-lang playground](https://expr-lang.org/playground).

The playground allows you to evaluate expressions against sample data and
see the results in real-time. This is especially useful for debugging and
validating your expressions before applying them to your `Warehouse` resources.

:::

The `expressionFilter` field provides a unified way to filter commits or tags
based on the selected commit selection strategy. The behavior and available
variables depend on your `commitSelectionStrategy`:

**For commit-based filtering** (`NewestFromBranch` strategy):

- Filters commits based on commit metadata
- Applied when selecting the newest commit from a branch

**For tag-based filtering** (`SemVer`, `Lexical`, and `NewestTag` strategies):

- Filters tags based on name and associated commit metadata
- Applied after `allowTagsRegexes`, `ignoreTagsRegexes` and `semverConstraint`
  fields

##### Available Expression Filtering Variables

The variables available in your expression depend on the commit selection
strategy:

**For `NewestFromBranch` (commit filtering):**

- `id`: The ID (SHA) of the commit
- `commitDate`: The date of the commit
- `author`: The author of the commit, in format `Name <email>`
- `committer`: The committer of the commit, in format `Name <email>`
- `subject`: The first line of the commit message

**For `SemVer`, `Lexical`, and `NewestTag` (tag filtering):**

- `tag`: The name of the tag
- `id`: The commit ID that the tag references
- `creatorDate`: The tag creation date (annotated tag) or commit date
  (lightweight tag)
- `author`: The author of the commit that the tag references, in the format of
  `Name <email>`
- `committer`: The committer of the commit that the tag references, in the
  format of `Name <email>`
- `subject`: The first line of the commit message associated with the tag
- `tagger`: The tagger of the tag, in the format of `Name <email>`. Only
  available for annotated tags.
- `annotation`: The first line of the tag annotation. Only available for
  annotated tags.

##### Expression Filtering Examples

**Filtering commits by excluding bot authors:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: NewestFromBranch
      expressionFilter: !(author contains '<bot@example.com>')
```

**Filtering commits with specific message patterns:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: NewestFromBranch
      expressionFilter: subject contains 'feat:' || subject contains 'fix:'
```

**Filtering commits with multiple criteria:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: NewestFromBranch
      expressionFilter: !(author == 'Example Bot') && commitDate.After(date('2025-01-01'))
```

**Filtering commits to exclude those with ignore markers:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: NewestFromBranch
      expressionFilter: !(subject contains '[kargo-ignore]')
```

**Filtering tags by author name:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: SemVer
      expressionFilter: author == 'John Doe <john@example.com>'
```

**Filtering tags created after a specific date:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: NewestTag
      expressionFilter: creatorDate.Year() >= 2024
```

**Filtering tags to exclude those committed by bots:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: Lexical
      expressionFilter: !(committer contains '<bot@example.com>')
```

**Filtering tags with complex conditions:**

```yaml
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/repo.git
      commitSelectionStrategy: SemVer
      expressionFilter: creatorDate.After(date('2024-01-01')) && !(tag contains 'alpha')
```

#### Git Subscription Path Filtering

In some cases, it may be necessary to constrain the paths within a Git
repository that a `Warehouse` will consider as triggers for `Freight`
production. This is especially useful for GitOps repositories that are
["monorepos"](../30-patterns/index.md#monorepo-layout) containing configuration
for multiple applications.

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

1. Not explicitly excluded via `excludePaths`.

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

Helm chart repository subscriptions can be defined using the following fields:

- `repoURL`: The URL of the Helm chart repository. This field is required.

  Chart repositories using http/s may contain versions of many _different_
  charts. Subscriptions to all chart repositories using http/s **must**
  additionally specify the chart's name in the `name` field.

  For chart repositories in OCI registries, the repository URL points only to
  revisions of a _single_ chart. Subscriptions to chart repositories in OCI
  registries **must** leave the `name` field empty.

- `name`: See above.

- `semverConstraint`: Selects the chart version best matching this constraint.
  If left unspecified, the subscription implicitly selects the semantically
  greatest version of the chart.

  :::info

  Helm _requires_ charts to be semantically versioned.
  :::

  :::info

  Kargo uses the [semver](https://github.com/masterminds/semver) package for
  parsing and comparing semantic versions and semantic version constraints.
  Refer to
  [these docs](https://github.com/masterminds/semver#checking-version-constraints)
  for detailed information on version constraint syntax.
  :::

- `discoveryLimit`: A chart repository subscription does not actually select a
  _single_ chart version; rather it selects the n best fits for the specified
  constraints. The _best_ fit is the zero element in the list of selected
  charts. `discoveryLimit` specifies how many chart versions to discover.

  The default is `20`.

  Example:

  ```yaml
  spec:
    subscriptions:
    - chart:
        repoURL: https://charts.example.com
        name: my-chart
        semverConstraint: ^1.0.0
  ```

## Working with Private Repositories

Frequently, `Warehouse`s require access to private repositories, in which case
appropriate credentials must be made available in some form. The many available
authentication options are covered in detail on the
[Managing Secrets](../50-security/30-managing-secrets.md) page.

## Automatic Freight Creation

By default, `Warehouse`s create new `Freight` following a discovery cycle if
new artifacts are found. This can be disabled by setting `freightCreationPolicy`
to `Manual`.

Example:

```yaml
spec:
  freightCreationPolicy: Manual
```

For more granular control over the creation of your `Freight`, you can define
`Freight` creation criteria in the form of an expression. One potential use case
could be back end and front end versions needing to match.

Example:

```yaml
spec:
  freightCreationPolicy: Automatic
  subscriptions:
  - image:
      repoURL: ghcr.io/example/frontend
  - image:
      repoURL: ghcr.io/example/backend
  freightCreationCriteria:
    expression: |
      imageFrom('ghcr.io/example/frontend.git').Tag == imageFrom('ghcr.io/example/backend.git').Tag
```

For more information on `Freight Creation Criteria` refer to the
[Expression Language Reference](../60-reference-docs/40-expressions.md).

## Performance Considerations

### Polling Frequency

`Warehouse` resources periodically poll the repositories to which they subscribe
in an attempt to discover new artifact revisions. By default, and under nominal
conditions, this discovery process occurs at an interval configured at the
system-level, however, the effective interval can be much longer if the system
is under heavy load or `Warehouse`s are poorly configured.

:::info

Discovery of container images, in particular, can be time-consuming.

Both the [`NewestBuild` selection strategy](#newest-build) and any
[`platform` constraints](#platform-constraint) are heavily dependent on the
retrieval of image metadata for every image in the repository not eliminated
from consideration up-front by other, more efficient constraints such as
[`allowTagsRegexes`](#allow-tags-regexes-constraint) or
[`ignoreTagsRegexes`](#ignore-tags-regexes-constraint). Registry architecture,
unfortunately, requires such metadata be retrieved image-by-image with a
separate API call for each. Even with aggressive caching, and especially when
the number of image revisions to consider is large, this process can take quite
some time. The time required to complete discovery can be protracted even
further if the registry's rate limit has been exceeded.

Kargo can execute a finite number of these discovery processes concurrently and
registries enforce rate limits on the basis of your public IP or, if applicable,
your credentials. (i.e. Rate limits are not enforced on a
`Warehouse`-by-`Warehouse` basis. Registries know nothing about your
`Warehouse`s.)

**Due to the above, even a well-tuned `Warehouse` that avoids inefficient image
selection criteria may experience large intervals between executions of its
discovery process (or slow discovery) if _other_ `Warehouse`s are configured
inefficiently.**

:::

With the goal of less frequent polling to reduce load on registries, avoid
encountering rate limits, and reduce occurrences of discovery running for a
prolonged period, only to find no new artifacts, you may wish to configure
your `Warehouse` resources to execute artifact discovery less frequently than
the system-wide default. (i.e. You may wish to _increase_ the polling interval.)
This can be done by tuning the `spec.interval` field of any `Warehouse`.

:::note

The effective polling interval is the _greater_ of a system-wide minimum and
any interval specified by `spec.interval`. i.e. You can configure a `Warehouse`
to execute its artifact discovery process _less_ frequently than the system-wide
minimum, _but not more frequently._

:::

:::info

If you're an operator wishing to reduce the frequency with which _all_
`Warehouse`s execute their discovery processes (increase the minimum polling
interval), refer to the
[Common Configurations](../../40-operator-guide/20-advanced-installation/30-common-configurations.md#tuning-warehouse-reconciliation-intervals)
section of the of the Operator's Guide for more information.

:::

With reduced polling frequency, overall system performance may improve, but will
be accompanied by the undesired side effect of increasing the average time
required for `Warehouse`s to notice new artifacts (of any kind; not just
container images). _This can be overcome by configuring repositories to alert
Kargo to the presence of new artifacts via webhooks._

### Caching Image Metadata by Tag

When using image selection strategies that require fetching image metadata
(such as [`NewestBuild`](#newest-build) or when using
[`platform`](#platform-constraint) constraints), a significant bottleneck can be
the number of API calls required to the container image registry.

If your image tags are **guaranteed to be immutable** (i.e., a tag always
references the exact same image and is never updated to point to a different
image), you can enable the [`cacheByTag`](#cacheByTag) option on individual
image subscriptions:

```yaml
spec:
  subscriptions:
  - image:
      repoURL: ghcr.io/example/myapp
      imageSelectionStrategy: NewestBuild
      cacheByTag: true
      allowTagsRegexes:
      - ^v\d+\.\d+\.\d+$
```

This enables significantly more aggressive caching of image metadata, which can
reduce API calls and improve performance by orders of magnitude in repositories
with large numbers of tags.

:::warning[Use with caution!]

Only enable this option if your tags are known to be **immutable**
(i.e., a tag always references the same image and is never updated to point
to a different image).

This setting does not apply to the `Digest` selection strategy, which always
assumes tags are mutable.

:::

## Triggering Artifact Discovery Using Webhooks

Configuring Kargo to receive webhook payloads from popular Git hosting providers
and container image / OCI registry providers is easy, and can be configured
[globally by an operator](../../40-operator-guide/35-cluster-configuration.md#triggering-artifact-discovery-using-webhooks)
or at the Project level by a Project admin.

The remainder of this section focuses on configuring webhook receivers at the
Project level.

:::info[Not what you were looking for?]

If you're an operator looking to understand how you can configure Kargo to
listen for inbound webhook requests to trigger the discovery processes of all
applicable `Warehouse`s across all Projects, refer to the
[Cluster Level Configuration](../../40-operator-guide/35-cluster-configuration.md#triggering-artifact-discovery-using-webhooks)
section of the Operator's Guide.

:::

### Configuring a Receiver

Creating and configuring a webhook receiver at the Project level is accomplished
by updating your `ProjectConfig` resource's `spec.webhookReceivers` field. If
your Project does not already have a `ProjectConfig` resource, you can create
one.

:::note

Every Kargo Project is permitted to have at most _one_ `ProjectConfig` resource.
This limit is enforced by requiring all `ProjectConfig` resources to be named
_the same_ as the Project / `Project` resource / namespace to which they belong.

For a Project `kargo-demo`, for example, the corresponding `ProjectConfig` must
be contained within the Project's namespace (`kargo-demo`) and must, itself, be
named `kargo-demo`.

:::

A `ProjectConfig` resource's `spec.webhookReceivers` field may define one or
more _webhook receivers_. A webhook receiver is an endpoint on a (typically)
internet-facing HTTP server that is configured to receive and process requests
from specific sources, and in response, trigger the discovery process of any
`Warehouse` within the Project that subscribes to a repository URL referenced by
the request payload.

Most types of webhook receivers require you only to specify a unique (within the
Project) name and a reference to a `Secret`. The expected keys and values for
each kind of webhook receiver vary, and are documented on
[each receiver type's own page](../60-reference-docs/80-webhook-receivers/index.md).

:::info

`Secret`s referenced by a webhook receiver typically serve _two_ purposes.

1. _Often_, some value(s) from the `Secret`'s data map are shared with the
   webhook sender (GitHub, for instance) and used to help authenticate requests.
   Some senders may use such "shared secrets" as bearer tokens. Others may use
   them as keys for signing requests. In such cases, the corresponding webhook
   receiver knows exactly what to do with this information in order to
   authenticate inbound requests.

1. _Always_, some value(s) from the `Secret`'s data map are used as a seed in
   deterministically constructing a complex, hard-to-guess URL where the
   receiver will listen for inbound requests.

   Some webhook senders (Docker Hub, for instance), do not natively implement
   any sort of authentication mechanism. No secret value(s) need to be shared
   with such a sender and requests from the sender contain no bearer token, nor
   are they signed. For cases such as these, a hard-to-guess URL is, itself,
   a _de facto_ shared secret and authentication mechanism.

   **Note that if a `Secret`'s value(s) are rotated, the URL where the receiver
   listens for inbound requests will also change. This is by design.**

   Kargo does not watch `Secret`s for changes because it lacks the permissions
   to do so, so it can be some time _after_ its `Secret`'s value(s) are rotated
   that a webhook receiver's URL will be updated. To expedite that update, your
   `ProjectConfig` resource can be manually "refreshed" using the `kargo` CLI:

   ```shell
   kargo refresh projectconfig --project <project name>
   ```

:::

The following example `ProjectConfig` configures two webhook receivers:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers:
  - name: my-first-receiver
    github:
      secretRef:
        name: my-first-secret
  - name: my-second-receiver
    gitlab:
      secretRef:
        name: my-second-secret
---
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: my-first-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret: c295bGVudCBncmVlbiBpcyBwZW9wbGUK
---
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: my-second-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret-token: cm9zZWJ1ZCB3YXMgYSBzbGVkCg==
```

:::note

The `kargo.akuity.io/cred-type: generic` label on `Secret`s referenced by
webhook receivers is not strictly required, but we _strongly_ recommend
including it.

:::

For each properly configured webhook receiver, Kargo will update the
`ProjectConfig` resource's `status` to reflect the URLs that can be registered
as endpoints with the senders.

For instance, the `ProjectConfig` and `Secret`s above result in the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  # ... omitted for brevity ...
status:
  conditions:
  - lastTransitionTime: "2025-06-11T22:53:21Z"
    message: ProjectConfig is synced and ready for use
    observedGeneration: 1
    reason: Synced
    status: "True"
    type: Ready
  webhookReceivers:
  - name: my-first-receiver
    path: /webhook/github/804b6f6bb40eb1f0e371f971d71dd95549be4bc9cbf868046941115f44073c67
    url: https://kargo.example.com/webhook/github/804b6f6bb40eb1f0e371f971d71dd95549be4bc9cbf868046941115f44073c67
  - name: my-second-receiver
    path: /webhook/gitlab/0eba9ff2a91f04f7787404b8f8f0edaf8cf8c39add34082651a474803cc99015
    url: https://kargo.example.com/webhook/gitlab/0eba9ff2a91f04f7787404b8f8f0edaf8cf8c39add34082651a474803cc99015
```

Above, you can see the URLs that can be registered with GitHub and GitLab as
endpoints to receive webhook requests from those platforms.

:::info

For more information about registering these endpoints with specific senders,
refer to
[each receiver type's own page](../60-reference-docs/80-webhook-receivers/index.md).

:::

:::info

If you're working with a large number of Kargo Projects and/or repositories and
wish for `Warehouse`s in all Projects to execute their discovery processes in
response to applicable events, it will likely be impractical to configure webhook
receivers Project-by-Project.

Refer to
[Cluster Level Configuration](../../40-operator-guide/35-cluster-configuration.md#triggering-artifact-discovery-using-webhooks)
section of the Operator's Guide to learn how to register cluster-scoped webhook
receivers that can trigger discovery for all applicable `Warehouse`s across all
Projects.

:::

### Receivers in Action

Once a webhook receiver has been assigned a URL and that URL has been registered
with a compatible sender, the receiver will begin receiving webhook requests in
response to events in your repositories. The payload (body) of such a request
contains structured information (usually JSON) the sender wishes to share about
some event. Invariably, among this information, is the URL of the repository
from which the event originated.

A webhook receiver's only job is to extract a repository URL from the webhook
request's payload, query for all `Warehouse` resources within the Project having
subscriptions to that repository, and request each to execute their discovery
process.
