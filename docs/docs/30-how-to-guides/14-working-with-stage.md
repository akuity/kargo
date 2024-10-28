---
description: Learn how to work effectively with stages
sidebar_label: Working with stage
---

# Working With Stage

The `Stage` is a core concept in Kargo's application orchestration, representing a structured checkpoint in a deployment pipeline where application components (known as `Freight`—such as versioned artifacts like container images or Kubernetes manifests) are processed before moving to the next `Stage`.

- **Requested Freight**: Defines which `Freight` should be moved into the `Stage`.
- **Promotion Templates**: Specifies the steps required to progress `Freight` from one `Stage` to another.
- **Verification**: Configures checks to ensure deployments meet specific criteria.

:::info
To understand the basic principles of Kargo `Stage`s, please review the [Concepts documentation](../concepts).
:::

## Refresh

Refreshing a `Stage` helps update its state based on any new changes in `Freight` or configuration. By refreshing, you can ensure that the `Stage` reflects the latest deployment status or artifact updates.

```shell
kargo refresh stage --project=kargo-demo staging
```

Running this command updates the `staging` `Stage` of the `kargo-demo` project to reflect the most recent `Freight` or configuration changes.

## Verification

Verification allows you to confirm that each `Freight` within a `Stage` meets the necessary requirements before promotion. Kargo provides the `kargo verify` command to manage this verification process, offering options to rerun or abort ongoing verifications.

To rerun the verification for `test` `Stage`:
```shell
kargo verify stage --project=kargo-demo test
```

If you need to stop an ongoing verification process in a `Stage`, you can use the `--abort` flag:
```shell
kargo verify stage --project=kargo-demo test --abort
```

This command stops the active verification in the `test` `Stage` of the `kargo-demo` project.
