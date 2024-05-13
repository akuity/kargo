---
sidebar_label: Roadmap
Description: See what's on the roadmap of Kargo and find out more about the latest releases
---

# Kargo Roadmap

Over a series of releases, Kargo's maintainers have settled into a cadence of a
minor release roughly every five weeks, with two or three major features
completed per release.

:::caution
This roadmap tracks only _major_ features and is subject to change at any time,
for the most up to date information, please see the [GitHub
Project](https://github.com/akuity/kargo/milestones)
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

__Status:__ Completed

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Warehouse` Rules/Filters | feature | Introduced optional path-based constraints on Git repository subscriptions. |
| Credential Storage | refactor | Simplified and streamlined format and storage of repository credentials. |
| Credential Management | feature | Added credential management capabilities to the CLI and UI. |
| CLI Improvements | refactor | Overhauled the CLI to make the tree of sub-commands more intuitive, with improved consistency in usage and documentation from command to command. |
| UI Improvements | feature | Achieved near-parity with CLI features. |

## v0.6.0

__Status:__ Completed

| Name | Type | Description |
| ---- | ---- | ----------- |
| Project Management | feature | Added user / role / permission management capabilities to the CLI and UI. |
| Events | feature | Kargo emits noteworthy events as Kubernetes events. Events are also viewable in the UI. |
| Production Readiness | chore | Prioritized stability of existing features. **This does not mean v0.6.0 is production-ready. It means it is several steps closer to it.** |

## v0.7.0

__Status:__ In Progress
__Expected:__ 2024-06-07

| Name | Type | Description |
| ---- | ---- | ----------- |
| Multiple `Warehouse`s | feature | Improve UI support for multiple Freightlines rooted in different `Warehouse`s. |
| Manual `Freight` Creation | feature | Add CLI and UI support for manual `Freight` creation. This will give users the flexibility to create novel combinations of artifacts that `Warehouse`s will not -- for instance, pairing the most recent version of a container image with an _older_ version of application manifests. |
| ECR/GitHub Auth | feature | Native authentication support for ECR registries and using GitHub applications | 
| [Patch Promotions](https://github.com/akuity/kargo/issues/1250) | poc | Support a generalized option to promote arbitrary configuration (e.g. strings, files, and directories) to other paths of a GitOps repository. |
| Production Readiness | chore | <ul><li>Prioritize stability of existing features.</li><li>Pay down technical debt.</li><li>**This is not a guarantee that v0.7.0 will be production-ready. It is a commitment to large steps in that direction.**</li></ul> |

## v0.8.0 .. v0.n.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Promotion Mechanism Extensibility | feature | User-defined promotion mechanisms. |
| `Project` Improvements | feature | <ul><li>Permit promotion policies to "freeze" `Freight` production and/or promotions based on time or other constraints.</li><li>Aggregate useful project-level status and statistics in `ProjectStatus`.</li></ul> |
| `Freight` Enrichment | feature | Enhance `Freight` metadata for improved insight into contents and the expected result of promoting a piece of `Freight` to a given `Stage`. |
| `kargo init` | feature | Addition of an `init` sub-command to the Kargo CLI for streamlining project / pipeline creation. |
| Standalone Image Writeback | feature | Write back image changes without having to subscribe to an image repository. |

## Criteria for a Production-Ready 1.0.0 Release

Maintainers will consider cutting a stable v1.0.0 release once:

* Confident in API stability. (No further breaking changes anticipated.)
* No critical, "show-stopping" bugs remaining in the backlog.
* Observing evidence of successful community adoption (of beta releases) in production environments
