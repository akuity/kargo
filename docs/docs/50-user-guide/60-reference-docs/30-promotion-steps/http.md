---
sidebar_label: http
description: Makes an HTTP/S request to enable basic integration with a wide variety of external services.
---

# `http`

`http` is a generic step that makes an HTTP/S request to enable basic integration
with a wide variety of external services.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `method` | `string` | N | The HTTP method to use. Defaults to `GET` |
| `url` | `string` | Y | The URL to which the request should be made. |
| `headers` | `[]object` | N | A list of headers to include in the request. |
| `headers[].name` | `string` | Y | The name of the header. |
| `headers[].value` | `string` | Y | The value of the header. |
| `queryParams` | `[]object` | N | A list of query parameters to include in the request. |
| `queryParams[].name` | `string` | Y | The name of the query parameter. |
| `queryParams[].value` | `string` | Y | The value of the query parameter. The provided value will automatically be URL-encoded if necessary. |
| `body` | `string` | N | The body of the request. __Note:__ As this field is a `string`, take care to utilize [`quote()`](../40-expressions.md#quotevalue) if the body is a valid JSON `object`. Refer to the example below of posting a message to a Slack channel. |
| `insecureSkipTLSVerify` | `boolean` | N | Indicates whether to bypass TLS certificate verification when making the request. Setting this to `true` is highly discouraged. |
| `timeout` | `string` | N | A string representation of the maximum time interval to wait for a request to complete. _This is the timeout for an individual HTTP request. If a request is retried, each attempt is independently subject to this timeout._ See Go's [`time` package docs](https://pkg.go.dev/time#ParseDuration) for a description of the accepted format. |
| `successExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine success. If this is left undefined and `failureExpression` _is_ defined, the default success criteria will be the inverse of the specified failure criteria. If both are left undefined, success is `true` when the HTTP status code is `2xx`. If `successExpression` and `failureExpression` are both defined and both evaluate to `true`, the failure takes precedence. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `failureExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine failure. If this is left undefined and `successExpression` _is_ defined, the default failure criteria will be the inverse of the specified success criteria. If both are left undefined, failure is `true` when the HTTP status code is _not_ `2xx`. If `successExpression` and `failureExpression` are both defined and both evaluate to `true`, the failure takes precedence. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `outputs` | `[]object` | N | A list of rules for extracting outputs from the HTTP response. These are only applied to responses deemed successful. |
| `outputs[].name` | `string` | Y | The name of the output. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract a value from the HTTP response. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |

:::note
An HTTP response that is not conclusively determined to have succeeded or failed
will result in the step reporting a result of `Running`. Kargo will
[retry](../15-promotion-templates.md#step-retries) such a step on its next
attempt at reconciling the`Promotion` resource. This will continue until the step
succeeds, fails, exhausts the configured maximum number of retries, or a configured
timeout has elapsed.
:::

## Expressions

The `successExpression`, `failureExpression`, and `outputs[].fromExpression`
fields all support [expr-lang][] expressions.

:::note
The expressions included in the `successExpression`, `failureExpression`, and
`outputs[].fromExpression` fields should _not_ be offset by `${{` and `}}`. This
is to prevent the expressions from being evaluated by Kargo during
pre-processing of step configurations. The `http` step itself will evaluate
these expressions.
:::

A `response` object (a `map[string]any`) is available to these expressions. It
is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `status` | `int` | The HTTP status code of the response. |
| `headers` | `http.Header` | The headers of the response. See applicable [Go documentation](https://pkg.go.dev/net/http#Header). |
| `header` | `func(string) string` | `headers` can be inconvenient to work with directly. This function allows you to access a header by name. |
| `body` | `map[string]any` | The body of the response, if any, unmarshaled into a map. If the response body is empty, this map will also be empty. |

## Outputs

The `http` step only produces the outputs described by the `outputs` field of
its configuration.

## Examples

### Basic Usage

This example configuration makes a `GET` request to the
[Cat Facts API](https://www.catfacts.net/api/) and uses the default
success/failure criteria.

```yaml
steps:
# ...
- uses: http
  as: cat-facts
  config:
    method: GET
    url: https://www.catfacts.net/api/
    outputs:
    - name: status
      fromExpression: response.status
    - name: fact1
      fromExpression: response.body.facts[0]
    - name: fact2
      fromExpression: response.body.facts[1]
```

Assuming a `200` response with the following JSON body:

```json
{
    "facts": [
        {
            "fact_number": 1,
            "fact": "Kittens have baby teeth, which are replaced by permanent teeth around the age of 7 months."
        },
        {
            "fact_number": 2,
            "fact": "Each day in the US, animal shelters are forced to destroy 30,000 dogs and cats."
        }
    ]
}
```

The step would succeed and produce the following outputs:

| Name     | Type | Value |
|----------|------|-------|
| `status` | `int` | `200` |
| `fact1` | `string` | `Kittens have baby teeth, which are replaced by permanent teeth around the age of 7 months.` |
| `fact2` | `string` | `Each day in the US, animal shelters are forced to destroy 30,000 dogs and cats.` |

### Polling

Building on the [basic example](#basic-usage), this configuration defines
explicit success and failure criteria. Any response meeting neither of these
criteria will result in the step reporting a result of `Running` and being
retried.

Note the use of [retry](../15-promotion-templates.md#step-retries) configuration
to set a timeout for the step.

```yaml
steps:
# ...
- uses: http
  as: cat-facts
  retry:
    timeout: 10m
  config:
    method: GET
    url: https://www.catfacts.net/api/
    successExpression: response.status == 200
    failureExpression: response.status == 404
    outputs:
    - name: status
      fromExpression: response.status
    - name: fact1
      fromExpression: response.body.facts[0]
    - name: fact2
      fromExpression: response.body.facts[1]
```

Our request is considered:

- Successful if the response status is `200`.
- A failure if the response status is `404`.
- Running if the response status is anything else. i.e. Any other status code
  will result in a retry.

### Posting to Slack

This example is adapted from
[Slack's own documentation](https://api.slack.com/tutorials/tracks/posting-messages-with-curl),
showing how to post a message to a Slack channel.

```yaml
vars:
- name: slackChannel
  value: C123456
steps:
# ...
- uses: http
  config:
    method: POST
    url: https://slack.com/api/chat.postMessage
    headers:
    - name: Authorization
      value: Bearer ${{ secret('slack').token }}
    - name: Content-Type
      value: application/json
    body: |
      ${{ quote({
        "channel": vars.slackChannel,
        "blocks": [
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": "Hi I am a bot that can post *_fancy_* messages to any public channel."
            }
          }
        ]
      }) }}
```

[expr-lang]: https://expr-lang.org/