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
| `successExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine success. When defined, the step succeeds only when this expression evaluates to `true`. If both `successExpression` and `failureExpression` are defined and both evaluate to `true`, the failure takes precedence and the step fails terminally. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `failureExpression` | `string` | N | An [expr-lang] expression that can evaluate the response to determine failure. When defined and evaluates to `true`, the step fails terminally. If both `successExpression` and `failureExpression` are defined and both evaluate to `true`, the failure takes precedence. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |
| `outputs` | `[]object` | N | A list of rules for extracting outputs from the HTTP response. These are only applied to responses deemed successful. |
| `outputs[].name` | `string` | Y | The name of the output. |
| `outputs[].fromExpression` | `string` | Y | An [expr-lang] expression that can extract a value from the HTTP response. Note that this expression should _not_ be offset by `${{` and `}}`. See examples for more details. |

## Success and Failure Determination

The step's outcome is determined by evaluating the success and failure criteria
as follows:

- If `failureExpression` is defined and evaluates to `true`, the step
  **fails terminally** (no retries).
- If `successExpression` is defined and evaluates to `true` (and failure
  criteria are not met), the step **succeeds**.
- If neither expression is defined: **2xx status codes** succeed,
  **non-2xx status codes** fail but will be retried.
- All other cases result in **Running** and will be retried.

:::note
The key distinction is between **terminal failures** (when `failureExpression`
evaluates to `true`) and **retried failures** (all other failure cases).
Terminal failures stop the promotion immediately, while retried failures allow
Kargo to retry the step according to the configured
[retry policy](../15-promotion-templates.md#step-retries).
:::

## Expressions

The `successExpression`, `failureExpression`, and `outputs[].fromExpression`
fields all support [expr-lang][] expressions.

:::note
The expressions included in the `successExpression`, `failureExpression`, and
`outputs[].fromExpression` fields should _not_ be offset by `${{` and `}}`. This
is to prevent the expressions from being evaluated by Kargo during
pre-processing of step configurations. The `http` step itself will evaluate
these expressions after receiving the HTTP response.

However, if you need to use variables or outputs from other steps within these
expressions, you can wrap those specific values in `${{ }}` so they are
pre-evaluated and substituted before the configuration reaches the step:

```yaml
vars:
- name: expectedStatus
  value: completed
steps:
- uses: http
  config:
    url: https://api.example.com/status
    successExpression: response.body.status == '${{ vars.expectedStatus }}'
    failureExpression: response.body.status == 'failed'
```

At runtime, `${{ vars.expectedStatus }}` is replaced with `'completed'`, so the
`http` step receives:

```yaml
successExpression: response.body.status == 'completed'
```
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

- **Successful** if the response status is `200`.
- **A terminal failure** if the response status is `404`.
- **Running** (will be retried) if the response status is anything else.

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
