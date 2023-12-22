---
sidebar_label: Roadmap
Description: See what's on the roadmap of Kargo and find out more about the latest releases
---

# Kargo Roadmap

Over a series of releases, Kargo's maintainers intend to establish and settle into a predictable, but yet to be determined release cadence.

:::caution
This roadmap is subject to change at any time, for the most up to date information, please see the [GitHub Project](https://github.com/akuity/kargo/milestones)
:::

## v0.2.0

__Status:__ Completed

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight` CRD | feature | Freight changed from being a property of a `Stage`, to being its own `Freight` CRD. |
| `Warehouse` CRD | feature | `Freight` production was decoupled from a pipeline's first `Stage` and now comes from a `Warehouse`. |
| Kargo Render | breaking change | The Bookkeeper project was rebranded as Kargo Render -- a Kargo sub-project for rendering manifests. |

## v0.3.0

__Status:__ Completed

| Name | Type | Description |
| ---- | ---- | ----------- |
| GitHub PR-Based Promotion | feature | Pull request-based promotions are now supported on GitHub. |
| Verifications | feature | `Stage` resources can now execute a user-defined verification process after a promotion. These can be defined using Argo Rollouts `AnalysisTemplate` resources, and executions take the form of `AnalysisRun` resources. |
| Improved RBAC | feature | SSO user identities can now be mapped to Kubernetes `ServiceAccount` resources using annotations. |

## v0.4.0

__Status:__ In Progress

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Warehouse` Rules/Filters | feature | Introduce features to constrain the conditions under which new `Freight` is produced. |
| Project Management | feature | <ul><li>Introduce `Project` CRD to simplify onboarding and project lifecycle management.</li><li>Aggregate important status information at the `Project` level.</li><li>Introduce additional `PromotionPolicy` options.</li><li>Credential management via CLI and UI.</li></ul> |

## v0.5.0

__Status:__

| Name | Type | Description |
| ---- | ---- | ----------- |
| Promotion Mechanism Extensibility | feature | User-defined promotion mechanisms. |

## v0.6.0 .. v0.n.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Freight Enrichment | feature | Enhance Freight metadata for improved insight into Freight contents and the expected result of promoting a piece of Freight to a given environment. This data will be exposed to the UI and CLI tools. |
| Improved Microservice Support | feature | Filters for Freightlines (for example, filter by Warehouse). Add the ability to merge parallel Freightlines at a control flow Stages. |
| `kargo init` | feature | Addition of an `init` sub-command to the Kargo CLI for streamlining project / pipeline creation. |
| Standalone Image Writeback` | feature | Write back image changes without having to subscribe to an image repository. |
| PromotionPolicy Improvements | feature | Add the ability to "freeze" Stages to prevent promotions. |

## Criteria for 1.0.0 Release

Maintainers will consider cutting a stable v1.0.0 release once:

* Confident in API stability. (No further breaking changes anticipated.)
* No critical, "show-stopping" bugs remaining in the backlog.
* Observing evidence of successful community adoption (of beta releases) in production environments
