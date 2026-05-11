---
sidebar_label: fail
description: Fails the promotion.
---

# `fail`

`fail` fails the promotion.

## Configuration

| Name      | Type     | Required | Description                   |
| --------- | -------- | -------- | ----------------------------- |
| `message` | `string` | N        | Optional message to fail with |

## Examples

### Common Usage

It may be necessary to fail a promotion if certain conditions are met. This can
be done by combining this step with
[conditional step execution](../15-promotion-templates.md#conditional-steps).

In this example, a HTTP request fetches an expected chart name from an external
API. If it does not match the chart name from the Freight, the promotion fails
with a message indicating the mismatch.

```yaml
steps:
  - uses: http
    as: expected-name
    config:
      url: https://api.example.com/expected-name
      outputs:
        - name: name
          fromExpression: response.body.name
  - uses: fail
    if: ${{ task.outputs['expected-name'].name != chartFrom('oci://example.com/my-chart').Name }}
    config:
      message: Expected chart name ${{ task.outputs['expected-name'].name }}
```
