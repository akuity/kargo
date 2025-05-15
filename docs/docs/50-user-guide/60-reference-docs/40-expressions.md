---
sidebar_label: Expressions
description: Learn about expression language support in promotion step configurations
---

# Expression Language Reference

The [steps](15-promotion-templates.md#steps) of a user-defined promotion process
as well as a `Stage`'s
[verification arguments](../20-how-to-guides/60-verification.md#arguments-and-metadata)
may take advantage of expressions in their configuration.

:::info
The documentation on this page assumes a general familiarity with the concepts
of Promotion Templates and Analysis Templates, as well as some knowledge of how
a promotion process is defined as a sequence of discrete steps and how
verification is defined in a `Stage`.

For more information on Promotion Templates, refer to the
[Promotion Templates Reference](15-promotion-templates.md).

For detailed coverage of individual promotion steps, refer to the
[Promotion Steps Reference](30-promotion-steps/index.md).

For information on Analysis Templates, refer to the
[Verification Guide](../20-how-to-guides/60-verification.md) and
[Analysis Templates Reference](50-analysis-templates.md).
:::

## Syntax

All steps in a user-defined promotion processes (i.e. those described by a
[Promotion Template](15-promotion-templates.md) and
[PromotionTasks](20-promotion-tasks.md)) support the use of
an [Expression Language](https://expr-lang.org) as a means of dynamically resolving
values in their configuration at promotion time.

In addition, [`Stage` verification arguments](../20-how-to-guides/60-verification.md#arguments-and-metadata)
may also use expressions to inject dynamic values into the `AnalysisRun` that
is created from an `AnalysisTemplate`.

All expressions must be enclosed within the `${{` and `}}` delimiters. This is
not universally true for all applications of expr-lang. Kargo selected these
specific delimiters to mimic GitHub Actions expression syntax, which many users
will already be familiar with.

Basic example:

```yaml
config:
  message: ${{ "Hello, world!" }}
```

The above example will be evaluated as the following:

```yaml
config:
  message: Hello, world!
```

The
[expr-lang language definition docs](https://expr-lang.org/docs/language-definition)
provide a comprehensive overview of the language's syntax and capabilities, so
this reference will continue to focus only on Kargo-specific extensions and
usage.

## Structure and Behavior

### Config Blocks

In promotion steps, expressions appear within configuration blocks that can
have nested values. Kargo will evaluate expressions just-in-time as each step
of a promotion process is executed. It will _only_ evaluate expressions within
_values_ of a configuration block and will _not_ evaluate expressions within
keys. Expressions in values are evaluated recursively, so expressions may be
nested any number of levels deep within a configuration block.

```yaml
config:
  nested:
    value: ${{ foo.bar }}
  other: ${{ baz.qux }}
```

### Variables

In promotion step and verification arguments, expressions appear in a flat list
of argument name-value pairs. Each argument has a name and a single value that
can contain an expression. Unlike configuration blocks, these arguments do not
support nested values.

### Validation

Kargo parses configuration blocks _before_ evaluating expressions, so any
configuration containing expressions _must_ be well-formed YAML even prior to
evaluation. Further validation (e.g. for adherence to a step-specific schema) is
performed only _after_ expressions are evaluated.

### Types

Due to the requirement that configuration blocks be well-formed YAML, all fields
containing expressions must be `string`s. Internally, all expressions will also
evaluate to `string`s, however, Kargo will attempt to coerce the results to
other valid JSON types (YAML is a superset of JSON) including `object`, `array`,
`number`, `boolean`, and `null` before concluding that the evaluated expression
should continue to be treated as a `string`.

This behavior should be unsurprising and perhaps even familiar to experienced
YAML users, as YAML parsers behave in the same way. `42`, for example, is
interpreted as a JSON `number` unless it is explicitly quoted (i.e. `"42"`) to
specify that it should be interpreted as a `string`.

In practice, this means care should be taken to use Kargo's built-in `quote()`
function in cases where an evaluated expression may appear to be a `number` or
`boolean`, for instance, but should be treated as a `string`.

For example:

```yaml
config:
  numField: ${{ 40 + 2 }} # Will be treated as a number
  strField: ${{ quote(40 + 2) }} # Will be treated as a string
```

The above example will be evaluated to the following:

```yaml
config:
  numField: 42
  strField: "42"
```

## Pre-Defined Variables

Kargo provides a number of pre-defined variables that are accessible within
expressions. This section enumerates these variables, their structure, and use.

:::info
Expect other useful variables to be added in the future.
:::

### Promotion Variables

| Name | Type | Description |
|------|------|-------------|
| `ctx` | `object` | Contains contextual information about the promotion. See detailed structure below. |
| `outputs` | `object` | A map of output from previous promotion steps indexed by step aliases. |
| `secrets` | `object` | A map of maps indexed by the names of all Kubernetes `Secret`s in the `Promotion`'s `Project` and the keys within the `Data` block of each.<br/><br/>__Deprecated: Use the [`secret()` function](#secretname) instead. Will be removed in v1.7.0.__ |
| `vars` | `object` | A user-defined map of variable names to static values of any type. The map is derived from a `Promotion`'s `spec.promotionTemplate.spec.vars` field. Variable names must observe standard Go variable-naming rules. Variables values may, themselves, be defined using an expression. `vars` (contains previously defined variables) and `ctx` are available to expressions defining the values of variables, however, `outputs` and `secrets` are not. |
| `task` | `object` | A map containing output from previous steps within the same PromotionTask under the `outputs` field, indexed by step aliases. Only available within `(Cluster)PromotionTask` steps. |

#### Context (`ctx`) Object Structure

The `ctx` object has the following structure:

```
ctx
├── project: string       # The name of the Project
├── stage: string         # The name of the Stage
├── promotion: string     # The name of the Promotion
└── meta
    └── promotion
        └── actor: string # The creator of the Promotion
```

The following example promotion process clones a repository and checks out
two branches to different directories, uses Kustomize with source from one
branch to render some Kubernetes manifests that it commits to the other branch,
and pushes back to the repository. These steps make extensive use of the
pre-defined variables `ctx`, `outputs`, and `vars`.

```yaml
promotionTemplate:
  spec:
    vars:
    - name: gitRepo
      value: https://github.com/example/repo.git
    - name: srcPath
      value: ./src
    - name: outPath
      value: ./out
    - name: targetBranch
      value: stage/${{ ctx.stage }}
    steps:
    - uses: git-clone
      config:
        repoURL: ${{ vars.gitRepo }}
        checkout:
        - fromFreight: true
          path: ${{ vars.srcPath }}
        - branch: ${{ vars.targetBranch }}
          create: true
          path: ${{ vars.outPath }}
    - uses: git-clear
      config:
        path: ${{ vars.outPath }}
    - uses: kustomize-set-image
      as: update-image
      config:
        path: ${{ vars.srcPath }}/base
        images:
        - image: public.ecr.aws/nginx/nginx
    - uses: kustomize-build
      config:
        path: ${{ vars.srcPath }}/stages/${{ ctx.stage }}
        outPath: ${{ vars.outPath }}
    - uses: git-commit
      as: commit
      config:
        path: ${{ vars.outPath }}
        message: ${{ outputs['update-image'].commitMessage }}
    - uses: git-push
      config:
        path: ${{ vars.outPath }}
        targetBranch: ${{ vars.targetBranch }}
    - uses: argocd-update
      config:
        apps:
        - name: example-${{ ctx.stage }}
          sources:
          - repoURL: ${{ vars.gitRepo }}
            desiredRevision: ${{ outputs.commit.commit }}
```

:::info
Since the usage of expressions and pre-defined variables effectively
parameterizes the promotion process, the same promotion process can be reused in
other `Projects` or `Stages` with few, if any, modifications (other than the
definition of the static variables).
:::

### Verification Variables

| Name | Type | Description |
|------|------|-------------|
| `ctx` | `object` | Contains contextual information about the stage. See structure below. |

#### Context (`ctx`) Object Structure for Verification

The `ctx` object for verification has the following structure:

```
ctx
├── project: string  # The name of the Project
└── stage: string    # The name of the Stage
```

## Functions

Several functions are built-in to Kargo's expression language. This section
describes each of them.

### `quote(value)`

The `quote()` function converts any value to its string representation. It has
one required argument:

- `value` (Required): A value of any type to be converted to a string.

This is useful for scenarios where an expression evaluates to a non-`string`
JSON type, but you wish to treat it as a `string` regardless. For string inputs,
it produces clean output without visible quotation marks.

Example:

```yaml
config:
  numField: ${{ 40 + 2 }} # Will be treated as a number (42)
  strField: ${{ quote(40 + 2) }} # Will be treated as a string ("42")
  rawField: ${{ quote("string") }} # Will result in "string"
```

### `unsafeQuote(value)`

The `unsafeQuote()` function converts any value to its string representation. It
has one required argument:

- `value` (Required): A value of any type to be converted to a string.

Compared to [`quote()`](#quotevalue) this function is considered "unsafe"
because it adds escaped quotes around input values that are already considered
strings. For non-string values, it behaves similarly to `quote()`.

Example:

```yaml
config:
  numField: ${{ 40 + 2 }} # Will be treated as a number (42)
  strField: ${{ unsafeQuote(40 + 2) }} # Will result in "42"
  rawField: ${{ unsafeQuote("string") }} # Will result in "\"string\""
```

### `configMap(name)`

The `configMap()` function returns the `Data` field (a `map[string]string`) of a
Kubernetes `ConfigMap` with the specified name from the `Project`'s namespace.
If no such `ConfigMap` exists, an empty map is returned.

Example:

```yaml
config:
  repoURL: ${{ configMap('my-config').repoURL }}
```

### `secret(name)`

The `secret()` function returns the `Data` field of a Kubernetes `Secret` with
the specified name from the `Project`'s namespace decoded into a
`map[string]string`. If no such `Secret` exists, an empty map is returned.

Examples:

```yaml
config:
  headers:
  - name: Authorization
    value: Bearer ${{ secret('slack').token }}
```

### `warehouse(name)`

The `warehouse()` function returns a `FreightOrigin` object representing a
`Warehouse`. It has one required argument:

- `name` (Required): A string representing the name of a `Warehouse` resource
  in the same `Project` as the `Promotion` being executed.

The returned `FreightOrigin` object has the following fields:

| Field | Description |
|-------|-------------|
| `Kind` | The kind of the `FreightOrigin`. Always equals `Warehouse` for this function. |
| `Name` | The name of the `Warehouse` resource. |

The `FreightOrigin` object can be used as an optional argument to the
`commitFrom()`, `imageFrom()`, or `chartFrom()` functions to disambiguate the
desired source of an artifact when necessary. These functions return `nil` when
relevant `Freight` is not found from the `FreightCollection`. 

:::tip
You can handle `nil` values gracefully in Expr using its
[nil coalescing](https://expr-lang.org/docs/language-definition#nil-coalescing) and
[optional chaining](https://expr-lang.org/docs/language-definition#optional-chaining) features.
:::

### `commitFrom(repoURL, [freightOrigin])`

The `commitFrom()` function returns a corresponding `GitCommit` object from the
`Promotion` or `Stage` their `FreightCollection`. It has one required and one
optional argument:

- `repoURL` (Required): The URL of a Git repository.
- `freightOrigin` (Optional): A `FreightOrigin` object (obtained from
  [`warehouse()`](#warehousename)) to specify which `Warehouse` should provide
  the commit information.

The returned `GitCommit` object has the following fields:

| Field | Description |
|-------|-------------|
| `RepoURL` | The URL of the Git repository the commit originates from. |
| `ID`      | The ID of the Git commit. |
| `Branch`  | The branch of the repository where this commit was found. Only present if the `Warehouse`'s Git subscription is configured to track branches. |
| `Tag`     | The tag of the repository where this commit was found. Only present if the `Warehouse`'s Git subscription is configured to track tags. |
| `Message` | The first line of the commit message (up to 80 characters). |
| `Author` | The name and email address of the commit author. |
| `Committer` | The name and email address of the committer. |

The optional `freightOrigin` argument should be used when a `Stage` requests
`Freight` from multiple origins (`Warehouse`s) and more than one can provide a
`GitCommit` object from the specified repository.

If a commit is not found from the `FreightCollection`, returns `nil`.

Examples:

```yaml
config:
  commitID: ${{ commitFrom("https://github.com/example/repo.git").ID }}
```

```yaml
config:
  commitID: ${{ commitFrom("https://github.com/example/repo.git", warehouse("my-warehouse")).ID }}
```

### `imageFrom(repoURL, [freightOrigin])`

The `imageFrom()` function returns a corresponding `Image` object from the
`Promotion` or `Stage` their `FreightCollection`. It has one required and
one optional argument:

- `repoURL` (Required): The URL of a container image repository.
- `freightOrigin` (Optional): A `FreightOrigin` object (obtained from
  [`warehouse()`](#warehousename)) to specify which `Warehouse` should provide
  the image information.

The returned `Image` object has the following fields:

| Field | Description |
|-------|-------------|
| `RepoURL` | The URL of the container image repository the image originates from. |
| `GitRepoURL` | (Deprecated as of version 1.5, will be removed in version 1.7) The URL of the Git repository which contains the source code for the image. Only present if Kargo was able to infer it from the URL. |
| `Tag` | The tag of the image. |
| `Digest` | The digest of the image. |
| `Annotations` | A map of [annotations](https://specs.opencontainers.org/image-spec/annotations/) discovered for the image. |

The optional `freightOrigin` argument should be used when a `Stage` requests
`Freight` from multiple origins (`Warehouse`s) and more than one can provide a
`Image` object from the specified repository.

If an image is not found from the `FreightCollection`, returns `nil`.

Examples:

```yaml
config:
  imageTag: ${{ imageFrom("public.ecr.aws/nginx/nginx").Tag }}
```

```yaml
config:
  imageTag: ${{ imageFrom("public.ecr.aws/nginx/nginx", warehouse("my-warehouse")).Tag }}
```

### `chartFrom(repoURL, [chartName], [freightOrigin])`

The `chartFrom()` function returns a corresponding `Chart` object from the
`Promotion` or `Stage` their `FreightCollection`. It has one required and two
optional arguments:

- `repoURL` (Required): The URL of a Helm chart repository.
- `chartName` (Optional): The name of the chart (required for HTTP/S
  repositories, not needed for OCI registries).
- `freightOrigin` (Optional): A `FreightOrigin` object (obtained from
  [`warehouse()`](#warehousename)) to specify which `Warehouse` should provide
  the chart information.

The returned `Chart` object has the following fields:

| Field | Description |
|-------|-------------|
| `RepoURL` | The URL of the Helm chart repository the chart originates from. For HTTP/S repositories, this is the URL of the repository. For OCI repositories, this is the URL of the container image repository including the chart's name. |
| `Name` | The name of the Helm chart. Only present for HTTP/S repositories. |
| `Version` | The version of the Helm chart. |

For Helm charts stored in OCI registries, the URL should be the full path to
the repository within that registry.

For Helm charts stored in classic (HTTP/S) repositories, which can store
multiple different charts within a single repository, the `chartName` argument
must be provided to specify the name of the chart within the repository.

The optional `freightOrigin` argument should be used when a `Stage` requests
`Freight` from multiple origins (`Warehouse`s) and more than one can provide a
`Chart` object from the specified repository.

If a chart is not found from the `FreightCollection`, returns `nil`.

Examples:

```yaml
# OCI registry
config:
  chartVersion: ${{ chartFrom("oci://example.com/my-chart").Version }}
```

```yaml
# OCI registry with specific warehouse
config:
  chartVersion: ${{ chartFrom("oci://example.com/my-chart", warehouse("my-warehouse")).Version }}
```

```yaml
# HTTP/S repository
config:
  chartVersion: ${{ chartFrom("https://example.com/charts", "my-chart").Version }}
```

```yaml
# HTTP/S repository with specific warehouse
config:
  chartVersion: ${{ chartFrom("https://example.com/charts", "my-chart", warehouse("my-warehouse")).Version }}
```
