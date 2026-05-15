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
| `git-commit` step `author` field | [v1.10.0](./89-v1.10.0.md) | Scheduled for v1.12.0 | Configure authorship and signing in the `git-clone` step or via `ClusterConfig`. See [git-commit docs](https://docs.kargo.io/user-guide/reference-docs/promotion-steps/git-commit). |
| `git-push` default integration policy (`AlwaysRebase`) | [v1.10.0](./89-v1.10.0.md) | Default changes in v1.12.0 | The default `git-push` push integration policy will change from `AlwaysRebase` to `RebaseOrMerge` in v1.12.0. Set [`controller.gitClient.pushIntegrationPolicy`](https://docs.kargo.io/operator-guide/advanced-installation/common-configurations#push-integration-policy) explicitly if you rely on unconditional rebase. |
| SSH URLs and SSH private keys for Git repositories | v1.10.0 | Scheduled for v1.13.0 | Use HTTPS URLs with a personal access token or equivalent. SSH keys cannot authenticate to git provider APIs, forcing users to maintain two sets of credentials. See [#5858](https://github.com/akuity/kargo/issues/5858) for details. |
| The `createTargetBranch` option in the `git-open-pr` promotion step | [v1.10.0](./89-v1.10.0.md) | Scheduled for v1.12.0 | The `createTargetBranch` option has been deprecated as the feature never worked. See [#5847](https://github.com/akuity/kargo/issues/5847) for details. |
| The Connect-based API | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | A new, RESTful API has been introduced. Most users will not be impacted beyond simply needing to upgrade their CLI when upgrading the back end to v1.9.0 or greater. |
| "global credentials namespace(s)" | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | Replaced with "shared secrets namespace." See [release notes](./90-v1.9.0.md#the-secret-shuffle) and [docs](../40-operator-guide/40-security/40-managing-secrets.md#transitioning) for details. |
| "cluster secrets namespace" | [v1.9.0](./90-v1.9.0.md) | Scheduled for v1.12.0 | Replaced with "system resources namespace." See [release notes](./90-v1.9.0.md#the-secret-shuffle) and [docs](../40-operator-guide/40-security/40-managing-secrets.md#transitioning) for details. |
| `Warehouse`'s container image subscription's `semverConstraint` field | [v1.7.0](./92-v1.7.0.md#new-deprecations) | [v1.9.0](./90-v1.9.0.md) | Users should migrate to using the [`constraint`](https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses/#image-selection-strategies) field which, accepts the same value but is named to better indicate it can also be used for tag selection (e.g. `latest`) when the image selection strategy is set to `Digest`. |
| `freightMetadata` functions optional second argument for the key name | [v1.8.0](./91-v1.8.0.md#new-deprecations) | [v1.10.0](./89-v1.10.0.md#breaking-changes) | Users should migrate to either dot notation (`freightMetadata(freightName).keyName`) or map access syntax (`freightMetadata(freightName)['key-name']`) to access specific values instead of using the optional second argument. |
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
