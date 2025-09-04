---
sidebar_label: set-metadata
description: Updates metadata on Stage or Freight resources during the promotion process.
---

# `set-metadata`

`set-metadata` updates metadata on `Stage` or `Freight` resources during the
promotion process. This step allows you to attach arbitrary key-value pairs to the
status of these resources, which can be useful for tracking deployment information,
version details, or any other relevant metadata.

## Configuration

| Name               | Type       | Required | Description                                                                                                                                                                                                 |
| ------------------ | ---------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `updates`          | `[]object` | Y        | List of metadata updates to apply to various resources.                                                                                                                                                      |
| `updates[].kind`   | `string`   | Y        | Kind of resource to update metadata for. Must be either "`Stage`" or "`Freight`".                                                                                                                            |
| `updates[].name`   | `string`   | Y        | Name of the resource to update metadata for.                                                                                                                                                                 |
| `updates[].values` | `object`   | Y        | Key-value pairs to set as metadata on the resource. Must contain at least one key-value pair. This field supports various types of values including strings, numbers, booleans, arrays, and nested objects. |

## Examples

### Common Usage

In this example, metadata is added to both a `Stage` and a `Freight` resource. This
pattern is commonly used to track deployment information or add context about the
promotion process.

```yaml
steps:
  - uses: set-metadata
    config:
      updates:
        - kind: Stage
          name: production
          values:
            deployedBy: "user@example.com"
            version: "1.0.0"
        - kind: Freight
          name: my-app-freight
          values:
            deployed: true
            deploymentStatus: "success"
```

### Complex Metadata

This example demonstrates setting more complex metadata structures, including nested
objects and arrays:

```yaml
steps:
  - uses: set-metadata
    config:
      updates:
        - kind: Stage
          name: staging
          values:
            deployment:
              timestamp: "xyz"
              components:
                - name: "frontend"
                  version: "2.1.0"
                  status: "healthy"
                - name: "backend"
                  version: "1.5.0"
                  status: "healthy"
            environment:
              region: "us-west-2"
              cluster: "staging-01"
```

### Using Metadata in Other Contexts

Once metadata is set (as shown in the examples above), it can be retrieved using
[`freightMetadata()`](../40-expressions.md#freightmetadatafreightname) and
[`stageMetadata()`](../40-expressions.md#stagemetadatastagename) expressions in other
steps or verification processes. To access the metadata in different contexts:

```yaml
# stage.yaml - Using metadata in Stage verification
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: production
spec:
  verification:
    analysisTemplate: verify-deployment
    args:
      # Access Stage metadata
      - name: frontend_version
        value: ${{ stageMetadata(ctx.stage)['deployment'].components[0].version }}
```

```yaml
# promotion-template.yaml - Using metadata in subsequent steps
steps:
  - uses: http
    if: ${{ deployment := stageMetadata()['deployment']; environment := stageMetadata()['environment']; true }}
    config:
      url: "https://slack.com/api/chat.postMessage"
      method: POST
      headers:
        Authorization: "Bearer ${{ secrets.SLACK_TOKEN }}"
      body:
        channel: "#deployments"
        # Then use the metadata maps in your message
        text: ${{ |
          sprintf(
            "Deployment to %s:\n- Region: %s\n- Number of Components: %d\n- Frontend: %s (%s)\n- Backend: %s (%s)",
            ctx.stage,
            environment.region,
            len(deployment.components),
            deployment.components[0].name,
            deployment.components[0].version,
            deployment.components[1].name,
            deployment.components[1].version
          )
        }}
```
