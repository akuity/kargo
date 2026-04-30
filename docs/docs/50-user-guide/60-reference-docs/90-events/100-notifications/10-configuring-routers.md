---
description: How to configure and use EventRouters for notifications
---

# Configuring Event Routers

<span class="tag professional"></span>
<span class="tag beta"></span>

[Event Routers](./00-overview.md#event-routers) are used to route events to different destinations
based on specified criteria. This document provides guidance on how to configure and use Event
Routers in Kargo.

:::info

Before creating Event Routers, ensure that you or your administrator have set up the necessary
[`ClusterMessageChannel`](../../../../40-operator-guide/35-cluster-configuration.md#cluster-message-channels)
or [`MessageChannel`](../../../20-how-to-guides/20-working-with-projects.md#message-channels)
resources that define the destinations for your notifications.

:::

## Managing Event Routers

You can manage Event Routers through the Kargo UI or declaratively via YAML configuration files
deployed by Argo CD. To manage it in the UI, go to your Project settings, then in the sidebar,
select "Notifications." In this section you can create, update, and delete both your channels and
your event routers.

:::note

Currently, event routers can only be created by editing YAML, which you can do in the UI. We plan to
improve this experience in future releases.

:::

The simplest form of an event router configuration looks like this:

```yaml
kind: EventRouter
apiVersion: ee.kargo.akuity.io/v1alpha1
metadata:
  name: promotion-status
  namespace: kargo-demo
spec:
  types:
    - PromotionFailed
    - PromotionErrored
  channels:
    - name: devops-team-slack
      kind: MessageChannel
```

In this example, the event router named `test-router` is configured to listen for two [event
types](../10-event-reference.md#event-types): `PromotionCreated` and `PromotionErrored`. When either
of these events occurs, a notification will be sent to the channel named `slack`, which is of kind
`MessageChannel` using a default message template defined for the type of event along with the
channel type. You can listen for multiple event types and send notifications to multiple channels by
adding more entries to the `types` and `channels` lists.

:::warning

If you're using default message templates, the specified event types _must_ all be of the same
"class" (such as Promotion events, Freight events, etc.). Mixing event types from different classes
is not supported when using default templates and will error.

:::

Additionally, you will likely want to make use of filters to control when notifications are sent.
This is done by setting the `when` field in the event router spec, which uses
[expr-lang](https://expr-lang.org/) expressions (see [the message formatting
docs](./20-message-formatting.md) for more information on available data). The expression must
resolve to a boolean value. For example, to only send notifications for promotions targeting a
specific stage, you could configure the event router like this:

```yaml
kind: EventRouter
apiVersion: ee.kargo.akuity.io/v1alpha1
metadata:
  name: prod-promotions
  namespace: kargo-demo
spec:
  types:
    - PromotionCreated
    - PromotionErrored
  channels:
    - name: devops-team-slack
      kind: MessageChannel
  when: "event.stageName == 'production'"
```

<!-- NOTE: We should add a supademo of how to create an Event Router once the UI stabilizes a bit -->

## Advanced Configuration

### Message Threading/Grouping

With many types of notifications, it can be helpful to group related messages together in a thread
or conversation based on items like the freight being promoted or specific items like a tag name or
commit ID. This is especially useful for channels like Slack where threaded messages help keep
discussions organized. To enable message grouping, you can use the `groupingKey` field in the event
router spec.

Currently, the `groupingKey` feature is only used by Slack channels, but this will be expanded to
other channel types in future releases.

A `groupingKey` is a string value that can be a plain string or constructed using expressions
enclosed in `${{ }}` which render to a string. The value of the `groupingKey` is used by the message
channel to determine which messages should be grouped together. All of the same context is used to
evaluate the expression as is used in things like message formatting (see see the [message
formatting documentation](./20-message-formatting.md) for more information). For example, to group
messages by the freight name and stage being promoted to, you could configure the event router like
this:

```yaml
kind: EventRouter
apiVersion: ee.kargo.akuity.io/v1alpha1
metadata:
  name: promotion-status
  namespace: kargo-demo
spec:
  groupingKey: "${{ event.freight.stageName }}-${{ event.freight.name }}"
  types:
    - PromotionFailed
    - PromotionErrored
  channels:
    - name: devops-team-slack
      kind: MessageChannel
```

:::warning

Please note that this feature currently behaves in the same way as Argo CD Notifications in that the
current state (i.e. which thread is mapped to a grouping key) is stored in memory. This means that
if the Kargo controller is restarted, the mapping will be lost and new threads may be created for
existing grouping keys. We will be adding persistent storage for this in a future release.

:::

### Custom Templates

The default templates and `when` field will likely cover many use cases, but you can also customize
the message content and formatting by specifying a custom message body. This allows you to tailor
the notifications to your specific needs. 

The top level fields for configuring a custom message body are:

- `output`: The main content of the message for all channels (unless overridden in channel-specific
  fields).
- `encodingType`: The format of the `output` field. Available options include plaintext (an empty
  string or omitted value), `json`, `yaml`, and `xml`.

The output can be customized using [`expr-lang`](https://expr-lang.org/) expressions to include
dynamic content based on the event data. These expressions must be enclosed in `${{ }}`. For
example, to include the stage name in the message, you could use `${{ event.stageName }}`. For
information on available data and formatting options, see the [message formatting
documentation](./20-message-formatting.md).

:::warning

If you are using the top level `output` field, in most cases the output should be a string as each
message type can generate a default message from the output in the proper format for that channel.
In other words, if you are sending to both SMTP and Slack, the structured data each would expect is
different so structured data will need to be provided for each channel type. However, if you are
sending to multiple channels of the same type (e.g., multiple Slack channels), you can provide a
common structured output that works for all of them.

:::

You can also specify channel-specific message bodies by providing `output` fields within the channel
configuration list. This allows you to customize the message content for each channel independently.
For example:

```yaml
kind: EventRouter
apiVersion: ee.kargo.akuity.io/v1alpha1
metadata:
  name: promotion-started
  namespace: kargo-demo
spec:
  types:
    - PromotionCreated
  channels:
    - name: slack
      kind: MessageChannel
      output: "Kargo has kicked off promotion to stage: ${{ event.stageName }}."
    - name: smtp
      kind: MessageChannel
      encodingType: yaml
      output: |
        subject: 🚀 Deployment Promotion Started
        to: email@example.com
        body: Kargo has kicked off promotion to stage: ${{ event.stageName }}.
```

For information on available data and formatting options, see the [message formatting
documentation](./20-message-formatting.md).

