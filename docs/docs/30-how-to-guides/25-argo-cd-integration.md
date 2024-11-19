---
description: Learn how to integrate Kargo Stages with Argo CD Applications.
sidebar_label: Argo CD Integration
---

# Argo CD Integration

Kargo integrates seamlessly with Argo CD to facilitate a more streamlined application lifecycle
management process. While Argo CD helps with deploying Kubernetes objects and synchronizing changes
in the cluster, Kargo focuses on orchestrating the promotion of these changes through various
`Stage`s of development, such as from `development` to `testing` and then to `production`.

:::note
This page is a work in progress.
During this process, you may find limited details here. Please bear with us as we work to add more information.
:::

### Authorizing Kargo `Stage`s to Modify Argo CD Applications

To enable Kargo `Stage`s to interact with and modify Argo CD applications, applications need
to explicitly authorize Kargo to perform these actions. This is accomplished using the
`kargo.akuity.io/authorized-stage` annotation.

Kargo requires the annotation in the following format:

```yaml
kargo.akuity.io/authorized-stage: "<project-name>:<stage-name>"
```

This annotation signifies consent for Kargo to manage the application on behalf of the designated `Project` and `Stage`.

In the following example, the `Application` manifest is configured to
authorize the `test` `Stage` of the `kargo-demo` `Project` to manage
the application by including the `kargo.akuity.io/authorized-stage: kargo-demo:test`
annotation:

```yaml
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    metadata:
        name: kargo-demo-test
        namespace: argocd
        annotations:
            kargo.akuity.io/authorized-stage: kargo-demo:test
    spec:
        # Application Specifications
```
