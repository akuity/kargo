# http_promo_step

Runs an `http` promotion step that POSTs a Slack-style message to a configured endpoint.

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `http_endpoint` | URL of an HTTP endpoint that accepts a POST and returns a 2xx response, reachable from **inside the cluster** (e.g. an in-cluster echo `Service`). The promotion's `http` step posts a Slack-style message to it. |

Example:

```yaml
context:
  http_endpoint: http://echo.default.svc.cluster.local:80
```
