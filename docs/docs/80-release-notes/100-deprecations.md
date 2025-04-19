---
sidebar_label: Deprecations and Breaking Changes
description: A summary of deprecated features and breaking changes
---

# Deprecations and Breaking Changes

As Kargo continues to evolve, features that have outlived their usefulness,
often having been replaced with better alternatives, are sometimes sunset. This
page documents past and pending deprecated features and breaking changes.

## Deprecations

The table below summarizes features that have been deprecated and either removed or scheduled for removal.

| Feature | Deprecated In | Removed In | Replacement/Notes |
|---------|---------------|------------|-------------------|
| `prNumber` of `git-open-pr` Promotion Steps | v1.5.0 | Scheduled for v1.6.0 | Users should migrate to using `pr.id` for referencing pull request numbers. |
| `messageFromSteps` of `git-commit` Promotion Step | [v1.3.0](./96-v1.3.0.md#new-deprecations) | Scheduled for v1.5.0 | Use the `message` field in combination with expressions. Refer to the [documentation](https://main.docs.kargo.io/user-guide/reference-docs/promotion-steps/git-commit/#composed-commit-message) for more information. |
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
