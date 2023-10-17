---
title: Roadmap
Description: Kargo Roadmap
---

# Roadmap

Kargo does not currently have a regular release cadance. The plan is to do a few releases, and based on the resluts, create a more formal release cadance.

:::caution
This roadmap is subject to change at any time, for the most up to date information, please see the [GitHub Project](<link goes here>)
:::

# v0.2.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight/Warehouse CRD` | feature | Freight will change from being a property of a `Stage`, to being it's own CRD. A collection of promotable `Freight` will be known as a `Warehouse`. |
| `Long Lived Promotions` | feature | Support for Pull Requset based promotion, which is indefinite. This will allow the promotion to last as long as the PR is open. |
| `Kargo Render` | breaking change | `Bookkeeper` to be retooled to be the Kargo specific way to do rendered manifests. |


# v0.3.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Analysis` | feature | Run analysis against a metrics provider to determite application health. This will allow assessing Promotion success based on more than just Argo CD Application state. |
| `Kargo RBAC` | feature | Map SSO user identities to Kubernetes ServiceAccounts. Predefined ServiceAccount/Role/RoleBinding per project based on persona. In this version, it'll mostly be backend work. |
| `Freight Production Rules/Filters` | feature | Optionally set up tag-based Git repository subscriptions. Additional options to produce new Freight only under certain conditions. |

# v0.4.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Project management` | feature | User/group/permissions management via UI and CL. Project CRD + reconciler will relieve API server of its most abusable permissions; reducing the risk profile. Additional `PromotionPolicy` options. Credential management via CLI and UI. |

# v0.5.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Promotion Extensibility` | feature | User defined promotion mechanisms. |

# v0.6.0 - v1.0.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight Enrichment` | feature | Include metadata about freight to help users understand what they’re looking at and decide what they promote. This data will be exposed to the UI and CLI tools. |
| `Improved microservice support` | feature | Filters on Freightlines (for example by `Warehouse`). Add the ability to merge parallel Freightlines at a control flow `Stage` |
| `kargo init` | feature | Add a `kargo init` subcommand. This subcommand will: Auto-generate namespace, stages, and git repo layout |
| `Stand-alone image writeback` | feature | Write back image changes without having to subscribe to an image repository. |
| `PromotionPolicy improvements` | feature | Add the ability to "freeze" deployments (bascially locks down a `Stage` from being promoted into). |

# Criteria for 1.0.0 Release

The criteria for a stable v1.0.0 release will be considered when we feel confident in API stability (no breaking changes) and that there are no show-stopping/critial bugs. We will also be looking for a community of users who are successfully using Kargo in production.