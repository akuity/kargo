---
sidebar_label: Deprecations and Breaking Changes
description: A summary of deprecated features and breaking changes
---

# Deprecations and Breaking Changes

As Kargo continues to evolve, features that have outlived their usefulness,
having often been replaced with better alternatives, are sometimes sunset. This
page documents past and pending deprecated features and breaking changes.

## Deprecations

The table below summarizes features that have been deprecated and either removed
or scheduled for removal.

| Feature | Deprecated In | Removed In | Replacement/Notes |
|---------|---------------|------------|-------------------|
| The Connect-based API | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | A new, RESTful API has been introduced. Most users will not be impacted beyond simply needing to upgrade their CLI when upgrading the back end to v1.9.0 or greater. |
| "global credentials namespace(s)" | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | Replaced with "shared secrets namespace." See [release notes](./90-v1.9.0.md#-the-secret-shuffle) and [docs](../40-operator-guide/40-security/40-managing-secrets.md#transitioning) for details. |
| "cluster secrets namespace" | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | Replaced with "system resources namespace." See [release notes](./90-v1.9.0.md#-the-secret-shuffle) and [docs](../40-operator-guide/40-security/40-managing-secrets.md#transitioning) for details. |
| `Warehouse`'s container image subscription's `semverConstraint` field | [v1.7.0](./92-v1.7.0.md#new-deprecations) | [v1.9.0](./90-v1.9.0.md) | Users should migrate to using the [`constraint`](https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses/#image-selection-strategies) field which, accepts the same value but is named to better indicate it can also be used for tag selection (e.g. `latest`) when the image selection strategy is set to `Digest`. |
| `Project` specification | [v1.5.0](./94-v1.5.0.md#new-deprecations) | [v1.7.0](./92-v1.7.0.md#breaking-changes) | Users should migrate to the dedicated [`ProjectConfig` resource](../50-user-guide/20-how-to-guides/20-working-with-projects.md#project-configuration). This resource kind accepts a `.spec` identitical to the `Project`, but allows for fine-grain permissions. |
| `secrets` object in the Promotion variables | [v1.5.0](./94-v1.5.0.md#new-deprecations) | [v1.7.0](./92-v1.7.0.md#breaking-changes) | Users should migrate to the [`secret()` function](../50-user-guide/60-reference-docs/40-expressions.md#promotion-variables) which resolves Secrets on-demand, reducing overhead. |
| `prNumber` field in `git-open-pr` Promotion Step output | [v1.5.0](./94-v1.5.0.md#new-deprecations) | [v1.7.0](./92-v1.7.0.md#breaking-changes) | Users should migrate to using [the `pr.id`](../50-user-guide/60-reference-docs/30-promotion-steps/git-open-pr.md#output) for referencing pull request IDs. |
| `messageFromSteps` of `git-commit` Promotion Step | [v1.3.0](./96-v1.3.0.md#new-deprecations) | [v1.5.0](./94-v1.5.0.md) | Use the `message` field in combination with expressions. Refer to the [documentation](https://main.docs.kargo.io/user-guide/reference-docs/promotion-steps/git-commit/#composed-commit-message) for more information. |
| Promotion Steps Fields | [v1.1.0](./98-v1.1.0.md#new-and-updated-promotion-steps) | [v1.3.0](./96-v1.3.0.md#breaking-changes) | Several fields in promotion steps, such as `prNumberFromStep` in the `git-wait-for-pr` step, are now deprecated. These fields were originally the only way to reference output from previous promotion steps. With the introduction of expressions, these fields have outlived their purpose, as expressions like `${{ outputs['open-pr'].pr.id }}` present a more flexible and straightforward way to reference the same output. [more info](./98-v1.1.0.md#new-and-updated-promotion-steps) |
| `helm-update-image` step | [v1.1.0](./98-v1.1.0.md#new-and-updated-promotion-steps) | [v1.3.0](./96-v1.3.0.md#breaking-changes) | Use the more flexible `yaml-update` step. [more info](./98-v1.1.0.md#new-and-updated-promotion-steps) |
| Legacy Promotion Mechanisms | v0.9.0 | [v1.0.0](./99-v1.0.0.md#breaking-changes) | Migrate to promotion steps. [more info](./99-v1.0.0.md#breaking-changes) |

## Breaking Changes

This section summarizes significant changes that create potential for disruption
upon upgrade. __The Kargo team strives to keep changes of this nature to an
absolute minimum.__

| Change | Version | Impact | Migration Path |
|--------|---------|--------|----------------|
| Global Credential Store Changes | [v1.0.0](./99-v1.0.0.md#breaking-changes) | Affects installations using `controller.globalCredentials.namespaces` | Manually create a `RoleBinding`s to permit controller access to "global" credential namespaces _or_ set `controller.serviceAccount.clusterWideSecretReadingEnabled` to `true` at install time (not recommended). [more info](./99-v1.0.0.md#breaking-changes) |
