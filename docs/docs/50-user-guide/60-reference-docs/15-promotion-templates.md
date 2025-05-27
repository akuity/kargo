---
sidebar_label: Promotion Templates
toc_max_heading_level: 4
---

# Promotion Templates Reference

Promotion Templates define how Kargo transitions `Freight` into a `Stage` by
executing a series of discrete, composable steps. Each step in a promotion
template performs a specific action, from simple operations like cloning a
Git repository to complex workflows like managing the lifecycle of a pull
request.

When `Freight` is promoted to a `Stage`, Kargo uses the promotion template
associated with that `Stage` to create a `Promotion` object which orchestrates
the promotion process.

## Defining a Promotion Template

A promotion template is defined within a `Stage`'s configuration using the
`spec.promotionTemplate` field. The template contains two main components:

- [Global variables](#variables) that provide configurable values
  for the promotion process
- [Steps](#steps) that define the sequence of actions to perform

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
  promotionTemplate:
    spec:
      # Template-wide variables
      vars:
        - name: gitRepo
          value: https://github.com/example/repo.git

      # Sequence of promotion steps
      steps:
        - uses: git-clone
          config:
            repoURL: ${{ vars.gitRepo }}
            checkout:
            - branch: main
              path: ./src
        - task:
            name: update-manifests
            kind: PromotionTask
```

### Variables

Variables provide a way to define reusable values that can be referenced
throughout the promotion template. They can be used to parameterize steps,
making the template more flexible and easier to maintain.

You can define variables at two levels: globally for the entire template, or
[scoped to a specific step](#step-variables). Global variables are perfect for
values you want to use throughout the promotion process, like repository URLs
or branch names.

To define global variables, use the `vars` field in the `spec` of the promotion
template. The value of the `vars` key is a list of variables, each of which is
an object with `name` and `value` keys.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
- name: targetBranch
  value: stage/${{ ctx.stage }}
```

Once defined, you can reference these variables throughout the template using
`${{ vars.<variable-name> }}`.

```yaml
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - branch: ${{ vars.targetBranch }}
      path: ./target
```

:::info
Global variables can use [expressions](40-expressions.md) within `${{ }}` to
compute values dynamically, including references to context variables like
`ctx.stage`.

They **do not** support referencing outputs from steps. To reference outputs,
use [step-specific variables](#step-variables).
:::

:::info
Variables defined at the template level are available to all steps within the
template. You can override these values within individual steps by defining
[step-specific variables](#step-variables).
:::

### Steps

The `steps` section in the `spec` of a promotion template defines the sequence
of actions to perform during the promotion process. Steps come in two forms:

- Steps that reference a [built-in promotion step](#built-in-steps)
- Steps that reference a
  [`PromotionTask` or `ClusterPromotionTask`](#promotion-task-steps)

#### Built-in Steps

A step can be used to reference a built-in promotion step using the `uses` key
whose value is the name of the step.

```yaml
steps:
- uses: step-name
```

:::info
For a list of built-in promotion steps and configuration options, see the
[Promotion Steps Reference](./promotion-steps).
:::

#### Promotion Task Steps

A step can be used to reference a
[`PromotionTask` or `ClusterPromotionTask`](20-promotion-tasks.md)
using the `task` key, whose value is an object with a `name` key that specifies
the name of the task and optionally a `kind` key that specifies the kind of task
to reference. The `kind` key is optional and defaults to `PromotionTask`.

```yaml
steps:
- task:
    name: task-name
    kind: ClusterPromotionTask
```

:::note
Steps referencing `PromotionTask` or `ClusterPromotionTask` do not support
configuration or retry options like built-in step, as the steps within the
task define their own configuration. For more information, see the
[Promotion Tasks Reference](20-promotion-tasks.md).
:::

#### Step Variables

A step can define variables that can be referenced in its configuration by
providing a `vars` key in the step definition. The value of the `vars` key is a
list of variables, each of which is an object with `name` and `value` keys.

```yaml
steps:
- uses: step-name
  vars:
  - name: var1
    value: value1
  - name: var2
    value: value2
  config:
    option1: ${{ vars.var1 }}
    option2: ${{ vars.var2 }}
```

Variables defined in a step are scoped to that step and are not accessible to
other steps like [global variables](#variables) are. The values of
variables may  contain [expressions](40-expressions.md). In addition, the
values of step variables  may contain references to the
[outputs](#step-outputs) of other steps.

```yaml
steps:
- uses: step-name
  as: step1
- uses: another-step
  vars:
  - name: var1
    value: ${{ outputs.step1.someOutput }}
```

:::info
Step variables with the same name as global variables will override the global
value for that step.
:::

#### Step Configuration

Each step in a promotion template can be configured with a set of options that
control its behavior. The `config` key in a step definition is an object that
contains the configuration options for the step.

```yaml
steps:
- uses: step-name
  config:
    option1: value1
    option2: value2
```

The configuration options available for a step are specific to the step itself
and are documented in the [Promotion Steps Reference](./promotion-steps).

#### Step Outputs

A promotion step may produce output that can be referenced by subsequent steps,
allowing the output of one step to be used as input to another. The output of a
step is defined by the step itself and is typically documented in the step's
[reference documentation](./promotion-steps).

```yaml
steps:
  - uses: step-name
    as: alias
  - uses: another-step
    config:
      input: ${{ outputs.alias.someOutput }}
```

#### Conditional Steps

You can conditionally execute a step based on the results of previous steps by
using the `if` key in the step definition. The value of the `if` key must be a
valid [expression](40-expressions.md) that evaluates to a boolean value. If the
expression evaluates to `true`, the step is executed as normal. If the
expression evaluates to `false`, the step is skipped and the next step in the
sequence is executed.

These conditionals may make use of special functions whose returned values
depend on the outcome of all previous steps:

- `always()`: Unconditionally evaluates to `true`.
- `failure()`: Evaluates to `true` only if _any_ previous step has errored or
  failed.
- `success()`: Evaluates to `true` only if _all_ previous steps have completed
  successfully or been skipped.

Examples:

```yaml
steps:
- uses: some-step # Assume this step succeeds
- uses: some-other-step
  if: ${{ always() }} # This step will execute; assume it succeeds
- uses: and-another-step
  if: ${{ failure() }} # This step will be skipped
- uses: yet-another-step
  if: ${{ success() }} # This step will execute; assume it succeeds

# This Promotion will succeed
```

```yaml
steps:
- uses: some-step # Assume this step fails
- uses: some-other-step
  if: ${{ always() }} # This step will execute
- uses: and-another-step
  if: ${{ failure() }} # This step will execute  
- uses: yet-another-step
  if: ${{ success() }} # This step will be skipped

# This Promotion will fail
```

It is also possible to directly access the status of a specific step using the
`status()` function with a step alias as an argument:

```yaml
steps:
- uses: some-step # Assume this step encounters an error
  as: my-step
- uses: some-other-step
  if: ${{ always() }} # This step will execute
- uses: and-another-step
  if: ${{ failure() }} # This step will execute
- uses: yet-another-step
  if: ${{ status('my-step') == 'Errored' }} # This step will execute

# This Promotion will fail
```

A step's `continueOnError` field can be set to `true` to indicate that an error
or failure should _both_:

- Not be counted as an error or failure when evaluating the `failure()` or
  `success()` functions.
- Not affect the overall outcome of the `Promotion`.

Example:

```yaml
steps:
- uses: some-step # Assume this step fails
  continueOnError: true
- uses: some-other-step
  if: ${{ always() }} # This step will execute; assume it succeeds
- uses: and-another-step
  if: ${{ failure() }} # This step will be skipped
- uses: yet-another-step
  if: ${{ success() }} # This step will execute; assume it succeeds

# This Promotion will succeed
```

Last, when a step's `if` field is empty, the default behavior is to execute the
step only if _all_ previous steps have either completed successfully, been
skipped, or completed with some other status, but had `continueOnError` set to
`true`.

Examples:

```yaml
steps:
- uses: some-step # Assume this step succeeds
- uses: some-other-step # This step will execute; assume it fails
  continueOnError: true
- uses: and-another-step # This step will execute; assume it succeeds
- uses: yet-another-step # This step will execute; assume it succeeds

# This Promotion will succeed
```

:::tip
Using the primitives described above, it is possible to create robust error-handling
logic in your promotion templates.
:::

:::tip
Conditional steps can be used in [Promotion Tasks](20-promotion-tasks.md) to
conditionally execute a task step based on provided
[task variables](20-promotion-tasks.md#task-variables).
:::

#### Step Retries

When a step fails for any reason, it can be retried instead of immediately
failing the entire `Promotion`. An _error threshold_ specifies the number of
_consecutive_ failures required for retry attempts to be abandoned and the
`Promotion` to fail.

Independent of the error threshold, steps are also subject to a _timeout_. Any
step that doesn't achieve its goal within that interval will cause the
`Promotion` to fail. For steps that exhibit any kind of polling behavior, the
timeout can cause a `Promotion` to fail with no _other_ failure having occurred.

System-wide, the default error threshold is 1 and the default timeout is
indefinite. Thus, default behavior is effectively no retries when a step fails
for any reason and steps with any kind of polling behavior will poll
indefinitely _as long a no other failure occurs._

The implementations of individual steps can override these defaults. Users also
may override these defaults through configuration. In the following example, the
`git-wait-for-pr` step is configured not to fail the `Promotion` until three
consecutive failed attempts to execute it. It is also configured to wait a
maximum of 48 hours for the step to complete successfully (i.e. for the PR to be
merged).

```yaml
steps:
# ...
- uses: wait-for-pr
  retry:
    errorThreshold: 3
    timeout: 48h
  config:
    prNumber: ${{ outputs['open-pr'].pr.id }}
```

:::info
This feature was introduced in Kargo v1.1.0, and is still undergoing refinements
and improvements to better distinguish between transient and non-transient
errors, and to provide more control over retry behavior like backoff strategies
or time limits.
:::
