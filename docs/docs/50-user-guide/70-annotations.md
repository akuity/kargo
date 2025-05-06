---
sidebar_label: Annotations
description: List of Kargo-specific annotations and labels
---

# Annotations and Labels used by Kargo

This page documents Kargo-specific annotations, labels, and finalizers. Some of these fields can be set by users, while others are system-managed and used internally by the control plane.

## Annotations

| Annotation key | Target resource(s) | Possible values | Description |
|----------------|--------------------|------------------|-------------|
| `kargo.akuity.io/refresh` | Any Kargo resource | Arbitrary string (acts as a token) | Triggers reconciliation when the value changes. Useful for forcing a resync. |
| `kargo.akuity.io/reverify` | `Stage` | Verification ID or JSON object (VerificationRequest) | Triggers re-verification of the currently assigned Freight. |
| `kargo.akuity.io/abort` | `Stage` | Verification ID | Aborts an in-progress Freight verification. |
| `kargo.akuity.io/description` | Any | Free-form string | Human-readable description displayed in the UI. |
| `kargo.akuity.io/color` | `Stage` | Hex color code (e.g. `#ff8800`) | Optional cosmetic color used in UI. |
| `kargo.akuity.io/create-actor` | All | Actor identity (username/service) | Injected by the control plane to indicate who created the resource. |
| `kargo.akuity.io/authorized-stage` | `Freight`, `Promotion`, etc. | `<project>:<stage>` | Indicates which Stage is authorized to manage the resource. |
| `kargo.akuity.io/promotion` | `PromotionTask`, internal resources | Promotion ID or name | Links the resources to a promotion process. |
| `kargo.akuity.io/argocd-context` | `Stage` | JSON or internal identifier | Records Argo CD application context from the last promotion. |

:::warning
Avoid setting system-managed annotations manually unless you're explicitly debugging internal behavior.
:::

### RBAC Annotations

| Annotation key | Target resource(s) | Possible values | Description |
|----------------|--------------------|------------------|-------------|
| `rbac.kargo.akuity.io/managed` | RBAC resources | `"true"` | Marks a resource as managed by Kargo's RBAC system. |
| `rbac.kargo.akuity.io/claim.<name>` | RBAC resources | Claim name as suffix | Maps an OIDC claim to a policy. |

### Event Annotations

The following annotations are added internally by Kargo's eventing subsystem for tracking and audit. They are **not user-settable**.

| Annotation key | Target resource(s) | Possible values | Description |
|----------------|--------------------|------------------|-------------|
| `event.kargo.akuity.io/actor` | Events | Actor string | Actor that triggered the event. |
| `event.kargo.akuity.io/project` | Events | Project name | Associated project. |
| `event.kargo.akuity.io/promotion-name` | Events | Promotion name | Related promotion. |
| `event.kargo.akuity.io/promotion-create-time` | Events | Timestamp | Creation time of the promotion. |
| `event.kargo.akuity.io/freight-*` | Events | Varies | Additional metadata about Freight. |
| `event.kargo.akuity.io/stage-name` | Events | Stage name | Target Stage. |
| `event.kargo.akuity.io/analysis-run-name` | Events | Name | Linked Argo CD `AnalysisRun`. |
| `event.kargo.akuity.io/verification-*` | Events | Timestamps/status | Tracks verification lifecycle. |
| `event.kargo.akuity.io/applications` | Events | Comma-separated names | Applications involved in promotion. |

## Labels

| Label key | Target resource(s) | Possible values | Description |
|-----------|--------------------|------------------|-------------|
| `kargo.akuity.io/alias` | `Freight` | User-defined string | Friendly identifier for the Freight version. |
| `kargo.akuity.io/stage` | `Freight`, `Promotion`, etc. | Stage name | Indicates the associated Stage. |
| `kargo.akuity.io/project` | All | Project name | Denotes Project ownership. |
| `kargo.akuity.io/freight-collection` | `Freight` | Collection name | Source collection from which Freight originated. |
| `kargo.akuity.io/cred-type` | Credentials | Credential type (e.g., `ssh`, `https`) | Internal marker for credential classification. |
| `kargo.akuity.io/shard` | Internal resources | Shard ID | Logical cluster or controller shard ID. |

## Finalizers

| Key | Target resource(s) | Possible values | Description |
|-----|--------------------|------------------|-------------|
| `kargo.akuity.io/finalizer` | All | N/A | Used by control plane to enforce proper cleanup. |


:::info
- If you're setting annotations via `kubectl`, use the `--overwrite` flag when updating existing annotations: 
  ```
  kubectl annotate stage my-stage kargo.akuity.io/refresh=$(date +%s) --overwrite
  ```
:::
