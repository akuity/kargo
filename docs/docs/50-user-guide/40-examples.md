---
sidebar_label: Examples
---

# Examples

## Basic Examples

A catalog of small Kargo examples can be found in a unified
[kargo-examples](https://github.com/akuity/kargo-examples/) repository.
This collection of micro-examples demonstrate specific functionality in
isolation.

## End-to-End Examples

A number of end-to-end examples that can be used as a starting point, or
inspiration for your own pipelines.

| Name | Description |
|------|-------------|
| [kargo-simple](https://github.com/akuity/kargo-simple) | A simple, 3-stage pipeline with environments configured by Kustomize. Images are updated to kustomize overlays using the [Image Updater](30-patterns/index.md#image-updater) pattern. |
| [kargo-helm](https://github.com/jessesuen/kargo-helm) | Helm example that updates images to environment specific values.yaml. Demonstrates the ability to promote feature flags using file copies to different environment directories. Utilizes the [Multiple Warehouses](30-patterns/index.md#multiple-warehouses) pattern to monitor both a container image repository as well as updates to files in git (feature flags), and the ability to promote image updates independently from features. |
| [kargo-microservices](https://github.com/jessesuen/kargo-microservices) | Example which shows a Warehouse that monitors multiple microservices such that the corresponding Freight can be promoted as a single unit. Demonstrates the [Grouped Services](30-patterns/index.md#grouped-services) pattern with a single Warehouse monitoring multiple image repositories. |
| [kargo-advanced](https://github.com/akuity/kargo-advanced) | An advanced example with a complex pipeline with A/B testing stages and multiple prod regions. Showcases [Verification](20-how-to-guides/16-verification.md) testing using [AnalysisTemplates](60-reference-docs/50-analysis-templates.md). Demonstrates: [Common Case](30-patterns/index.md#common-case), [Control Flow Stages](30-patterns/index.md#control-flow-stages), [Rendered Configs](30-patterns/index.md#rendered-configs) patterns. |
