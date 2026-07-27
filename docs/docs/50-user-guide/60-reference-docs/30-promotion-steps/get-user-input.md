---
sidebar_label: get-user-input
description: Pauses a promotion to collect structured input from a user through a form rendered in the Kargo UI.
---

# `get-user-input`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.12.0 and above.
:::

The `get-user-input` step pauses a promotion and waits for a user to submit
input through a form in the Kargo UI. You describe the fields to collect with a
JSON schema; the UI renders a form from it, validates the submission against it,
and exposes the submitted values as step outputs for later steps to use.

## Responders

The `responders` field lists the rules that decide who may submit input. The
rules are ORed: a caller matching any rule may respond. When `responders` is
omitted, any authenticated user who can see the Promotion may respond.

Each rule matches either an OIDC claim or a Kargo role:

| Field   | Type     | Description                                                                                                          |
| ------- | -------- | ------------------------------------------------------------------------------------------------------------------- |
| `claim` | `string` | The name of an OIDC claim to match (e.g. `groups`, `email`). Must be paired with `value`.                           |
| `value` | `string` | The claim value to match. For list-valued claims (e.g. `groups`), the rule matches if the list contains this value. |
| `role`  | `string` | The name of a Kargo role (a project ServiceAccount) the caller must be mapped to.                                   |

A rule sets either `claim` and `value`, or `role` — not both.

## Configuration

| Name           | Type       | Required | Description                                                                                                                                                                                                       |
| -------------- | ---------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `schema`       | `object`   | Y        | A complete JSON schema describing the values to collect. The UI renders a form from it and validates the submission against it. Declare optional fields by leaving them out of the schema's own `required` list. |
| `responders`   | `[]object` | N        | Rules identifying who may submit input. See [Responders](#responders). When empty, any authenticated user who can see the Promotion may respond.                                                                 |
| `display`      | `string`   | N        | An optional description shown alongside the input form in the UI.                                                                                                                                                |
| `pollInterval` | `string`   | N        | How often to re-check for input as a fallback (the step also wakes immediately when input arrives). A Go duration string. Defaults to `30s`.                                                                     |

## Output

| Name          | Type     | Description                                                       |
| ------------- | -------- | ---------------------------------------------------------------- |
| `values`      | `object` | The values the user submitted, matching the configured `schema`. |
| `respondedBy` | `string` | The identity of the user who submitted the input.                |
| `respondedAt` | `string` | When the input was submitted, as an RFC 3339 timestamp.          |

## Example

Collect a release version and summary before continuing, restricted to the
project's `release-manager` role, then reference the values in a later step:

```yaml
steps:
- uses: get-user-input
  as: collect
  config:
    display: Provide the details for this production release.
    responders:
    - role: release-manager
    schema:
      type: object
      additionalProperties: false
      required:
      - version
      - summary
      properties:
        version:
          type: string
          pattern: "^v?[0-9]+\\.[0-9]+\\.[0-9]+$"
          description: Release version, e.g. v1.24.1
        summary:
          type: string
          minLength: 1
          description: What is changing in this release

- uses: gh-create-issue
  config:
    repoURL: https://github.com/myorg/myrepo
    title: "Release ${{ outputs.collect.values.version }}"
    body: |
      **Summary:** ${{ outputs.collect.values.summary }}

      Filed by ${{ outputs.collect.respondedBy }} via Kargo.
```
