---
sidebar_label: Annotations and Labels
description: List of Kargo-specific annotations and labels
---

# Annotations and Labels used by Kargo

This page documents certain annotations and labels that are applicable to
Kargo resource types and other Kubernetes resource types (e.g. `Secret`s or
`ServiceAccounts`) and affect how Kargo handles those resources. The subset
documented here are those that the maintainers have judged most likely to be of
use to advanced users who may, for instance, be looking to interact
programmatically with Kargo.

## Annotations

| Key | Target Resource(s) | Possible Values | Description |
|-----|--------------------|-----------------|-------------|
| `kargo.akuity.io/abort` | `Stage` | A plain string (verification ID from `.status.verifications[*].id` of the `Stage`). | Aborts an in-progress `Freight` verification. |
| `kargo.akuity.io/authorized-stage` | `Argo CD Application` | `<project>:<stage>` | Indicates which `Stage` is authorized to manage the `Application` resource. |
| `kargo.akuity.io/color` | `Stage` | Hex color code (e.g. `#ff8800`) | Optional cosmetic color used in the UI's pipeline view. |
| `kargo.akuity.io/description` | Any | Any string | Optional human-readable description of the resource. May be used by the Kargo UI to display additional context or details. |
| `kargo.akuity.io/refresh` | `Warehouse`, `Stage`, `Promotion` | A string that is unique or at least unlikely to repeat, such as a UUID or a timestamp of "now" | Triggers reconciliation of the resource when its value changes. |
| `kargo.akuity.io/reverify` | `Stage` | Either a plain string (verification ID from `.status.verifications[*].id` of the `Stage`) or a JSON object with `id` (required), `actor`, and `controlPlane` fields. If a JSON object is provided, it is parsed as a `VerificationRequest`. | Triggers re-verification of a previously completed verification for the current `Freight`. |
| `rbac.kargo.akuity.io/claim.<name>` | `ServiceAccount` | Any valid OIDC claim value (e.g., `sub`, `email`, or `groups`) | Maps an OIDC claim to a `ServiceAccount`, enabling user-to-ServiceAccount mappings. For more details, refer to the access control sections of the [Operator Guide](../../40-operator-guide/40-security/30-access-controls.md) and [User Guide](../50-security/20-access-controls/index.md). |
| `rbac.kargo.akuity.io/managed` | `ServiceAccount`, `Role`, `RoleBinding` | `"true"` | Permits the UI or CLI (via the API server) to programmatically manage trios of `ServiceAccount`, `Role`, and `RoleBinding` resources via Kargo's own ["roles" abstraction](../50-security/20-access-controls/index.md#managing-mappings-and-permissions). Omit this annotation if you wish to exclusively manage these resources [declaratively](../50-security/20-access-controls/index.md#managing-kargo-roles-declaratively). |

## Labels

| Key | Target Resource(s) | Possible Values | Description |
|-----|--------------------|-----------------|-------------|
| `kargo.akuity.io/alias` | `Freight` | Any string that is unique within the project | Mutable, human-readable alias for a piece of `Freight`. This label is automatically synced from the resource's `alias` field. Users are discouraged from modifying the label directly.  The label exists primarily to enable querying for `Freight` by alias using `kubectl`. |
| `kargo.akuity.io/cred-type` | `Secret` | `git`, `helm`, `image`, `generic` | Indicates a `Secret` represents credentials for a repository of the specified type. For more details, see the [Managing Credentials](../50-security/30-managing-credentials.md#repository-credentials-as-secret-resources). |
| `kargo.akuity.io/project` | `Namespace` | `"true"` | Indicates that the `Namespace` is eligible for adoption by a `Project` with the same name. This label is useful when `Namespace`s are unavoidably pre-created by some other agent. For more details, see the [Working with Projects](../20-how-to-guides/20-working-with-projects.md#namespace-adoption) section. |
| `kargo.akuity.io/shard` | `Promotion`, `Stage`, `Warehouse` | Shard ID | Indicates a specific controller instance responsible for reconciling the resource. For `Warehouse` and `Stage` resources, this label is automatically synced from the resource's `spec.shard` field. Users are discouraged from modifying the label directly. The label exists primarily to enable querying for resources by shard using `kubectl`. |
