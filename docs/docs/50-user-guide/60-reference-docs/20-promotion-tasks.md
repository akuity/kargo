---
sidebar_label: Promotion Tasks
toc_max_heading_level: 4
---

# Promotion Tasks Reference

Promotion Tasks define reusable sets of promotion steps that can be shared across
multiple [Promotion Templates](15-promotion-templates.md). They come in two forms:

- `PromotionTask`: Scoped to a specific project
- `ClusterPromotionTask`: Available globally across all projects

When a Promotion Template references a promotion task, Kargo inflates the task's
steps and merges them with the template's steps when creating a `Promotion`.
This makes it easy to standardize common promotion workflows across your project
or organization.

## Defining a Promotion Task

A promotion task is defined using the `PromotionTask` or `ClusterPromotionTask`
resource type. The task contains two main components:

- [Variables](#task-variables) that provide configurable inputs when the task
  is used
- [Steps](#task-steps) that define the sequence of built-in steps to perform

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionTask
metadata:
  name: open-pr-and-wait
  namespace: kargo-demo
spec:
  # Task-wide input variables
  vars:
  - name: repoURL
  - name: sourceBranch
  - name: targetBranch
    value: main

  # Sequence of promotion steps
  steps:
  - uses: git-open-pr
    as: open-pr
    config:
      repoURL: ${{ vars.repoURL }}
      createTargetBranch: true
      sourceBranch: ${{ vars.sourceBranch }}
      targetBranch: ${{ vars.targetBranch }}
```

### Task Variables

Variables in a Promotion Task define the inputs required when the task is used within
a Promotion Template. They provide a way to parameterize the task's behavior while
maintaining reusability.

To define variables, use the `vars` field in the task's `spec`. Each variable requires
a `name` and optionally a default `value`.

```yaml
vars:
# Required variable (no default)
- name: repoURL
# Optional variable with default value
- name: targetBranch
  value: main
```

:::info
Variables without a default value are required and must be provided when the task
is referenced in a Promotion Template.
:::

Variables can be referenced throughout the task using `${{ vars.<variable-name> }}`:

```yaml
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.repoURL }}
    checkout:
    - branch: ${{ vars.targetBranch }}
      path: ./target
```

When using a promotion task in a template, variables can be provided in two ways:

1. Through the template's
   [global `vars` section](15-promotion-templates.md#variables)
   (inherited by the task)
1. Through the
   [task step's `vars` section](15-promotion-templates.md#step-variables)
   (overrides inherited values)

```yaml
spec:
  promotionTemplate:
    spec:
      # Global vars inherited by tasks
      vars:
      - name: repoURL
        value: https://github.com/example/repo.git

      steps:
      - task:
          name: my-task
        # Step-level vars override inherited values
        vars:
        - name: targetBranch
          value: feature-branch
```

### Task Steps

The `steps` section in a Promotion Task defines the sequence of actions to
perform when the task is used. Each step can reference a
[built-in promotion step](./promotion-steps) using the `uses` key:

```yaml
steps:
- uses: git-clone
  as: clone
  config:
    repoURL: ${{ vars.repoURL }}
    checkout:
    - branch: ${{ vars.targetBranch }}
      path: ./target
```

:::note
Unlike Promotion Templates, task steps cannot reference other Promotion Tasks.
This prevents circular dependencies and keeps tasks focused on a specific
workflow.
:::

#### Task Context

Steps within a promotion task have access to an additional
[pre-defined variable](40-expressions.md#pre-defined-variables) called
`task` that provides access to outputs from previous steps in the task. The
`task.outputs` property is a map of step aliases within the `PromotionTask`
to their outputs.

```yaml
steps:
- uses: git-open-pr
  as: open-pr
  config:
    repoURL: ${{ vars.repoURL }}
- uses: git-wait-for-pr
  config:
    prNumber: ${{ task.outputs['open-pr'].prNumber }}
```

:::info
The `task.outputs` variable is required when referencing outputs from previous
steps within the same task.

This requirement exists because tasks are inflated during the creation of a
`Promotion`, and after inflation, the alias of the task step is namespaced
to avoid conflicts with other steps in the template. This means that the
alias of a task step at runtime is not known to the `PromotionTask` definition,
so it cannot be used to reference outputs.
:::

### Task Outputs

A promotion task can expose outputs that become available to subsequent steps in
the parent Promotion Template. To define outputs for a task, use the
[`compose-output` step](30-promotion-steps/compose-output.md).

```yaml
steps:
# ...omitted for brevity
- uses: compose-output
  as: output
  config:
    commit: ${{ task.outputs['wait-for-pr'].commit }}
    branch: ${{ vars.targetBranch }}
```

The composed outputs become available in the template under the task step's alias.

```yaml
steps:
- task:
    name: my-task
  as: promotion
- uses: http
  config:
    url: https://api.example.com/notify
    body: |
      New commit: ${{ outputs.promotion.commit }}
```

## Defining a Global Promotion Task

To create a promotion task that's available across all projects, use the
`ClusterPromotionTask` resource. It's defined exactly like a `PromotionTask`
but without namespace scope:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterPromotionTask
metadata:
  name: global-task
spec:
  # ...equivalent to a PromotionTask
```

To use a `ClusterPromotionTask` in a template, specify the `kind` in the task
reference:

```yaml
steps:
- task:
    name: global-task
    kind: ClusterPromotionTask
```

:::info
`ClusterPromotionTasks` are perfect for standardizing promotion workflows across
your organization, such as promotion patterns that should be consistently applied
across all projects.
:::
