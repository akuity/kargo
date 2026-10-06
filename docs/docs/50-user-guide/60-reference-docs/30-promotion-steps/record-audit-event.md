---
sidebar_label: record-audit-event
description: Records a custom, auditable event during a promotion, surfaced in the Akuity Platform audit log.
---

# `record-audit-event`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.12.0 and above.
:::

The `record-audit-event` step records a custom, auditable event as part of a
promotion. Use it to capture actions that would otherwise live only inside
referenced Kubernetes objects — for example, who approved a release — so they
appear in the Akuity Platform's central audit log. The event is also visible in
the Kargo project's Events tab.

Each run records exactly one event. The step is commonly paired with an
interactive step such as [`wait-for-approval`](./wait-for-approval.md) to record
the outcome of a human decision.

:::warning
Do not reference secrets in this step's config. Kargo renders the config
(evaluating expressions, including `secret()`) before the step runs, and the
`message` and `data` are recorded verbatim in the audit log and surfaced in the
Kargo UI. Any secret pulled into the config this way may be exposed to anyone
who can view the audit log or the Promotion's events.
:::

## Configuration

| Name      | Type     | Required | Description                                                                                                                                    |
| --------- | -------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `message` | `string` | Y        | The human-readable message describing the event. Recorded verbatim in the audit log.                                                         |
| `type`    | `string` | N        | A short label categorizing the event (e.g. `approved`, `deployed`), recorded as the audit action. Defaults to `recorded`.                     |
| `actor`   | `string` | N        | The identity to attribute the event to. Defaults to the actor that triggered the Promotion. Set this to attribute the event to someone else. |
| `data`    | `object` | N        | Arbitrary string key/value metadata to attach to the event. Recorded in the audit log's details.                                             |

This step produces no output.

## Reliability

The step records the event by creating a Kubernetes Event through the Kargo
control-plane API. A transient API error fails the step, so configure
[`retry`](../15-promotion-templates.md#step-retries) on the step (via
`retry.errorThreshold`) to ride out a temporary control-plane hiccup rather than
fail the promotion and drop the event.

Recording is idempotent: the event's name is derived from the Promotion and the
step's alias, so a retry — or a re-run after a controller restart — reuses the
same event and never records a duplicate.

```yaml
steps:
- uses: record-audit-event
  as: audit
  retry:
    errorThreshold: 3
  config:
    message: Production release recorded.
```

## Examples

### Recording a Simple Event

```yaml
steps:
- uses: record-audit-event
  as: audit
  config:
    type: deployed
    message: "Promoted ${{ ctx.stage }} to production."
    data:
      environment: production
```

### Recording an Approval

Pair the step with [`wait-for-approval`](./wait-for-approval.md) to record who
approved a release in the central audit log, attributing the event to the
approver:

```yaml
steps:
- uses: wait-for-approval
  as: approve
  config:
    minApprovals: 1

- uses: record-audit-event
  as: audit
  retry:
    errorThreshold: 3
  config:
    type: approved
    message: "Release to ${{ ctx.stage }} approved by ${{ outputs.approve.responses[0].user }}."
    actor: ${{ outputs.approve.responses[0].user }}
```
