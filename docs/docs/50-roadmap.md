---
title: Roadmap
Description: Kargo Roadmap
---

# Roadmap

Over a series of releases, Kargo's maintainers intend to establish and settle into a predictable, but yet to be determined release cadence.

:::caution
This roadmap is subject to change at any time, for the most up to date information, please see the [GitHub Project](https://github.com/akuity/kargo/milestones)
:::

## v0.2.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Freight and Warehouse CRDs | feature | Freight will change from being a property of a `Stage`, to being its own CRD. The source of where promotable `Freight` comes from will be known as a `Warehouse`. |
| GitHub PR-Based Promotion | feature | Support for pull request based promotions, which do not complete until the underlying PR is closed. |
| Kargo Render | breaking change | Bookkeeper to be rebranded as Kargo Render -- a Kargo child project for rendering manifests. |


## v0.3.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Analysis | feature | Ability to execute user-defined analysis steps to qualify or disqualify Freight for further promotion. |
| Improved RBAC | feature | Map SSO user identities to Kubernetes ServiceAccounts. Predefined ServiceAccount/Role/RoleBinding per project based on persona. |
| Freight Production Rules/Filters | feature | Git repository subscriptions options to constrain conditions under which new Freight is produced. |

## v0.4.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Project Management | feature | Introduce Project CRD to simplify onboarding and project lifecycle management. Support aggregate information at the project status level. Additional `PromotionPolicy` options. Credential management via CLI and UI. |

## v0.5.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Promotion Mechanism Extensibility | feature | User-defined promotion mechanisms. |

## v0.6.0 .. v0.n.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Freight Enrichment | feature | Enhance Freight metadata for improved insight into Freight contents and the expected result of promoting a piece of Freight to a given environment. This data will be exposed to the UI and CLI tools. |
| `Improved Microservice Support | feature | Filters for Freightlines (for example, filter by Warehouse). Add the ability to merge parallel Freightlines at a control flow Stages. |
| `kargo init` | feature | Addition of an `init` sub-command to the Kargo CLI for streamlining project / pipeline creation. |
| Standalone Image Writeback` | feature | Write back image changes without having to subscribe to an image repository. |
| PromotionPolicy Improvements | feature | Add the ability to "freeze" Stages to prevent promotions. |

## Criteria for 1.0.0 Release

Maintainers will consider cutting a stable v1.0.0 release once:

* Confident in API stability. (No further breaking changes anticipated.)
* No critical, "show-stopping" bugs remaining in the backlog.
* Observing evidence of successful community adoption (of beta releases) in production environments
