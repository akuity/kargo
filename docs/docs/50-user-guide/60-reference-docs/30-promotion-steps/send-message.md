---
sidebar_label: send-message
description: Send a message to a specified destination, such as Slack or email, during the promotion process.
---

<span class="tag professional"></span>
<span class="tag beta"></span>
# `send-message`

:::info

This promotion step is only available in Kargo on the [Akuity
Platform](https://akuity.io/akuity-platform), versions v1.8 and above.

:::

The `send-message` step allows you to send messages to various destinations, such as Slack channels
or email addresses, during the promotion process. This can be useful for notifying team members
about promotion events, approvals, or other important information. This step is useful when you want
to imperatively send a notification as part of your promotion workflow. For an event-driven option
that can send notifications based on specific events, consider using the [Notifications
feature](../90-events/100-notifications/index.md).

This feature is evolving quickly and more configuration options and destinations will continue to be
added in future releases.

## Configuration

| Name              | Type      | Required | Description                                                                                                                                                                                                                                                                               |
| ----------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `channel`         | `object`  | Y        | Configuration for the destination channel where the message will be sent.                                                                                                                                                                                                                 |
| `channel.kind`    | `string`  | Y        | The type of [channel](../../20-how-to-guides/20-working-with-projects.md#message-channels) to send the message to. Supported values are `MessageChannel` and `ClusterMessageChannel`                                                                                                      |
| `channel.name`    | `string`  | Y        | The name of the [channel](../../20-how-to-guides/20-working-with-projects.md#message-channels) to send the message to. This should correspond to a configured channel in your Kargo instance.                                                                                             |
| `message`         | `string`  | Y        | The content of the message to be sent. This can include plain text or [richly formatted content](#sending-richly-formatted-messages).                                                                                                                                                     |
| `encodingType`    | `string`  | N        | The encoding format of the message. Used when sending [richly formatted messages](#sending-richly-formatted-messages). If omitted or an empty string, it means the message is plaintext and should be used verbatim. Supported values are `json`, `yaml`, and `xml`.                      |
| `slack`           | `object`  | N        | Optional configuration specific to Slack messages.                                                                                                                                                                                                                                        |
| `slack.channelID` | `string`  | N        | The ID of the Slack channel to send the message to. If omitted, the default channel configured in the Slack channel configuration will be used.                                                                                                                                           |
| `slack.threadTS`  | `string`  | N        | The timestamp of the Slack thread to send the message in. If omitted, the message will be sent as a new message in the channel. This can be set by using `set-metadata` and `freightMetadata()` to ensure you set and fetch the timestamp according to some criteria (such as freight ID) |
| `smtp`            | `object`  | N        | Optional configuration specific to email messages.                                                                                                                                                                                                                                        |
| `smtp.to`         | `string`  | N        | The email address to send the message to. If omitted, the default recipient configured in the SMTP channel configuration will be used.                                                                                                                                                    |
| `smtp.subject`    | `string`  | N        | The subject line for the email. If omitted, a default subject will be used.                                                                                                                                                                                                               |
| `smtp.html`       | `boolean` | N        | Whether the email body should be sent as HTML. If `true`, the message will be treated as HTML content. If `false` or omitted, the message will be treated as plain text.                                                                                                                  |

### Sending richly formatted messages

By default, the `send-message` step will use the given message and send a plain formatting message
to the configured channel. However, in some cases you may want to send a richly formatted message,
such as a Slack message with [blocks](https://docs.slack.dev/block-kit/). To do this, you can use
the `encodingType` field to specify the format of the message. The supported structures for each
message type can be found in the [Notification
documentation](../90-events/100-notifications/20-message-formatting.md). See the example section
below for an example of sending a richly formatted Slack message.

:::warning

If `encodingType` is set for a message, it assumes that all configuration options will be passed in
the encoded body passed as the `message` field (see [the Slack example
below](#sending-a-richly-formatted-slack-message)) rather than using the config specific fields such
as `slack.channelID` or `smtp.html`. Any options that are set in the `config` block other than
`message` and `encodingType` will be ignored.

:::

Please note that if you use `json` encoding as a message format, you must ensure that the JSON is
properly quoted. The step runner assumes that any JSON content should be decoded, so unquoted JSON
will result in an error. This can be done like so:

```yaml
steps:
- uses: send-message
  config:
    channel:
      kind: MessageChannel
      name: smtp
    encodingType: json
    message: |
      ${{ quote({
        "subject": "🚀 Deployment Promotion Started",
        "body": "Kargo has kicked off promotion to stage: " + ctx.stage,
        "to": [
          "email@example.com"
        ],
        "html": false
      }) }}
```

## Outputs

The `send-message` step returns different outputs depending on the type of channel used. Each output
is nested under a key corresponding to the channel kind (e.g., `slack`, `smtp`).

### Slack Output

When sending a message to a Slack channel, the output will include the following fields:

| Name       | Type   | Description                                                                                                                         |
| ---------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| `threadTS` | string | When starting a new thread, the timestamp of the newly created message. When replying to an existing thread, the provided threadTS. |

### SMTP Output

Currently SMTP messages do not return any specific outputs. This may change in future releases.

## Examples

### Sending a plain text message to Slack

```yaml
steps:
- if: ${{ always() }}
  uses: send-message
  config:
    channel:
      kind: MessageChannel
      name: slack
    message: "Kargo promotion to ${{ ctx.stage }} ${{ success() ? 'succeeded' : 'failed' }}"
```

### Overriding the Slack channel ID

```yaml
steps:
- uses: send-message
  config:
    channel:
      kind: MessageChannel
      name: slack
    message: "Kargo has kicked off promotion to stage: ${{ ctx.stage }}."
    slack:
      channelID: C1234567890
```

### Sending an email with all custom options

```yaml
steps:
- uses: send-message
  config:
    channel:
      kind: MessageChannel
      name: smtp
    message: "<marquee>Kargo has kicked off promotion to stage: ${{ ctx.stage }}.</marquee>"
    smtp:
      to: [foo@bar.com]
      subject: "🚀 Deployment Promotion Started"
      html: true
```

### Sending a richly formatted Slack message

This shows an example of using inline YAML to send a richly formatted Slack message with blocks. The
`encodingType` is set to `yaml` to indicate that the message content is in YAML format. Note that
the `channel` is also set in the body as all to override the default channel configured in the Slack
channel configuration.

```yaml
steps:
- uses: send-message
  config:
    channel:
      kind: MessageChannel
      name: slack
    message: |
      blocks:
        - type: section
          text:
            type: mrkdwn
            text: "*Kargo Promotion Started* :rocket:"
        - type: section
          fields:
            - type: mrkdwn
              text: "*Stage:*\n${{ ctx.stage }}"
            - type: mrkdwn
              text: "*Time:*\n${{ ctx.timestamp }}"
      icon_emoji: ":fire:"
      channel: C1234567890
    encodingType: yaml
```

### Sending a message to a thread in Slack

This shows an example of sending a message to a specific thread in Slack by using the `threadTS` field.
It leverages the `set-metadata` step to store and retrieve the thread timestamp associated with a specific
freight. This allows messages related to the same freight to be grouped together in a single thread.

```yaml
steps:
- uses: send-message
  as: send-slack
  config:
    channel:
      kind: MessageChannel
      name: slack
    message: "Kargo has kicked off promotion to stage: ${{ ctx.stage }}."
    slack:
      # NOTE: Make sure to wrap this in quotes with `quote` otherwise it will get rendered as
      # a number
      threadTS: "${{ quote(freightMetadata(ctx.targetFreight.name)?.slackThreadTS ?? '') }}"
- uses: set-metadata
  config:
    updates:
      - kind: Freight
        name: ${{ ctx.targetFreight.name }}
        values:
          # NOTE: Make sure to wrap this in quotes with `quote` otherwise it will get rendered
          # as a number
          slackThreadTS: "${{ quote(outputs['send-slack']?.slack?.threadTS ?? '') }}"
```
