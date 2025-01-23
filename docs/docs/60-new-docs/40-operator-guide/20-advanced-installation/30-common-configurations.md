---
sidebar_label: Common Configurations
---

# Common Configurations

This document outlines common advanced installation scenarios and
configurations for Kargo. 

:::info
For complete parameter documentation, refer to the
[chart documentation](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md).
:::

:::info
For more information on how to apply these configurations, see the
advanced installation guides for [Helm](10-advanced-with-helm.md) and
[Argo CD](20-advanced-with-argocd.md).
:::

## Standard Kubernetes Configuration

For common Kubernetes configurations, like setting a `nodeSelector`, `labels`,
`annotations`, `affinity`, `tolerations`, etc. Kargo supports both global and
resource-specific configurations. For example, to add `labels` to all resources
created by Kargo, the following configuration can be used:

```yaml
global:
  labels:
    label.example.com/key: value
```

To add `labels` to a specific component (the `kargo-controller` in this example),
the following configuration can be used:
    
```yaml
controller:
  labels:
    label.example.com/key: value
```

:::note
For a full list of supported configurations, refer to the
[Global Parameters](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md#global-parameters)
or the component-specific parameters in the chart documentation.
:::

## Git Configuration

Kargo supports a number of Git-related configurations that can be set at
installation time.

### Default Commit Author

To set the default commit author Kargo will use when committing changes using
the [`git-commit` Promotion step](../../50-user-guide/60-reference-docs/30-promotion-steps/git-commit.md),
the following configuration can be set (shown here with the default values):

```yaml
controller:
  gitClient:
    name: Kargo
    email: no-reply@kargo.io
```

### Signing Commits

To sign commits made by Kargo, a reference to a `Secret` (in the same namespace
as Kargo is installed to) containing a signing key can be configured. The key
should be a GPG key in ASCII-armored format  without a passphrase, under the
key `signingKey`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kargo-git-signing-key
type: Opaque
data:
  signingKey: <base64-encoded-ascii-armored-gpg-key>
```

The `Secret` can then be referenced in the Kargo configuration.

```yaml
controller:
  gitClient:
    signingKeySecret:
      name: kargo-git-signing-key
      type: gpg
```

:::note
When using a signing key, the `gitClient.name` and `gitClient.email`
configuration options must match the name and email associated with the GPG
key.
:::

## Argo CD Configuration

Kargo supports a number of Argo CD-related configurations that can be set at
installation time.

### Disabling the Argo CD Integration

By default, Kargo will enable the Argo CD integration, which configures Kargo
to work with `Application` resources created by Argo CD. This can be disabled
as follows:

```yaml
controller:
  argocd:
    integrationEnabled: false
```

When not enabled, the controller will not watch Argo CD `Application` resources
and disable Argo CD specific features. Explicitly disabling is preferred if this
integration is not desired, as it will grant fewer permissions to the controller.

### Argo CD Namespace

By default, Kargo expects Argo CD to be installed to the `argocd` namespace,
which is also the default namespace it will use for `Application` resources
if a namespace is not specified in the
[`argocd-update` Promotion step](../../50-user-guide/60-reference-docs/30-promotion-steps/argocd-update.md).

If you want to use a different default namespace, this can be configured as
follows:

```yaml
controller:
  argocd:
    namespace: argocd
```

## Argo Rollouts Configuration

### Disabling the Argo Rollouts Integration

By default, Kargo will enable the Argo Rollouts integration, which configures
Kargo to work with `Rollout` resources created by Argo Rollouts as part of the
[verification feature](../../50-user-guide/20-how-to-guides/14-working-with-stages.md#verifications).

This can be disabled as follows:

```yaml
controller:
  argoRollouts:
    integrationEnabled: false
```

When not enabled, the controller will not reconcile Argo Rollouts `AnalysisRun`
resources and attempts to verify Stages via `Analysis` will fail. Explicitly
disabling is preferred if this integration is not desired, as it will grant
fewer permissions to the controller.

## Resource Management

### Tuning Concurrent Reconciliation Limits

By default, Kargo will reconcile up to 4 resources of the same kind concurrently
(e.g. 4 `Stage` resources at a time). It is possible to tune this limit by
setting a new global value, or by setting a limit per resource kind:

```yaml
controller:
  reconcilers:
    # Global setting
    maxConcurrentReconciles: 4

  managementController:
    # Kind specific setting
    namespaces:
      maxConcurrentReconciles: 2
```

:::note
For a list of resource kinds that can be configured, refer to the
[chart documentation](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md).
:::

## Garbage Collection

Kargo includes a garbage collector that automatically removes old `Freight` and
`Promotion` resources. The garbage collector offers a number of configuration
options to manage the retention of these resources.

### Disabling the Garbage Collector

By default, the garbage collector is enabled. To disable it, set the following
configuration:

```yaml
garbageCollector:
  enabled: false
```

:::caution
Disabling the garbage collector will result in old `Freight` and `Promotion`
resources accumulating in the cluster. This can lead to increased resource
usage and potential performance issues. Therefore, this is typically not
recommended and should only be done with caution.
:::

### Scheduling the Garbage Collection

By default, the garbage collector runs every 24 hours. This can be configured
using a cron expression:

```yaml
garbageCollector:
  schedule: "0 0 * * *"
```

### Retention Settings

The garbage collector offers a number of settings to control the retention of
`Freight` and `Promotion` resources. The following settings are available:

```yaml
garbageCollector:
  # The minimum age a Promotion resource must be before it can be deleted.
  # This is a duration string (e.g. 336h for 14 days).
  minPromotionDeletionAge: 336h
  # The number of Promotion resources for each Stage to retain that are older
  # than the minimum deletion age. I.e., if a Stage has 30 Promotions older
  # than minPromotionDeletionAge, only the 20 most recent will be retained.
  maxRetainedPromotions: 20
  
  # The minimum age a Freight resource must be before it can be deleted.
  # This is a duration string (e.g. 336h for 14 days).
  minFreightDeletionAge: 336h
  # The number of Freight resources for each Warehouse to retain that are older
  # than the minimum deletion age. I.e., if a Warehouse has 20 Freight older
  # than minFreightDeletionAge, only the 20 most recent will be retained.
  maxRetainedFreight: 10
```

:::note
`Promotion` resources are only deleted if they are in a terminal state (i.e.
`Succeeded` or `Failed`). `Freight` resources are only deleted if they are not
actively in use by any `Stage`.

In both cases, this holds true even if the resource is older than the minimum
deletion age.
:::
