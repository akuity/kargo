---
description: Message formatting options for Kargo notifications.
---

# Message Formatting

<span class="tag professional"></span>
<span class="tag beta"></span>

## Overview

`EventRouter`s have support for customizing the format of notification messages using
[`expr-lang`](https://expr-lang.org/). This allows you to create richly formatted messages tailored
to the specific channel you are sending notifications to in addition to the data available from the
event payload.

### Using expr-lang

Within the `when` field, the entire string is treated as an expression and must evaluate to a
boolean. In other fields where expressions are supported, such as the `message` field in the
[`send-message`](../../30-promotion-steps/send-message.md) step or the `output` field in an
[`EventRouter`](./00-overview.md#event-routers), expressions must be enclosed in `${{ }}`.

### Encoding Types

If sending richly formatted messages, the `encodingType` field specifies the format of the message
body. Supported values include:

- An empty string or omitted value: Plaintext format. The message is treated as plain text and sent
  verbatim.
- `json`: JSON format. The message body is expected to be a valid JSON structure.
- `yaml`: YAML format. The message body is expected to be a valid YAML structure.
- `xml`: XML format. The message body is expected to be a valid XML structure.

## Event Data Structures

For `EventRouter`s and notifications, the event data structure varies depending on the type of event
being handled. However, the object passed to expressions does have 3 common top-level fields:

- `type`: A string representing the type of event (e.g., `PromotionCreated`).
- `event`: An object containing the event-specific data. The structure of this object varies based
  on the event type. See [Event Reference](../10-event-reference.md) for more details.
- `data`: The Kubernetes object associated with the event, if applicable. This is typically the
  resource that triggered the event such as `Promotion` or `Freight`. It is fetched in its entirety
  from the Kubernetes API and kept in its raw form. So all fields will be accessible as if you were
  accessing items in a JSON object (e.g., `data.metadata.name`, `data.spec.stage`, etc.).
- `ctx`: Optional. Extra fields injected by the controller, for example `ctx.uiBaseURL` for UI links formatting.

## Available Functions

All [built-in functions](https://expr-lang.org/docs/language-definition) provided by `expr-lang` are
available for use in expressions. Additionally, the following custom functions are provided
specifically for Kargo notifications:

- `deref(any)` - Dereferences pointers or references within the event data structure, returning the
  underlying value. This is useful when printing a value. It is not necessary to use when accessing
  fields under the pointer.
- `escape_json_string(string)` - Escapes special characters in a string to make it safe for
  inclusion in JSON. It is highly recommended to use this function when adding things like the event
  message to a JSON formatted message to avoid invalid JSON.

## Message Body Formats

### Slack

Slack messages can be formatted using JSON with the following structure:

| Key          | Type   | Description                                                                                                                                          |
| ------------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `icon_emoji` | string | An emoji to use as the icon for the message. Overrides `icon_url` if both are set. Should be in the format `:emoji_name:`                            |
| `icon_url`   | string | A URL to an image to use as the icon for the message                                                                                                 |
| `blocks`     | array  | Rich text message content, represented as Slack "blocks". See [Slack Block Kit documentation](https://docs.slack.dev/block-kit) for more information |
| `channel`    | string | The Slack channel ID to send the message to. If not set, the channel ID from the channel spec will be used                                           |

#### Example Slack Message Body

```json
{
  "icon_emoji": ":rocket:",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "🚀 A new promotion has been created for stage *${{ event.stageName }}* with freight *${{ event.freight.alias }}*!"
      }
    }
  ],
  "channel": "C1234567890"
}
```

### SMTP

SMTP messages can be formatted using JSON with the following structure:

| Key       | Type    | Description                                                                                              |
| --------- | ------- | -------------------------------------------------------------------------------------------------------- |
| `subject` | string  | The subject line of the email                                                                            |
| `body`    | string  | The body content of the email                                                                            |
| `to`      | array   | An array of recipient email addresses                                                                    |
| `html`    | boolean | Whether the body should be interpreted as HTML. If `false` or omitted, the body is treated as plain text |

#### Example SMTP Message Body

```json
{
  "subject": "🚀 New Promotion Created",
  "body": "<h1>New Promotion Created</h1><p>A new promotion has been created for stage <strong>${{ event.stageName }}</strong> with freight <strong>${{ event.freight.alias }}</strong>!</p>",
  "to": [
    "recipient@example.com"
  ],
  "html": true 
}
```
