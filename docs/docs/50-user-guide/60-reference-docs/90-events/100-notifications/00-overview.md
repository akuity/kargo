---
description: An overview of Kargo notifications concepts and components.
---

# Key Concepts

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info

This set of features are only available in Kargo on the [Akuity
Platform](https://akuity.io/akuity-platform), versions v1.8 and above.

:::

This document provides an overview of the key concepts related to Kargo's Notifications feature,
explaining the main components involved in sending notifications based on events.

## Channels

Channels define the destination and method for sending notifications. They specify where the
notifications should be sent, such as to a Slack channel, email address, or other supported platforms.
Channels are configured using [`MessageChannel`](../../../20-how-to-guides/20-working-with-projects.md#message-channels)
or [`ClusterMessageChannel`](../../../../40-operator-guide/35-cluster-configuration.md#cluster-message-channels)
resources. Each channel type may have its own specific configuration options, such as authentication details,
recipient information, and message formatting preferences.

You can think of channels as a "connection string" of sorts that tells Kargo where to send data.
They do not define the message or other information, only the connection details. They are used by
[`EventRouter`s](#event-routers) and by the [`send-message`
step](../../30-promotion-steps/send-message.md) to send notifications when specific events occur.

## Event Routers

At their core, `EventRouter`s are responsible for listening to specific events within Kargo and
routing them to the appropriate channels based on defined criteria. They act as the bridge between
events and channels, ensuring that notifications are sent to the right destinations when relevant
events occur. Due to their design, the same event can be routed to multiple channels or different
events can be routed to the same channel without duplicating notification logic.

The lack of "notification" in the name is intentional, as `EventRouter`s do not themselves send
notifications. Instead, they route events to channels after rendering data from the event, which
then handle the actual sending of notifications. This design allows for greater flexibility and
reusability, as you can glue together any type of message or data you are interested in without
confining it to a specific notification-only format. There is also the possibility (but not
guarantee) that in the future we may implement routing of events to other systems beyond just
notifications.
