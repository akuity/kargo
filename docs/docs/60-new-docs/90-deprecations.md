---
sidebar_label: Deprecations and Breaking Changes
description: Overview of deprecated features, their removal timeline, reasons for deprecation, and suggested replacements.
---

# Deprecation Notices

As Kargo continues to evolve, features and functionalities are periodically updated,
improved, or replaced to enhance the platform's capabilities.
When these changes occur, older features are deprecated and scheduled for removal.

This page provides a list of features that have been deprecated, along with
their scheduled removal dates and suggested replacements, if any.

## Removed items by Release

### [v1.1.0](./80-release-notes/98-v1.1.0.md#-new-and-updated-promotion-steps)

#### Deprecated Promotion Steps Fields

Several fields in promotion steps, such as `prNumberFromStep` in the
`git-wait-for-pr` step, are now deprecated. These fields were initially used
to reference outputs from previous steps directly.
With the introduction of expressions, these fields have become redundant,
as expressions like `${{ outputs['open-pr'].prNumber }}` now offer a more
flexible and straightforward way to achieve the same functionality.

These deprecated fields are scheduled for removal in **v1.3.0**.

#### Deprecated Step: `helm-update-image`

The `helm-update-image` step is deprecated in favor of the more flexible `yaml-update` step,
which works seamlessly with expressions.
This provides a broader and more versatile approach.

This step is scheduled for removal in **v1.3.0**.

Please refer to the [promotion steps reference documentation](./60-user-guide/60-reference-docs/30-promotion-steps/10-git-clone.md) for
detailed information about the deprecated promotion steps and fields.

### [v1.0.0](./80-release-notes/99-v1.0.0.md#%EF%B8%8F-breaking-changes)

No deprecated features in this release. The focus was on stability and completing the transition to flexible promotion steps started in v0.9.0.

## What Next?

For detailed information on updates and deprecated features, please refer to the respective [release notes](https://github.com/akuity/kargo/releases/).
