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
| `responseContentType` | `string` | N | Overrides automatic content-type detection for response parsing. Accepts `application/json`, `application/yaml`, or `text/plain`. When not set, the step uses the response's `Content-Type` header, falling back to JSON parsing for unrecognized types. |
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

:::warning

Expressions in the `successExpression` and `failureExpression` fields must _not_
be enclosed by `${{` and `}}` since all such expressions are evaluated _prior_
to step execution. (i.e. All steps are _actually_ executed against static,
pre-evaluated configuration!) Since these two expressions are intended to be
evaulated _internally_ by the `http` step, and only after receiving an HTTP
response, not enclosing them within `${{` and `}}` prevents premature evaluation
and ensures they are passed to the `http` step exactly as they've been written.

If your `successExpression` or `failureExpression` need to reference variables
or output from previous steps, use expressions that _are_ enclosed by `${{`
and `}}` _within_ those expressions. The "inner expressions" will be evaluated
prior to step execution, while the "outer expressions" will be evaluated by the
step itself.

Consider the following:

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

After expressions enclosed within `${{` and `}}` have been pre-evaluated, the
`http` step defined above is executed against the following _now static_
configuration:

```yaml
{
  "failureExpression": "response.body.status == 'failed'",
  "successExpression": "response.body.status == 'completed'",
  "url": "https://api.example.com/status"
}
```

Internally, the step evaluates the `successExpression` and `failureExpression`
exactly as if the user had written them as they now appear.

:::

A `response` object (a `map[string]any`) is available to these expressions. It
is structured as follows:

| Field | Type | Description |
|-------|------|-------------|
| `status` | `int` | The HTTP status code of the response. |
| `headers` | `http.Header` | The headers of the response. See applicable [Go documentation](https://pkg.go.dev/net/http#Header). |
| `header` | `func(string) string` | `headers` can be inconvenient to work with directly. This function allows you to access a header by name. |
| `body` | `any` | The parsed response body. For JSON responses (`application/json`), this can be any valid JSON value: objects, arrays, strings, numbers, booleans, or `null`. For YAML responses (`application/yaml`, `text/yaml`, `application/x-yaml`), this is the unmarshaled YAML structure. For `text/plain` responses, this is the raw string. For unrecognized content types, the step attempts JSON parsing and falls back to an empty map if the content is not valid JSON. Empty responses result in an empty map. |

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

### YAML Responses

This example demonstrates working with an API that returns YAML-formatted data.

```yaml
steps:
# ...
- uses: http
  as: config-fetch
  config:
    method: GET
    url: https://api.example.com/config.yaml
    successExpression: response.status == 200
    outputs:
    - name: version
      fromExpression: response.body.version
    - name: features
      fromExpression: response.body.features
```

If the server returns a `200` response with `Content-Type: application/yaml` and
the following body:

```yaml
version: "2.0"
features:
  - name: feature-a
    enabled: true
  - name: feature-b
    enabled: false
```

The step would succeed and produce structured outputs that can be accessed in
subsequent steps.

#### Overriding Content-Type Detection

Some APIs return structured data with incorrect or generic content types. Use
`responseContentType` to explicitly specify how to parse the response:

```yaml
steps:
- uses: http
  config:
    url: https://api.example.com/data
    responseContentType: application/yaml  # Force YAML parsing
    outputs:
    - name: value
      fromExpression: response.body.key
```

[expr-lang]: https://expr-lang.org/
