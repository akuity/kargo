---
description: Reference documentation for Kargo events and types.
---

# Kargo Event Reference

This document contains a complete reference of all Kargo events, including their types and
descriptions.

:::info

Kargo publishes every event as a [CloudEvent](https://cloudevents.io/) to the
[NATS](https://nats.io/) server it is configured with. Events are not stored by default. If you want
to keep a history of them, see [Storing Events with JetStream](./20-storing-events.md). These events
are also used by the [Notification feature](./100-notifications/index.md)

:::

## Event Envelope

Each event is published in CloudEvents structured content mode, meaning the whole event (its
attributes and its data) is the JSON payload of the NATS message. The message also carries a
`Content-Type: application/cloudevents+json` header.

Every event has the following attributes:

| Attribute         | Description                                                                                                                             |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `specversion`     | The CloudEvents spec version. Always `1.0`.                                                                                             |
| `id`              | A unique identifier for the event. This is a UUIDv7, so sorting by `id` also sorts events by when they were created.                    |
| `source`          | The Kargo component that published the event, in the form `/kargo/<component>` (see [Event Sources](#event-sources)).                   |
| `type`            | The event type (e.g. `PromotionSucceeded`). See [Event Types](#event-types) for the full list.                                          |
| `subject`         | The NATS subject the event was published on (e.g. `akuity.kargo.events.promotion.succeeded`).                                           |
| `time`            | When the event was published (RFC3339).                                                                                                 |
| `datacontenttype` | The content type of `data`. Always `application/json`.                                                                                  |
| `kind`            | The kind of Kubernetes object the event is about (`Freight`, `Promotion`, or `Secret`), so it can be looked up without decoding `data`. |
| `data`            | The event's payload, described in [Event Fields](#event-fields).                                                                        |

### Event Subjects

Events are published on subjects under `akuity.kargo.events`. The rest of the subject comes from
the event type, with each word of the type lowercased and turned into its own token. For example,
`FreightVerificationSucceeded` is published on
`akuity.kargo.events.freight.verification.succeeded`. Since subjects have different numbers of
tokens, subscribe with the NATS `>` wildcard (e.g. `akuity.kargo.events.>` for every event or
`akuity.kargo.events.promotion.>` for every Promotion event).

The subject for each event type is listed in [Event Types](#event-types).

### Event Sources

The `source` attribute identifies which component published the event:

| Source                        | Component                                                          |
| ----------------------------- | ------------------------------------------------------------------ |
| `/kargo/api`                  | The API server                                                     |
| `/kargo/warehouse-controller` | The controller, while reconciling `Warehouse`s                     |
| `/kargo/stage-controller`     | The controller, while reconciling `Stage`s                         |
| `/kargo/promotion-controller` | The controller, while reconciling `Promotion`s                     |
| `/kargo/freight-webhook`      | The Kubernetes webhooks server, while handling `Freight` changes   |
| `/kargo/promotion-webhook`    | The Kubernetes webhooks server, while handling `Promotion` changes |

When a controller is configured with a shard name, the shard name is appended to its source (e.g.
`/kargo/stage-controller-us-west`).

## Event Fields

Many Kargo events share common fields. The following sections describe these common fields, which
are referenced in individual event definitions. Each field is given as it appears in the event
payload (serialized as JSON) and, if optional, notes whether the type is a pointer when used in
expr-lang expressions.

In each event definition in [Event Types](#event-types), the included fields are listed under
"Payload Includes" with references to the relevant sections. All of the fields described in those
events are found at the top level of the event's `data` unless otherwise noted.

### Common Event Fields

These fields are included in all Kargo events:

| Field Name | Type   | Description                                    | Optional          |
| ---------- | ------ | ---------------------------------------------- | ----------------- |
| `project`  | String | The project name the event originated from.    | No                |
| `actor`    | String | The user or system that triggered the event.   | Yes (is pointer)  |
| `message`  | String | A human-readable message describing the event. | No (may be empty) |

The unique identifier of an event is the `id` [attribute](#event-envelope) of the CloudEvent
rather than a field of its data.

### Freight Fields

Freight payloads describe the collection of artifacts under evaluation or promotion.

| Field Name      | Type                       | Description                                                                                                         | Optional         |
| --------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `name`          | String                     | Name of the freight object.                                                                                         | No               |
| `stageName`     | String                     | Stage associated with the freight when the event fired.                                                             | No               |
| `warehouseName` | String                     | Warehouse the freight originated from.                                                                              | Yes              |
| `createTime`    | String (RFC3339)           | Creation timestamp of the freight object.                                                                           | No               |
| `alias`         | String                     | Human-friendly alias assigned to the freight.                                                                       | Yes (is pointer) |
| `commits`       | Array\<GitCommit\>         | Git commits that compose the freight (see [GitCommit fields](#gitcommit-fields)).                                   | Yes              |
| `images`        | Array\<Image\>             | Container images included in the freight (see [Image fields](#image-fields)).                                       | Yes              |
| `charts`        | Array\<Chart\>             | Helm charts included in the freight (see [Chart fields](#chart-fields)).                                            | Yes              |
| `artifacts`     | Array\<ArtifactReference\> | Additional arbitrary artifacts included in the freight (see [ArtifactReference fields](#artifactreference-fields)). | Yes              |

#### `GitCommit` Fields

| Field Name  | Type   | Description                              | Optional |
| ----------- | ------ | ---------------------------------------- | -------- |
| `repoURL`   | String | URL of the Git repository.               | Yes      |
| `id`        | String | Commit SHA in the referenced repository. | Yes      |
| `branch`    | String | Branch where the commit was discovered.  | Yes      |
| `tag`       | String | Tag that resolved to the commit.         | Yes      |
| `message`   | String | Commit message subject line.             | Yes      |
| `author`    | String | Author of the commit.                    | Yes      |
| `committer` | String | Committer recorded for the commit.       | Yes      |

#### `Image` Fields

| Field Name    | Type                 | Description                                     | Optional |
| ------------- | -------------------- | ----------------------------------------------- | -------- |
| `repoURL`     | String               | Repository that hosts the container image.      | Yes      |
| `tag`         | String               | Mutable tag identifying a version of the image. | Yes      |
| `digest`      | String               | Immutable digest identifying the image content. | Yes      |
| `annotations` | Map\<String,String\> | Arbitrary metadata associated with the image.   | Yes      |

#### `Chart` Fields

| Field Name | Type   | Description                                                           | Optional |
| ---------- | ------ | --------------------------------------------------------------------- | -------- |
| `repoURL`  | String | Helm chart repository URL.                                            | Yes      |
| `name`     | String | Chart name within the repository (empty for OCI-style references).    | Yes      |
| `version`  | String | Specific chart version selected for inclusion in the freight payload. | Yes      |

#### `ArtifactReference` Fields

| Field Name         | Type                 | Description                                                                                                                                                                                                                                                                                                       | Optional |
| ------------------ | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| `artifactType`     | String               | Unique type of the artifact.                                                                                                                                                                                                                                                                                      | No       |
| `subscriptionName` | String               | Name of the subscription that discovered the artifact.                                                                                                                                                                                                                                                            | No       |
| `version`          | String               | Version identifies a specific revision of this artifact.                                                                                                                                                                                                                                                          | No       |
| `metadata`         | Map\<String,Object\> | Additional metadata associated with the artifact. It is a mostly opaque collection of attributes. "Mostly" because Kargo may understand how to interpret some documented, well-known, top-level keys. Those aside, this metadata is only understood by a corresponding Subscriber implementation that created it. | Yes      |

### Freight Verification Fields

Freight verification metadata accompanies events emitted while verifying freight.

| Field Name                     | Type             | Description                                                                     | Optional         |
| ------------------------------ | ---------------- | ------------------------------------------------------------------------------- | ---------------- |
| `verificationStartTime`        | String (RFC3339) | Timestamp when the verification run began.                                      | Yes (is pointer) |
| `verificationFinishTime`       | String (RFC3339) | Timestamp when the verification run finished.                                   | Yes (is pointer) |
| `analysisRunName`              | String           | Name of the Argo Rollouts AnalysisRun created for the verification, if present. | Yes (is pointer) |
| `analysisTriggeredByPromotion` | String           | Name of the promotion that triggered the verification analysis run, if present. | Yes (is pointer) |

### API Token Fields

API token payloads describe the token Secret an event is about and the Kargo Role it belongs to.

| Field Name    | Type    | Description                                                                     | Optional |
| ------------- | ------- | ------------------------------------------------------------------------------- | -------- |
| `name`        | String  | Name of the token (and its Secret).                                             | No       |
| `roleName`    | String  | Name of the Role the token is bound to.                                         | No       |
| `systemLevel` | Boolean | `true` when the Role is system-level, in which case `project` is not a Project. | No       |

### Promotion Fields

Promotion payloads describe a promotion resource and the freight it targets.

| Field Name     | Type                                                  | Description                                           | Optional         |
| -------------- | ----------------------------------------------------- | ----------------------------------------------------- | ---------------- |
| `freight`      | Object ([Freight fields](#freight-fields))            | Snapshot of the freight referenced by the promotion.  | Yes (is pointer) |
| `name`         | String                                                | Name of the promotion resource.                       | No               |
| `stageName`    | String                                                | Stage targeted by the promotion.                      | No               |
| `createTime`   | String (RFC3339)                                      | Creation timestamp of the promotion resource.         | No               |
| `applications` | Array\<[NamespacedName](#applications-entry-fields)\> | Argo CD applications resolved for the promotion step. | Yes              |
| `rollback`     | Boolean                                               | Indicates whether the promotion is a rollback.        | Yes              |

#### Applications Entry Fields

Each promotion `applications` entry is a Kubernetes `NamespacedName` tuple.

| Field Name  | Type   | Description                              | Optional |
| ----------- | ------ | ---------------------------------------- | -------- |
| `namespace` | String | Namespace that contains the Argo CD app. | No       |
| `name`      | String | Name of the Argo CD application.         | No       |

## Event Types

The complete list of built-in Kargo event types is provided below, along with the subject each is
published on and the value of its `kind` attribute:

| Event Type                        | Subject                                                 | `kind`      |
| --------------------------------- | ------------------------------------------------------- | ----------- |
| `PromotionCreated`                | `akuity.kargo.events.promotion.created`                 | `Promotion` |
| `PromotionSucceeded`              | `akuity.kargo.events.promotion.succeeded`               | `Promotion` |
| `PromotionFailed`                 | `akuity.kargo.events.promotion.failed`                  | `Promotion` |
| `PromotionErrored`                | `akuity.kargo.events.promotion.errored`                 | `Promotion` |
| `PromotionAborted`                | `akuity.kargo.events.promotion.aborted`                 | `Promotion` |
| `PromotionDiscarded`              | `akuity.kargo.events.promotion.discarded`               | `Promotion` |
| `FreightCreated`                  | `akuity.kargo.events.freight.created`                   | `Freight`   |
| `FreightApproved`                 | `akuity.kargo.events.freight.approved`                  | `Freight`   |
| `FreightVerificationSucceeded`    | `akuity.kargo.events.freight.verification.succeeded`    | `Freight`   |
| `FreightVerificationFailed`       | `akuity.kargo.events.freight.verification.failed`       | `Freight`   |
| `FreightVerificationErrored`      | `akuity.kargo.events.freight.verification.errored`      | `Freight`   |
| `FreightVerificationAborted`      | `akuity.kargo.events.freight.verification.aborted`      | `Freight`   |
| `FreightVerificationInconclusive` | `akuity.kargo.events.freight.verification.inconclusive` | `Freight`   |
| `FreightVerificationUnknown`      | `akuity.kargo.events.freight.verification.unknown`      | `Freight`   |
| `APITokenCreated`                 | `akuity.kargo.events.api.token.created`                 | `Secret`    |
| `APITokenDeleted`                 | `akuity.kargo.events.api.token.deleted`                 | `Secret`    |

`APITokenCreated` and `APITokenDeleted` have a `kind` of `Secret` because API tokens are stored as
ServiceAccount token `Secret`s.

Below are the detailed definitions for each event type.

### `PromotionCreated`

This event is emitted when a promotion resource is created.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

### `PromotionSucceeded`

This event is emitted when a promotion completes successfully. The payload matches
`PromotionCreated` with one additional field.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

Unique to this event:

| Field Name            | Type    | Description                                                                 | Optional         |
| --------------------- | ------- | --------------------------------------------------------------------------- | ---------------- |
| `verificationPending` | Boolean | Indicates whether post-promotion freight verification is still outstanding. | Yes (is pointer) |

### `PromotionFailed`

This event is emitted when a promotion fails, typically because a step or verification did not
succeed.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

### `PromotionErrored`

This event is emitted when a promotion encounters an unexpected error.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

### `PromotionAborted`

This event is emitted when a promotion run is aborted before completion.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

### `PromotionDiscarded`

This event is emitted when the control plane removes a promotion that never started running, so
that the reason it will not run is not lost along with it. Routine deletions by the garbage
collector do not emit this event; those promotions already reached a terminal phase and reported
an outcome of their own.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Promotion fields](#promotion-fields)

### `FreightCreated`

This event is emitted when a new piece of freight is created by a warehouse or the API server.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)

### `FreightApproved`

This event is emitted when freight is manually approved for a stage.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)

### `FreightVerificationSucceeded`

This event is emitted when freight verification completes successfully.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `FreightVerificationFailed`

This event is emitted when freight verification completes with a failure.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `FreightVerificationErrored`

This event is emitted when freight verification encounters an unexpected error.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `FreightVerificationAborted`

This event is emitted when freight verification is aborted before completion.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `FreightVerificationInconclusive`

This event is emitted when freight verification finishes with an inconclusive result.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `FreightVerificationUnknown`

This event is emitted when freight verification ends in an unknown state.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [Freight fields](#freight-fields)
- [Freight verification fields](#freight-verification-fields)

### `APITokenCreated`

This event is emitted by the API server when an API token is created. The `project` field is the
namespace holding the token: the Project for a project-level token, or Kargo's own namespace for a
system-level one, which `systemLevel` flags. The `actor` field identifies who created the token.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [API token fields](#api-token-fields)

### `APITokenDeleted`

This event is emitted by the API server when an API token is deleted through the Kargo API. It
carries the same payload as `APITokenCreated`, describing the token as it was before deletion.
Tokens removed by other means, such as `kubectl` or the deletion of their Role, do not emit it.

**Payload Includes**

- [Common event fields](#common-event-fields)
- [API token fields](#api-token-fields)
