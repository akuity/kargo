---
sidebar_label: Deprecations and Breaking Changes
description: Overview of breaking changes and deprecated features, their removal timeline, reasons for deprecation, and suggested replacements.
---

# Deprecations and Breaking Changes

As Kargo continues to evolve, features and functionalities are periodically updated,
improved, or replaced to enhance the platform's capabilities. This page documents both
deprecated features (scheduled for future removal) and breaking changes that have occurred
across releases.

## Deprecations

The table below outlines features that are currently deprecated and have either been removed or are scheduled for removal.

| Feature | Deprecated In | Removed In | Replacement/Notes |
|---------|---------------|------------|-------------------|
| Promotion Steps Fields | [v1.1.0](./80-release-notes/98-v1.1.0.md#-new-and-updated-promotion-steps) | Scheduled for v1.3.0 | Several fields in promotion steps, such as `prNumberFromStep` in the `git-wait-for-pr` step, are now deprecated. These fields were initially used to reference outputs from previous steps directly. With the introduction of expressions, these fields have become redundant, as expressions like `${{ outputs['open-pr'].prNumber }}` now offer a more flexible and straightforward way to achieve the same functionality. |
| `helm-update-image` step | [v1.1.0](./80-release-notes/98-v1.1.0.md#-new-and-updated-promotion-steps) | Scheduled for v1.3.0 | Use the more flexible `yaml-update` step. |
| Legacy Promotion Mechanisms | v0.9.0 | [v1.0.0](./80-release-notes/99-v1.0.0.md#%EF%B8%8F-breaking-changes) | Migrate to new promotion step system |

## Breaking Changes

This section highlights significant changes that could disrupt existing implementations.
It's essential to review these changes to ensure your configurations remain functional and up to date.

| Change | Version | Impact | Migration Path |
|--------|----------|---------|---------------|
| Global Credential Store Changes | [v1.0.0](./80-release-notes/99-v1.0.0.md#%EF%B8%8F-breaking-changes) | Affects installations using `controller.globalCredentials.namespaces` | Either: 1) Provide custom RoleBindings for controller access, or 2) Enable `controller.serviceAccount.clusterWideSecretReadingEnabled` (not recommended) |

## What Next?

For detailed information on updates and changes, refer to the [release notes](https://github.com/akuity/kargo/releases).
