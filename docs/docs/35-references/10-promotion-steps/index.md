---
sidebar_label: Promotion Steps Reference
description: Learn about all of Kargo's built-in promotion steps
---

import DocCardList from '@theme/DocCardList';

# Promotion Steps Reference

Kargo's promotion steps are the building blocks of a promotion process. They
perform the actions necessary to promote a piece of Freight into a Stage.
Promotion steps are designed to be composable, allowing users to construct
complex promotion processes from simple, reusable components.

## Defining a Promotion Step

A promotion step is a YAML object with at least one key, `uses`, whose value is
the name of the step to be executed. The step's configuration is provided in a
subsequent key, `config`. The `config` key's value is an object whose keys are
the configuration options for the step.

```yaml
steps:
- uses: step-name
  config:
    option1: value1
    option2: value2
```

:::info
For a list of built-in promotion steps and configuration options, see the
[Built-in Steps](#built-in-steps) section.
:::

### Step Aliases

A step can be given an alias by providing an `as` key in the step definition.
The value of the `as` key is the alias to be used to reference the
[step's output](#step-outputs).

```yaml
steps:
- uses: step-name
  as: alias
```

### Step Variables

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
other steps. The values of variables may contain [expressions](../20-expression-language.md).
In addition, the values of step variables may be references to the
[outputs](#step-outputs) of other steps.

```yaml
steps:
- uses: step1
  as: step1
- uses: step2
  vars:
  - name: var1
    value: ${{ outputs.step1.someOutput }}
```

When a variable in a step is also defined as a global variable in the
[Promotion Template](../../30-how-to-guides/14-working-with-stages.md#promotion-templates),
the step variable takes precedence over the global variable.

### Step Outputs

A promotion step may produce output that can be referenced by subsequent steps,
allowing the output of one step to be used as input to another. The output of a
step is defined by the step itself and is typically documented in the step's
reference.

```yaml
steps:
- uses: step-name
  as: alias
- uses: another-step
  config:
    input: ${{ outputs.alias.someOutput }}
```

### Step Retries

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
    prNumber: ${{ outputs['open-pr'].prNumber }}
```

:::info
This feature was introduced in Kargo v1.1.0, and is still undergoing refinements
and improvements to better distinguish between transient and non-transient
errors, and to provide more control over retry behavior like backoff strategies
or time limits.
:::

### Promotion Task Step

A step can be used to reference a
[`PromotionTask` or `ClusterPromotionTask`](../30-promotion-tasks.md)
using the `task` key, whose value is an object with a `name` key that specifies
the name of the task and optionally a `kind` key to specify if the task is a
`ClusterPromotionTask`.

```yaml
steps:
- task:
    name: task-name
    kind: ClusterPromotionTask
```

When a task is referenced, the `uses` key is not required.

## Built-in Steps

Below is a list of all the promotion steps built directly into Kargo. Each page
provides detailed information about the step, including its purpose, configuration
options, and examples.

:::info
Promotion steps support the use of [expr-lang] expressions in their
configuration. Many examples in this reference document will include expressions
to demonstrate their use. For more information on expressions, refer to our
[Expression Language Reference](../20-expression-language.md).
:::

<DocCardList />

[expr-lang]: https://expr-lang.org/
