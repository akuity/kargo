---
description: Reference documentation for Kargo events and types.
---

# Kargo Event Reference

This document contains a complete reference of all Kargo events, including their types and
descriptions.

:::info

These events are always emitted by Kargo to the Kubernetes event log with the keys encoded in the
annotations of the event. However, they are primarily consumed by the Pro version of Kargo (such as
the [Notification feature](./100-notifications/index.md))

:::

## Event Fields

Many Kargo events share common fields. The following sections describe these common fields, which
are referenced in individual event definitions. Each field is given as it appears in the event
payload (serialized as JSON) and, if optional, notes whether the type is a pointer when used in
expr-lang expressions.

In each event definition in [Event Types](#event-types), the included fields are listed under
"Payload Includes" with references to the relevant sections. All of the fields described in those
events are found at the top level of the event payload unless otherwise noted.

### Common Event Fields

These fields are included in all Kargo events:

| Field Name | Type   | Description                                    | Optional          |
| ---------- | ------ | ---------------------------------------------- | ----------------- |
| `project`  | String | The project name the event originated from.    | No                |
| `actor`    | String | The user or system that triggered the event.   | Yes (is pointer)  |
| `message`  | String | A human-readable message describing the event. | No (may be empty) |
| `id`       | String | A unique identifier for the event.             | No                |

### Freight Fields

Freight payloads describe the collection of artifacts under evaluation or promotion.

| Field Name   | Type                       | Description                                                                                                         | Optional         |
| ------------ | -------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `name`       | String                     | Name of the freight object.                                                                                         | No               |
| `stageName`  | String                     | Stage associated with the freight when the event fired.                                                             | No               |
| `createTime` | String (RFC3339)           | Creation timestamp of the freight object.                                                                           | No               |
| `alias`      | String                     | Human-friendly alias assigned to the freight.                                                                       | Yes (is pointer) |
| `commits`    | Array\<GitCommit\>         | Git commits that compose the freight (see [GitCommit fields](#gitcommit-fields)).                                   | Yes              |
| `images`     | Array\<Image\>             | Container images included in the freight (see [Image fields](#image-fields)).                                       | Yes              |
| `charts`     | Array\<Chart\>             | Helm charts included in the freight (see [Chart fields](#chart-fields)).                                            | Yes              |
| `artifacts`  | Array\<ArtifactReference\> | Additional arbitrary artifacts included in the freight (see [ArtifactReference fields](#artifactreference-fields)). | Yes              |

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

### Promotion Fields

Promotion payloads describe a promotion resource and the freight it targets.

| Field Name     | Type                                                  | Description                                           | Optional         |
| -------------- | ----------------------------------------------------- | ----------------------------------------------------- | ---------------- |
| `freight`      | Object ([Freight fields](#freight-fields))            | Snapshot of the freight referenced by the promotion.  | Yes (is pointer) |
| `name`         | String                                                | Name of the promotion resource.                       | No               |
| `stageName`    | String                                                | Stage targeted by the promotion.                      | No               |
| `createTime`   | String (RFC3339)                                      | Creation timestamp of the promotion resource.         | No               |
| `applications` | Array\<[NamespacedName](#applications-entry-fields)\> | Argo CD applications resolved for the promotion step. | Yes              |

#### Applications Entry Fields

Each promotion `applications` entry is a Kubernetes `NamespacedName` tuple.

| Field Name  | Type   | Description                              | Optional |
| ----------- | ------ | ---------------------------------------- | -------- |
| `namespace` | String | Namespace that contains the Argo CD app. | No       |
| `name`      | String | Name of the Argo CD application.         | No       |

## Event Types

The complete list of built-in Kargo event types is provided below:

- `PromotionCreated`
- `PromotionSucceeded`
- `PromotionFailed`
- `PromotionErrored`
- `PromotionAborted`
- `FreightApproved`
- `FreightVerificationSucceeded`
- `FreightVerificationFailed`
- `FreightVerificationErrored`
- `FreightVerificationAborted`
- `FreightVerificationInconclusive`
- `FreightVerificationUnknown`

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
