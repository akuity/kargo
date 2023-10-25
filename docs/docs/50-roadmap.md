---
title: Roadmap
Description: Kargo Roadmap
---

# Roadmap

Kargo does not currently have a regular release cadence. The plan is to do a few releases, and based on the results, create a more formal release cadence.

:::caution
This roadmap is subject to change at any time, for the most up to date information, please see the [GitHub Project](https://github.com/akuity/kargo/milestones)
:::

# v0.2.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight/Warehouse CRD` | feature | Freight will change from being a property of a `Stage`, to being it's own CRD. The source of where promotable `Freight` comes from will be known as a `Warehouse`. |
| `GitHub PR-based Promotion` | feature | Support for Pull Requset based promotion, which is indefinite. This will allow the promotion to last as long as the PR is open. |
| `Kargo Render` | breaking change | `Bookkeeper` to be rebranded as `kargo-render`, a Kargo sub-component for rendering manifests. |


# v0.3.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Analysis` | feature | Ability to run Analysis for the purposes of Freight qualification. This enables users to define additional quality gates (in addition to Argo CD Health) to allow/disallow promotion. |
| `Kargo RBAC` | feature | Map SSO user identities to Kubernetes ServiceAccounts. Predefined ServiceAccount/Role/RoleBinding per project based on persona. |
| `Freight Production Rules/Filters` | feature | Optionally set up tag-based Git repository subscriptions. Additional options to produce new Freight only under certain conditions. |

# v0.4.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Project management` | feature | Introduce Project CRD to simplify onboarding and project lifecycle management. Support aggregate information at the project status level.. Additional `PromotionPolicy` options. Credential management via CLI and UI. |

# v0.5.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Promotion Extensibility` | feature | User defined promotion mechanisms. |

# v0.6.0 .. v0.n.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight Enrichment` | feature | Include metadata about freight to help users understand what they’re looking at and decide what they promote. This data will be exposed to the UI and CLI tools. |
| `Improved microservice support` | feature | Filters on Freightlines (for example by `Warehouse`). Add the ability to merge parallel Freightlines at a control flow `Stage` |
| `kargo init` | feature | Add a `kargo init` subcommand. This subcommand will: Auto-generate namespace, stages, and git repo layout |
| `Stand-alone image writeback` | feature | Write back image changes without having to subscribe to an image repository. |
| `PromotionPolicy improvements` | feature | Add the ability to "freeze" deployments (bascially locks down a `Stage` from being promoted into). |

# Criteria for 1.0.0 Release

The criteria for a stable v1.0.0 release will be considered when we feel confident in API stability (no breaking changes) and that there are no show-stopping/critial bugs. We will also be looking for a community of users who are successfully using Kargo in production.
