---
sidebar_label: wait-for-approval
description: Pauses a promotion until the required number of users approve, or one rejects, through the Kargo UI.
---

# `wait-for-approval`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.12.0 and above.
:::

The `wait-for-approval` step pauses a promotion and waits for human approval
before it continues. The promotion stays in a running state, surfacing an
approval prompt in the Kargo UI, until enough distinct users approve or any
eligible user rejects. A rejection fails the step (and the promotion). Both the
approvals and the rejection are recorded, so the full decision trail is
available to later steps.

## Approvers

The `approvers` field lists the rules that decide who may approve or reject.
The rules are ORed: a caller matching any rule may respond. When `approvers` is
omitted, any authenticated user who can see the Promotion may respond.

Each rule matches either an OIDC claim or a Kargo role:

| Field      | Type      | Description                                                                                                                 |
| ---------- | --------- | --------------------------------------------------------------------------------------------------------------------------- |
| `claim`    | `string`  | The name of an OIDC claim to match (e.g. `groups`, `email`). Must be paired with `value`.                                   |
| `value`    | `string`  | The claim value to match. For list-valued claims (e.g. `groups`), the rule matches if the list contains this value.         |
| `role`     | `string`  | The name of a Kargo role (a project ServiceAccount) the caller must be mapped to.                                           |
| `required` | `boolean` | When `true`, the step cannot succeed until at least one approval comes from a user matching this rule. Defaults to `false`. |

A rule sets either `claim` and `value`, or `role` — not both.

## Configuration

| Name           | Type       | Required | Description                                                                                                                                            |
| -------------- | ---------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `approvers`    | `[]object` | N        | Rules identifying who may approve or reject. See [Approvers](#approvers). When empty, any authenticated user who can see the Promotion may respond.   |
| `minApprovals` | `integer`  | N        | The number of distinct users that must approve before the step succeeds. Defaults to `1`.                                                            |
| `pollInterval` | `string`   | N        | How often to re-check for responses as a fallback (the step also wakes immediately when a response arrives). A Go duration string. Defaults to `30s`. |

## Output

| Name        | Type       | Description                                                                                            |
| ----------- | ---------- | ----------------------------------------------------------------------------------------------------- |
| `responses` | `[]object` | Every response received, in chronological order. Includes the approvals and, on rejection, the rejection. |

Each entry in `responses` has the following fields:

| Field     | Type     | Description                                                                        |
| --------- | -------- | --------------------------------------------------------------------------------- |
| `user`    | `string` | The identity of the responder.                                                    |
| `action`  | `string` | The action taken: `approve` or `reject`.                                          |
| `time`    | `string` | When the response was recorded, as an RFC 3339 timestamp.                         |
| `message` | `string` | An optional free-text message the responder left. Omitted when none was provided. |

## Example

### Single Approval

Pause until any authenticated user who can see the Promotion approves:

```yaml
steps:
- uses: wait-for-approval
  as: approve
```

### Quorum With a Required Approver

Require two distinct approvals, at least one of which must come from a member of
the `platform-leads` group, and let the project's `release-manager` role
approve as well:

```yaml
steps:
- uses: wait-for-approval
  as: approve
  config:
    minApprovals: 2
    approvers:
    - claim: groups
      value: platform-leads
      required: true
    - role: release-manager
```

### Recording the Decision

The `responses` output carries the full trail, so a later step can record who
approved (or why a promotion was rejected):

```yaml
steps:
- uses: gh-create-issue
  as: ticket
  config:
    repoURL: https://github.com/myorg/myrepo
    title: "Promote to ${{ ctx.stage }}"

- uses: wait-for-approval
  as: approve
  config:
    minApprovals: 1

- uses: gh-issue-add-comment
  config:
    repoURL: https://github.com/myorg/myrepo
    issueNumber: ${{ outputs.ticket.number }}
    body: |
      Approved by:
      ${{ join(map(outputs.approve.responses, "- " + .user + (.message == nil ? "" : ": " + .message)), "\n") }}
```
