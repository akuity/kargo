---
sidebar_label: argocd-wait
description: Waits for one or more Argo CD Application resources to reach desired conditions.
---

# `argocd-wait`

`argocd-wait` waits for one or more Argo CD `Application` resources to reach
desired conditions. `Application`s can be selected either by exact name or by
using label selectors to match multiple `Application`s at once.

This step is useful when a sync is triggered by means other than an
[`argocd-update` step](argocd-update.md) — for example, when an external system
or a human operator triggers the sync — and you want a `Promotion` to wait for
the `Application` to become healthy and synced before proceeding.

:::note

Unlike [`argocd-update`](argocd-update.md), `argocd-wait` does **not** require
the `kargo.akuity.io/authorized-stage` annotation on the `Application`. It can
wait for any `Application` that the Kargo controller has read access to.

:::

## Application Selection

The `argocd-wait` step supports two methods for selecting Argo CD `Application`
resources:

1. **By Name**: Specify an exact application name using the `name` field.
2. **By Label Selector**: Match one or more `Application`s using the `selector`
   field with label-based criteria.

These two methods are mutually exclusive — you must specify either `name` or
`selector`, but not both.

See the [`argocd-update` documentation](argocd-update.md#application-selection)
for a full description of both selection methods including label selector syntax.

## Wait Conditions

The `waitFor` field controls which conditions must be satisfied before the step
succeeds. Supported values are:

| Value | Description |
|-------|-------------|
| `health` | `Application` health is `Healthy`. |
| `sync` | `Application` sync status is `Synced`. |
| `operation` | No operation (e.g. sync) is currently in progress. |
| `suspended` | `Application` health is `Suspended`. |
| `degraded` | `Application` health is `Degraded`. |

When `waitFor` is omitted, it defaults to `[health, sync, operation]`.

The health-related values (`health`, `suspended`, `degraded`) are OR'd: the
health check passes if _any_ of the specified health conditions is met. All
other conditions (`sync`, `operation`) are AND'd with the result.

:::note

If `waitFor` includes `health` and an `Application` transitions to `Degraded`
from a state that was neither `Degraded` nor `Unknown`, the step fails
immediately rather than continuing to wait. This degradation detection prevents
a `Promotion` from hanging indefinitely on an `Application` that regressed
during the wait.

:::

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apps` | `[]object` | Y | Describes Argo CD `Application` resources to wait for. At least one must be specified. |
| `apps[].name` | `string` | N | The name of the Argo CD `Application`. Mutually exclusive with `selector`. Either `name` or `selector` must be specified. __Note:__ Expressions in this field are limited to accessing `ctx` and `vars` and may not access `secrets` or any Freight. This is because templates in this field are, at times, evaluated outside the context of an actual `Promotion` for the purposes of building an index. In practice, this restriction does not prove to be especially limiting. |
| `apps[].namespace` | `string` | N | The namespace of the Argo CD `Application` resource(s) to wait for. If left unspecified, the namespace will be the Kargo controller's configured default — typically `argocd`. __Note:__ This field is subject to the same restrictions as the `name` field. See above. |
| `apps[].selector` | `object` | N | Label selector for matching one or more Argo CD `Application` resources. Mutually exclusive with `name`. Either `name` or `selector` must be specified. |
| `apps[].selector.matchLabels` | `map[string]string` | N | A map of label key-value pairs. All specified labels must match for an `Application` to be selected (AND logic). At least one of `matchLabels` or `matchExpressions` must be specified. |
| `apps[].selector.matchExpressions` | `[]object` | N | A list of label selector requirements. All expressions must be satisfied for an `Application` to be selected. At least one of `matchLabels` or `matchExpressions` must be specified. |
| `apps[].selector.matchExpressions[].key` | `string` | Y | The label key that the selector applies to. |
| `apps[].selector.matchExpressions[].operator` | `string` | Y | The operator to use for matching. Valid values: `In`, `NotIn`, `Exists`, `DoesNotExist`. |
| `apps[].selector.matchExpressions[].values` | `[]string` | N | An array of string values. Required when `operator` is `In` or `NotIn`. Must be empty when `operator` is `Exists` or `DoesNotExist`. |
| `apps[].waitFor` | `[]string` | N | Conditions to wait for. Valid values: `health`, `sync`, `operation`, `suspended`, `degraded`. Defaults to `[health, sync, operation]` when omitted. |

## Examples

### Common Usage

In this example, `argocd-wait` is used after [`argocd-update`](argocd-update.md)
to confirm that the `Application` has reached a healthy and synced state before
the `Promotion` is marked as succeeded.

```yaml
steps:
# Clone, render manifests, commit, push, etc...
- uses: git-commit
  as: commit
  config:
    path: ./out
    message: ${{ outputs['update-image'].commitMessage }}
- uses: git-push
  config:
    path: ./out
- uses: argocd-update
  config:
    apps:
    - name: my-app
      sources:
      - repoURL: https://github.com/example/repo.git
        desiredRevision: ${{ outputs.commit.commit }}
- uses: argocd-wait
  config:
    apps:
    - name: my-app
```

### Waiting Only for Operation Completion

This example waits only for any in-progress sync operation to finish, without
requiring the `Application` to be healthy or synced to a specific revision.

```yaml
steps:
- uses: argocd-wait
  config:
    apps:
    - name: my-app
      waitFor:
      - operation
```

### Waiting with Label Selector

This example waits for all `Application`s with a given label to become healthy,
synced, and idle — useful when multiple `Application`s are updated together.

```yaml
steps:
- uses: argocd-wait
  config:
    apps:
    - selector:
        matchLabels:
          environment: staging
```

### Waiting for a Suspended Application

This example waits for an `Application` to reach the `Suspended` health state,
which can serve as a manual gate: a human suspends the `Application` to pause
a rollout, and this step waits until that state is observed.

```yaml
steps:
- uses: argocd-wait
  config:
    apps:
    - name: my-app
      waitFor:
      - suspended
```
