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
or the component-specific parameter sections in the chart documentation.
:::

## API Configuration

Kargo supports a number of API-related configurations that can be set at
installation time. These configurations are used to control the behavior of
Kargo's API server and its web-based user interface.

:::info
The sections below outline common configurations for the API server. For a full
list of supported configurations, refer to the
[API Parameters](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md#api).
section in the chart documentation.
:::

### API Host

By default, the API server host (i.e. the domain or IP address that the API is
accessible at) is set to `localhost`. This host is used for generation of
`Ingress` resources, certificates, the OpenID Connect issuer and callback URLs,
and any URLs that are exposed to users.

To configure the API host, set the following configuration:

```yaml
api:
  host: kargo.example.com
```

:::note
The host is allowed to include a port number, e.g. `kargo.example.com:8080`,
but should not include a protocol (e.g. `http://` or `https://`) as this is
automatically inferred from other configuration options.
:::

### API Service

By default, Kargo will create a `Service` resource for the API server with the
type `ClusterIP`, which is only accessible within the cluster or through
[port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/).

:::caution
Changing the API server service type can expose the API server to the internet
in an insecure manner. Refer to the
[Secure Configuration](../40-security/10-secure-configuration.md) section for
more information on securing the API server.
:::

:::info
Instead of making use of a `Service` resource, you can also expose the API
server using an [`Ingress` resource](#api-ingress).
:::

If you want to expose the API server to the internet, but do not want to make
use of an `Ingress` resource, you can change the service type:

```yaml
api:
  service:
    type: LoadBalancer
```

In addition, when using a `LoadBalancer` or `NodePort` service type, you can
configure the service to use a specific port:

```yaml
api:
  service:
    type: LoadBalancer
    port: 443
```

### API TLS

By default, Kargo will enable TLS directly on the API server using a
self-signed certificate issued by [cert-manager](https://cert-manager.io/).

:::note
When making use of the self-signed certificate option, cert-manager must be
installed in the cluster.
:::

To supply your own certificate, set the following configuration:

```yaml
api:
  tls:
    selfSignedCert: false
```

When setting `selfSignedCert` to `false`, a
[TLS `Secret`](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets)
with the name `kargo-api-cert` is expected to exist in the same namespace as
the API server.

#### Terminating TLS

If you have opted out of all the chart's built-in TLS options, but TLS is
terminated upstream from the API server by a component such as a reverse proxy,
load balancer, or `Ingress` controller using a wildcard certificate, Kargo will be
unable to infer from its own configuration that any generated URLs should use
HTTPS.

For such cases, you can explicitly declare that TLS is terminated upstream and
Kargo will sort out the rest:

```yaml
api:
  tls:
    enabled: false
    terminatedUpstream: true
```

:::info
If you exercise any of the chart's built-in TLS configuration options,
setting `terminatedUpstream` is never required.
:::

### API Ingress

:::info
Instead of making use of an `Ingress` resource, you can also expose the API
server using a `LoadBalancer` or `NodePort` service type. Refer to the
[API Service](#api-service) section for more information.
:::

By default, Kargo will not create an `Ingress` resource for the API server
and will only be accessible within the cluster or through the API server's
[`Service` resource](#api-service). To expose the API server to the internet,
an `Ingress` resource can be created.

:::caution
Enabling the API server Ingress without proper configuration can expose the API
server to the internet in an insecure manner. Refer to the
[Secure Configuration](../40-security/10-secure-configuration.md) section for
more information on securing the API server.
:::

To configure the API server to use an `Ingress` resource, set the following
configuration:

```yaml
api:
  ingress:
    enabled: true
    ingressClassName: nginx
```

#### Ingress TLS

By default, Kargo will enable TLS on the `Ingress` resource using a self-signed
certificate using [cert-manager](https://cert-manager.io). 

:::note
When making use of the self-signed certificate option, cert-manager must be
installed in the cluster.
:::

To supply your own certificate, set the following configuration:

```yaml
api:
  ingress:
    tls:
      selfSignedCert: false
```

When setting `selfSignedCert` to `false`, the `Ingress` resource expects a
[TLS `Secret`](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)
with the name `kargo-api-ingress-cert` to exist in the same namespace as the
API server.

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

When disabled, the controller will not watch Argo CD `Application` resources
and disable Argo CD specific features.

:::info
If the integration is enabled without Argo CD installed, the controller will
disable the Argo CD specific features on boot. Explicitly disabling is preferred
if this integration is not desired, as it will grant fewer permissions to the
controller.
:::

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

In certain cases, you may not want Kargo to be able to look up Argo CD resources
in other namespaces than the configured default. This can be enforced by setting
the following configuration:

```yaml
controller:
  argocd:
    watchArgocdNamespaceOnly: true
```

When enabled, the controller will only watch for `Application` resources in the
namespace specified by `controller.argocd.namespace`.

## Argo Rollouts Configuration

### Disabling the Argo Rollouts Integration

By default, Kargo will enable the Argo Rollouts integration, which configures
Kargo to work with `Rollout` resources created by Argo Rollouts as part of the
[verification feature](../../50-user-guide/20-how-to-guides/40-working-with-stages.md#verification).

This can be disabled as follows:

```yaml
controller:
  argoRollouts:
    integrationEnabled: false
```

When disabled, the controller will not reconcile Argo Rollouts `AnalysisRun`
resources and attempts to verify Stages via `Analysis` will fail.

:::info
If the integration is enabled without Argo Rollouts installed, the controller
will disable the Argo Rollouts specific features on boot. Explicitly disabling
is preferred if this integration is not desired, as it will grant fewer
permissions to the controller.
:::

### Logs from Job Metrics

For those utilizing Argo Rollouts integration for verifications,
[job metrics](https://argoproj.github.io/argo-rollouts/analysis/job/) stand out
as an especially useful feature because they are implemented as Kubernetes
[`Job`s](https://kubernetes.io/docs/concepts/workloads/controllers/job/), which
give users the flexibility to define any arbitrary post-promotion tests they'd
like to run against a `Stage` by simply providing appropriate `Job` specs
[as described here](../../50-user-guide/20-how-to-guides/60-verification.md#configuring-analysistemplates).
In cases such as these, access to logs produced by the `Job`'s underlying `Pod`
is helpful for debugging purposes and for understanding results.

Since it's common for multiple Kargo controllers to be deployed to many
different clusters, such logs need to be aggregated in a centralized location
for the API server to access them and, in-turn, stream them to the Kargo UI or
CLI. Rather than build support for many different logging stacks, Kargo has
settled on a "lowest common denominator" approach: The API server can stream
any logs that it can access via an HTTP GET request.

To facilitate this, operators may, at the time of installation, provide a URL
template that the API server can use to construct the URL for any job metric logs
as a function of `project` (name), `namespace` (always equal to the `Project`'s
name), `shard`, `stage` (name), `analysisRun` (name), `metricName`,
`jobNamespace`, `jobName`, and `container` (name).

A token can be specified by referencing a Kubernetes `Secret` that is managed
"out of band." HTTP headers may also be specified, and may reference the token
if one is provided.

Example:

```yaml
api:
  rollouts:
    integrationEnabled: true
    logs:
      enabled: true
      urlTemplate: https://logs.kargo.example.com/${{project}}/${{analysisRun}}/${{jobName}}/${{container}}
      tokenSecret:
        name: kargo-logs-token
        key: token
      httpHeaders:
        Authentication: "Bearer ${{ token }}"
```

:::note
This "lowest common denominator" approach to streaming job metric logs does
leave it as an exercise for the Kargo administrator to arrange for the
forwarding and storage of applicable logs.

Users of Kargo via the [Akuity Platform](https://akuity.io/akuity-platform),
however, will have this seamlessly handled for them.
:::

:::note
For more information, refer to the
[chart documentation](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md).
:::

## Resource Management

### Tuning Concurrent Reconciliation Limits

By default, Kargo will reconcile up to four resources of the same kind
concurrently (e.g. four `Stage` resources at a time). It is possible to tune
this limit by setting a new global value, or by setting a limit per resource
kind:

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
