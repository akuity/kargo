---
sidebar_label: Expression Language Reference
description: Learn about expression language support in promotion step configurations
---

# Expression Language Reference

The steps of a user-defined promotion process may take advantage of expressions
in their configuration.

:::info
The documentation on this page assumes a general familiarity with the concept of
Promotions and some knowledge of how a promotion process is defined as a
sequence of discrete steps.

For an overview of Promotions, refer to the
[concepts doc](../concepts#promotions).

For detailed coverage of individual promotion steps, refer to the
[Promotion Steps Reference](./10-promotion-steps.md).
:::

## Syntax

All steps in a user-defined promotion processes (i.e. those described by a
`Stage` resource's `spec.promotionTemplate.spec.steps` field) support the use of
[expr-lang](https://expr-lang.org/) as a means of dynamically resolving values
in their configuration at promotion time.

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

## Behavior

Kargo will evaluate expressions just-in-time as each step of a promotion process
is executed. It will _only_ evaluate expressions within _values_ of a
configuration block and will _not_ evaluate expressions within keys. Expressions
in values are evaluated recursively, so expressions may be nested any number of
levels deep within a configuration block.

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

| Name | Type | Description |
|------|------|-------------|
| `ctx` | `object` | `string` fields `project`, `stage`, and `promotion` provide convenient access to details of a `Promotion`. |
| `outputs` | `object` | A map of output from previous promotion steps indexed by step aliases. |
| `secrets` | `object` | A map of maps indexed by the names of all Kubernetes `Secret`s in the `Promotion`'s `Project` and the keys within the `Data` block of each. |
| `vars` | `object` | A user-defined map of variable names to static values of any type. The map is derived from a `Promotion`'s `spec.promotionTemplate.spec.vars` field. Variable names must observe standard Go variable-naming rules. Variables values may, themselves, be defined using an expression. `vars` (contains previously defined variables) and `ctx` are available to expressions defining the values of variables, however, `outputs` and `secrets` are not. |

:::info
Expect other useful variables to be added in the future.
:::

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
        messageFromSteps:
        - update-image
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

At present, such re-use can be achieved only through manual copy/paste, but
support for a new, top-level `PromotionTemplate` resource type is planned for an
upcoming release.
:::

## Functions

Several functions are built-in to Kargo's expression language. This section
describes each of them.

### `quote()`

The `quote()` function takes a single argument of any type and returns a string
representation. This is useful for scenarios where an expression evaluates to a
non-`string` JSON type, but you wish to treat it as a `string` regardless.

Example:

```yaml
config:
  numField: ${{ 40 + 2 }} # Will be treated as a number
  strField: ${{ quote(40 + 2) }} # Will be treated as a string
```

### `warehouse()`

The `warehouse()` function takes a single argument of type `string`, which is the
name of a `Warehouse` resource in the same `Project` as the `Promotion` being
executed. It returns a `FreightOrigin` object representing that `Warehouse`.

The `FreightOrigin` object can be used as an optional argument to the
`commitFrom()`, `imageFrom()`, or `chartFrom()` functions to disambiguate the
desired source of an artifact when necessary.

See the next sections for examples.

### `commitFrom()`

The `commitFrom()` function takes the URL of a Git repository as its first
argument and returns a corresponding `GitCommit` object from the `Promotion`'s
`FreightCollection`.

In the event that a `Stage` requests `Freight` from multiple origins
(`Warehouse`s) and more than one of those can provide a `GitCommit` object from
the specified repository, a `FreightOrigin` may be used as a second argument to
disambiguate the desired source.

Example:

```yaml
config:
  commitID: ${{ commitFrom("https://github.com/example/repo.git", warehouse("my-warehouse")).ID }}
```

### `imageFrom()`

The `imageFrom()` function takes the URL of a container image repository as its
first argument and returns a corresponding `Image` object from the `Promotion`'s
`FreightCollection`.

In the event that a `Stage` requests `Freight` from multiple origins
(`Warehouse`s) and more than one of those can provide an `Image` object from the
specified repository, a `FreightOrigin` may be used as a second argument to
disambiguate the desired source.

Example:

```yaml
config:
  imageTag: ${{ imageFrom("public.ecr.aws/nginx/nginx", warehouse("my-warehouse")).Tag }}
```

### `chartFrom()`

The `chartFrom()` function takes the URL of a Helm chart repository as its first
argument and returns a corresponding `Chart` object from the `Promotion`'s
`FreightCollection`.

For Helm charts stored in OCI registries, the URL should be the full path to the
repository within that registry.

For Helm charts stored in classic (http/s) repositories, which can store
multiple different charts within a single repository, a second argument should
be used to specify the name of the chart within the repository.

In the event that a `Stage` requests `Freight` from multiple origins
(`Warehouse`s) and more than one of those can provide a `Chart` object from the
specified repository, a `FreightOrigin` may be used as a final argument to
disambiguate the desired source.

OCI registry example:

```yaml
config:
  chartVersion: ${{ chartFrom("oci://example.com/my-chart", warehouse("my-warehouse")).Version }}
```

Classic repository example:

```yaml
config:
  chartVersion: ${{ chartFrom("https://example.com/charts", "my-chart", warehouse("my-warehouse")).Version }}
```
