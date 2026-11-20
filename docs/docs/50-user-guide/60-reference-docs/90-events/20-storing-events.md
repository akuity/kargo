---
sidebar_label: Storing Events
description: Keep a short-lived history of Kargo events in a NATS JetStream stream and query it.
---

# Storing Events with JetStream

Kargo publishes its events as [CloudEvents](https://cloudevents.io/) to [NATS](https://nats.io/) in
version 2.0. Out of the box, those events are fire-and-forget: if nothing is subscribed when an
event is published, it is gone. That is great for things that react to events in real time, but not
so great if you want to go back and see what happened 20 minutes ago.

With the emitted Kubernetes events in 1.0, you got a bit of history for free, since the Kubernetes API
server keeps events around for an hour by default. This page walks through how to get that same
behavior back with a [JetStream](https://docs.nats.io/nats-concepts/jetstream) stream that stores
every Kargo event in memory for one hour, and then how to query it.

## How Events Map to Subjects

Every event is published on a subject under `akuity.kargo.events`. The rest of the subject comes
from the event's type, with each word of the type becoming its own (lowercase) token:

| Event Type                     | Subject                                              |
| ------------------------------ | ---------------------------------------------------- |
| `PromotionCreated`             | `akuity.kargo.events.promotion.created`              |
| `FreightVerificationSucceeded` | `akuity.kargo.events.freight.verification.succeeded` |
| `APITokenCreated`              | `akuity.kargo.events.api.token.created`              |

Because subjects have a varying number of tokens, we use the NATS `>` wildcard (which matches one or
more tokens) rather than `*` (which matches exactly one). So `akuity.kargo.events.>` captures every
event, while something like `akuity.kargo.events.freight.verification.>` gets you only verification
outcomes. See the [event reference](./10-event-reference.md) for the full list of event types.

Each message is a structured-mode CloudEvent, meaning the whole event (attributes and all) is the
JSON payload. A `PromotionSucceeded` event looks like this:

```json
{
  "specversion": "1.0",
  "id": "01a15f23-9228-7b7f-930c-9f9ccf5c106d",
  "source": "/kargo/promotion-controller",
  "type": "PromotionSucceeded",
  "subject": "akuity.kargo.events.promotion.succeeded",
  "datacontenttype": "application/json",
  "time": "2026-10-21T15:45:29.128721305Z",
  "kind": "Promotion",
  "data": {
    "project": "events-demo",
    "actor": "controller:promotion-controller",
    "message": "Promotion Succeeded",
    "name": "dev.01m5fj74gga1wv53vakaepjxq9.beae948",
    "stageName": "dev",
    "createTime": "2026-10-21T15:45:29Z",
    "verificationPending": false,
    "freight": {
      "name": "beae948372c00e99af7a1af1188560630d5f88c6",
      "alias": "torrid-squirrel",
      "warehouseName": "nginx",
      "images": [
        {
          "repoURL": "public.ecr.aws/nginx/nginx",
          "tag": "1.31.7",
          "digest": "sha256:aa221d5fa45aab5c901e245bc653a537aecfdf7977169fda8b0b41200ddd3223"
        }
      ]
    }
  }
}
```

## Running the NATS CLI

The easiest way to talk to the NATS server installed by the chart is to start a throwaway
[`nats-box`](https://github.com/nats-io/nats-box) pod in the same namespace as Kargo. It comes with
the `nats` CLI (and `jq`) already installed:

```shell
kubectl run nats-box --rm -it --restart=Never \
  --namespace kargo \
  --image natsio/nats-box:0.19.7 \
  --env NATS_URL=nats://kargo-nats:4222
```

Once you see the `nats-box:~#` prompt, you're good to go. The pod is deleted as soon as you exit the
shell. All of the commands in the rest of this page are run from that prompt.

:::info

Swap `kargo` for whatever namespace you installed Kargo into. If you're using your own NATS server,
set `NATS_URL` to its address instead and pass along whatever credentials it requires (e.g. `--nkey`
with the path to an nkey seed).

:::

## Creating the Stream

Creating the stream is a single command:

```shell
nats stream add KARGO_EVENTS \
  --subjects 'akuity.kargo.events.>' \
  --storage memory \
  --retention limits \
  --discard old \
  --max-age 1h \
  --max-bytes 200MiB \
  --replicas 3 \
  --defaults
```

Here's what each of those flags is doing:

- `--subjects 'akuity.kargo.events.>'` captures every Kargo event (the quotes keep your shell from
  trying to do anything with the `>`).
- `--storage memory` keeps everything in memory. Nothing survives a restart of the NATS servers,
  which is the same tradeoff Kubernetes events made.
- `--retention limits` with `--discard old` means messages stay in the stream until they hit one of
  the limits below, at which point the oldest ones are dropped. Reading a message does _not_ remove
  it.
- `--max-age 1h` drops events once they are an hour old.
- `--max-bytes 200MiB` caps the size of the stream so it fits in the JetStream memory store (256Mi
  with the chart's defaults). Without it, a busy Kargo instance could fill the memory store, at
  which point NATS stops accepting new events into the stream instead of dropping old ones.
- `--replicas 3` matches the three-server NATS cluster the chart installs by default, so the stream
  survives losing a server. If you're running a single NATS server, use `--replicas 1`.
- `--defaults` accepts the defaults for everything else instead of prompting for it.

You can check on the stream (including how many messages it has) at any time
with:

```shell
nats stream info KARGO_EVENTS
```

:::note

A stream only captures events published _after_ it is created. Anything that happened before then is
not coming back.

:::

## Querying the Stream

There are two main ways to get events back out of the stream. Ad-hoc queries are great for poking
around and debugging. Consumers are what you want when something needs to keep track of which events
it has already seen.

### Ad-hoc Queries

To get a quick overview of which kinds of events are in the stream (and how many of each), list its
subjects:

```shell
nats stream subjects KARGO_EVENTS
```

```plaintext
╭────────────────────────────────────────────────────────────╮
│              4 Subjects in stream KARGO_EVENTS             │
├────────────────────────────────────────────────────┬───────┤
│ Subject                                            │ Count │
├────────────────────────────────────────────────────┼───────┤
│ akuity.kargo.events.freight.verification.succeeded │ 1     │
│ akuity.kargo.events.promotion.created              │ 1     │
│ akuity.kargo.events.promotion.succeeded            │ 1     │
│ akuity.kargo.events.freight.created                │ 1     │
╰────────────────────────────────────────────────────┴───────╯
```

To page through events interactively, use `nats stream view`. The `--subject` flag narrows it down
to the events you care about:

```shell
nats stream view KARGO_EVENTS --subject 'akuity.kargo.events.promotion.>'
```

If all you need is the most recent event of a given type, grab it directly:

```shell
nats stream get KARGO_EVENTS --last-for akuity.kargo.events.freight.created
```

For anything more involved, `nats sub` can replay events out of the stream. Pair `--raw` (which
prints only the event JSON) with `-T` (which exits once it reaches the end of the stream) and you
can pipe the results into `jq`. For example, to see every Promotion event from the last 30 minutes:

```shell
nats sub 'akuity.kargo.events.promotion.>' --stream KARGO_EVENTS --since 30m -T --raw \
  | jq -c '{type, time, stage: .data.stageName, message: .data.message}'
```

```plaintext
{"type":"PromotionCreated","time":"2026-10-21T15:45:29.113718922Z","stage":"dev","message":"Automatically promoted Freight from origin \"Warehouse/nginx\" for Stage \"dev\""}
{"type":"PromotionSucceeded","time":"2026-10-21T15:45:29.128721305Z","stage":"dev","message":"Promotion Succeeded"}
```

Or, to see the type of every event in the stream for a single Project:

```shell
nats sub 'akuity.kargo.events.>' --stream KARGO_EVENTS --all -T --raw \
  | jq -c 'select(.data.project == "events-demo") | .type'
```
