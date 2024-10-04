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
for the most up-to-date information, please see the [GitHub
Project](https://github.com/akuity/kargo/milestones)
:::

## In Progress

### v1.0.0

__Expected:__ 18 October 2024

v1.0.0 will be our long-anticipated GA release. There are no major features
planned and the release will focus almost entirely on bug fixes and stability.

__One notable change, however, will be the removal of the legacy (opinionated)
promotion mechanisms that were deprecated in v0.9.0.__

## Upcoming

v1.0.0 is a major milestone for the project and signals that we are confident in
the design and stability of Kargo's _core features_, but we're still just
getting started!

### v1.1.0 and Beyond

v0.9.0's strategic refactor to promotion steps has opened up a world of
possibilities for Kargo.

The general theme for our first several minor releases post-GA will be
_extensibility_. With an end-goal of enabling third-party integrations in the
form of promotion steps, we will be working on:

* Ensuring a secure and isolated execution environment for promotion steps.

* Publishing a formal specification for developers wishing to implement their
  own promotion steps.

* Providing the mechanisms for operators to install and users to leverage
  versioned, third-party promotion steps.

Through this work, we intend to enable a rich ecosystem of promotion steps that will provide a wide range of capabilities including, but not limited to, notifications, approval workflows, alternative GitOps agents, and non-Kubernetes deployments.

Several UX improvements are also planned, including:

* Packaging common workflows as pre-defined, composite promotion steps.

* Implementing an expression language that permits promotion step configuration
  to more easily reference things like Freight, credentials, and the output of
  previous promotion steps.

## Completed

### v0.9.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| [Promotion Steps](https://github.com/akuity/kargo/issues/2219) | strategic refactor | Transitioned from opinionated promotion mechanisms to an ordered lists of more finely-grained promotion directives steps reminiscent of GitHub Actions. These enable greater flexibility in addressing outlying use cases and have left us with a clear path forward for to eventually enable third-party integrations. |
| Production Readiness | chore | <ul><li>Prioritized stability of existing features.</li><li>Paid down technical debt.</li><li>**This does not mean v0.9.0 is production-ready. It means it is several steps closer to it.**</li></ul> |

### v0.8.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Auth via [GitHub Apps](https://docs.github.com/en/apps) | feature | Support GitHub Apps as an authentication option for GitHub repositories. |
| Multiple `Freight` per `Stage` | feature | Permit `Stage`s to host multiple pieces of `Freight` from different `Warehouse`s. Different artifacts, or sets of artifacts, can be promoted through parallel pipelines with different/independent cadence. |
| Production Readiness | chore | <ul><li>Prioritized stability of existing features.</li><li>Paid down technical debt.</li><li>**This does not mean v0.8.0 is production-ready. It means it is several steps closer to it.**</li></ul> |

### v0.7.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Multiple `Warehouse`s | feature | Improved UI support for displaying Freight from multiple `Warehouse`s. |
| Manual `Freight` Creation | feature | Added UI feature for manual `Freight` creation. |
| ECR/GAR Support | feature | Added multiple options for authenticating to image repositories in ECR and Google Artifact Registry, including support for EKS Pod Identity and GKE Workload Identity Federation. | 
| [Patch Promotions](https://github.com/akuity/kargo/issues/1250) | poc | Support a generalized option to promote arbitrary configuration (e.g. strings, files, and directories) to other paths of a GitOps repository. |
| Production Readiness | chore | Prioritized stability of existing features. **This does not mean v0.7.0 is production-ready. It means it is several steps closer to it.** |

### v0.6.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| Project Management | feature | Added user / role / permission management capabilities to the CLI and UI. |
| Events | feature | Kargo emits noteworthy events as Kubernetes events. Events are also viewable in the UI. |
| Production Readiness | chore | Prioritized stability of existing features. **This does not mean v0.6.0 is production-ready. It means it is several steps closer to it.** |

### v0.5.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Warehouse` Rules/Filters | feature | Introduced optional path-based constraints on Git repository subscriptions. |
| Credential Storage | refactor | Simplified and streamlined format and storage of repository credentials. |
| Credential Management | feature | Added credential management capabilities to the CLI and UI. |
| CLI Improvements | refactor | Overhauled the CLI to make the tree of sub-commands more intuitive, with improved consistency in usage and documentation from command to command. |
| UI Improvements | feature | Achieved near-parity with CLI features. |

### v0.4.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Warehouse` Rules/Filters | feature | Introduced optional tag-based constraints on Git repository subscriptions. |
| Project Management | feature | <ul><li>Introduced `Project` CRD to simplify project initialization.</li><li>Removed `PromotionPolicy` CRD and folded its functionality directly into the `Project` CRD.</li></ul> |

### v0.3.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| GitHub PR-Based Promotion | feature | Pull request-based promotions are now supported on GitHub. |
| Verifications | feature | `Stage` resources can now execute a user-defined verification process after a promotion. These can be defined using Argo Rollouts `AnalysisTemplate` resources, and executions take the form of `AnalysisRun` resources. |
| Improved RBAC | feature | SSO user identities can now be mapped to Kubernetes `ServiceAccount` resources using annotations. |

### v0.2.0

| Name | Type | Description |
| ---- | ---- | ----------- |
| `Freight` CRD | feature | Freight changed from being a property of a `Stage`, to being its own `Freight` CRD. |
| `Warehouse` CRD | feature | `Freight` production was decoupled from a pipeline's first `Stage` and now comes from a `Warehouse`. |
| Kargo Render | breaking change | The Bookkeeper project was rebranded as Kargo Render -- a Kargo sub-project for rendering manifests. |

## Criteria for a Production-Ready 1.0.0 Release

Maintainers will consider cutting a stable v1.0.0 release once:

* Confident in API stability. (No further breaking changes anticipated.)
* No critical, "show-stopping" bugs remaining in the backlog.
* Observing evidence of successful community adoption (of beta releases) in production environments
