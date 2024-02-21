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

__Status:__ Completed

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Warehouse` Rules/Filters | feature | Introduced optional tag-based constraints on Git repository subscriptions. |
| Project Management | feature | <ul><li>Introduced `Project` CRD to simplify project initialization.</li><li>Removed `PromotionPolicy` CRD and folded its functionality directly into the `Project` CRD.</li></ul> |

## v0.5.0

__Status:__ In Progress

| Name | Type | Description |
| ---- | ---- | ----------- |
| Project Management | feature | Add sensible `ServiceAccount`s, `Role`s, and `RoleBinding`s to boilerplate project setup. |
| `Warehouse` Rules/Filters | feature | Introduce optional path-based constraints on Git repository subscriptions. |
| UI Improvements | feature | <ul><li>Enabled credential management via UI.</li><li> UI features are lagging behind back end advancements. This release will have a strong focus on getting caught up. </li></ul> |
| CLI Improvements | refactor | The CLI will receive a near-total overhaul to make the tree of sub-commands more intuitive, with greater consistency in documentation and usage from command to command. |
| Promotion Mechanism Extensibility | design/proposal | User-defined promotion mechanisms. |
| [Patch Promotions](https://github.com/akuity/kargo/issues/1250) | poc | Support a generalized option to promote arbitrary configuration (e.g. strings, files, and directories) to other paths of the Git repository. |

## v0.6.0

__Status:__

| Name | Type | Description |
| ---- | ---- | ----------- |
| Promotion Mechanism Extensibility | feature | User-defined promotion mechanisms. |

## v0.7.0 .. v0.n.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Project` Improvements | feature | <ul><li>Permit promotion policies to "freeze" `Freight` production and/or promotions based on time or other constraints.</li><li>Aggregate useful project-level status and statistics in `ProjectStatus`.</li></ul> |
| `Freight` Enrichment | feature | Enhance `Freight` metadata for improved insight into contents and the expected result of promoting a piece of `Freight` to a given `Stage`. |
| Improved Microservice Support | feature | Filters for Freightlines (for example, filter by `Warehouse`). Add the ability to merge parallel pipelines at a "junction" `Stage`. |
| `kargo init` | feature | Addition of an `init` sub-command to the Kargo CLI for streamlining project / pipeline creation. |
| Standalone Image Writeback | feature | Write back image changes without having to subscribe to an image repository. |

## Criteria for 1.0.0 Release

Maintainers will consider cutting a stable v1.0.0 release once:

* Confident in API stability. (No further breaking changes anticipated.)
* No critical, "show-stopping" bugs remaining in the backlog.
* Observing evidence of successful community adoption (of beta releases) in production environments
