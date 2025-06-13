---
sidebar_label: Cluster Level Configuration
---

# Cluster Level Configuration

This document is a guide to Kargo configuration options that are available to
operators _at runtime_. These are generally analogs to Project level
configuration options available to developers.

:::info
__Not what you were looking for?__

Most system level configuration options are exercised by operators at the time
of installation or upgrade. For details, refer to
[Common Configurations](./20-advanced-installation/30-common-configurations.md)
and the
[Kargo Helm Chart's README.md](https://github.com/akuity/kargo/tree/main/charts/kargo).
:::

## Triggering Artifact Discovery Using Webhooks

If your cluster contains many `Warehouse` resources, which periodically poll
artifact repositories, or if developers have
[configured any of those `Warehouse`s poorly](../50-user-guide/20-how-to-guides/30-working-with-warehouses.md#performance-considerations),
you may have elected to reduce the frequency with which all `Warehouse`s execute
their artifact discovery processes (i.e. You may have elected to _increase_ the
minimum polling interval. See
[Common Configurations](./20-advanced-installation/30-common-configurations.md/#tuning-warehouse-reconciliation-intervals).
)

If you have done this, it may have relieved degraded performance and helped to
avoid encountering rate limits, but it will have been accompanied by the
undesired side effect of increasing the average time required for every
`Warehouse` to notice new artifacts. _This can be overcome by configuring
repositories to alert Kargo to the presence of new artifacts via webhooks._

Developers are able to configure Kargo to listen for inbound webhook requests
from various sources
[at the Project level](../50-user-guide/20-how-to-guides/30-working-with-warehouses.md#triggering-artifact-discovery-using-webhooks),
however, in an organization with many separate repositories in one (or a few)
Git hosting providers or container image registries, it may make more sense for
an operator to configure Kargo to listen for inbound webhook requests _at the
cluster level_.

To illustrate, consider a GitHub organization having many repositories belonging
to different teams within the organization. Each team may have their own Kargo
Project(s) for self-managing their promotion pipelines. Instead of every team
configuring Project level GitHub webhook receivers that may trigger artifact
discovery only for their own applicable `Warehouse`s, you, as the operator, can
configure _one_ cluster-level GitHub webhook receiver to trigger the artifact
discovery process of every applicable `Warehouse` across all Projects.

This can be accomplished easily by updating your `ClusterConfig` resource's
`spec.webhookReceivers` field. If your cluster does not already have a
`ClusterConfig` resource, you can create one.

:::note
Every cluster hosting a Kargo control plane is permitted to have at most _one_
`ClusterConfig` resource. This limit is enforced by requiring all
`ClusterConfig` resources to be named `cluster`.
:::

A `ClusterConfig` resource's `spec.webhookReceivers` field may define one or
more _webhook receivers_. A webhook receiver is an endpoint on a (typically)
internet-facing HTTP server that is configured to receive and process requests
from specific sources, and in response, trigger the discovery process of any
`Warehouse` _across all Projects_ that subscribes to a repository URL referenced
by the request payload.

Most types of webhook receivers require you only to specify a unique name and a
reference to a `Secret`. The expected keys and values for each kind of webhook
receiver vary, and are documented on
[each receiver type's own page](../50-user-guide/60-reference-docs/80-webhook-receivers).

:::note
Because `ClusterConfig` resources are _cluster-scoped_ resources and Kubernetes
has no such thing as a "`ClusterSecret`" resource type (i.e. a cluster-scoped
analog to `Secret`), Kargo will look for the referenced `Secret` in a designated
namespace. By default, that namespace is `kargo-cluster-secrets`, but can be
changed by the operator at the time of installation. (Refer to the
[Kargo Helm Chart's README.md](https://github.com/akuity/kargo/tree/main/charts/kargo).
)
:::

:::info
`Secret`s referenced by a webhook receiver typically serve _two_ purposes.

1. _Often_, some value(s) from the `Secret`'s data map are shared with the
   webhook sender (GitHub, for instance) and used to help authenticate requests.
   Some senders may use such "shared secrets" as bearer tokens. Others may use
   them as keys for signing requests. In such cases, the corresponding webhook
   receiver knows exactly what to do with this information in order to
   authenticate inbound requests.

1. _Always_, some value(s) from the `Secret`'s data map are used as a seed in
   deterministically constructing a complex, hard-to-guess URL where the
   receiver will listen for inbound requests.

    Some webhook senders (Docker Hub, for instance), do not natively implement
    any sort of authentication mechanism. No secret value(s) need to be shared
    with such a sender and requests from the sender contain no bearer token, nor
    are they signed. For cases such as these, a hard-to-guess URL is, itself,
    a _de facto_ shared secret and authentication mechanism.

    __Note that if a `Secret`'s value(s) are rotated, the URL where the receiver
    listens for inbound requests will also change. This is by design.__

    Kargo does not watch `Secret`s for changes because it lacks the permissions
    to do so, so it can be some time _after_ its `Secret`'s value(s) are rotated
    that a webhook receiver's URL will be updated. To expedite that update, your
    `ClusterConfig` resource can be manually "refreshed" using the `kargo` CLI:

    ```shell
    kargo refresh clusterconfig
    ```

:::

The following example `ClusterConfig` configures two webhook receivers:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Cluster
metadata:
  name: cluster
spec:
  webhookReceivers:
  - name: my-first-receiver
    github:
      secretRef:
        name: my-first-secret
  - name: my-second-receiver
    gitlab:  
      secretRef:
        name: my-second-secret
---
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: my-first-secret
  namespace: kargo-cluster-secrets
data:
  secret: c295bGVudCBncmVlbiBpcyBwZW9wbGUK
---
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: my-second-secret
  namespace: kargo-cluster-secrets
data:
  secret-token: cm9zZWJ1ZCB3YXMgYSBzbGVkCg==
```

For each properly configured webhook receiver, Kargo will update the
`ClusterConfig` resource's `status` to reflect the URLs that can be registered
as endpoints with the senders.

For instance, the `ClusterConfig` and `Secret`s above result in the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ClusterConfig
metadata:
  name: cluster
spec:
  # ... omitted for brevity ...
status:
  conditions:
  - lastTransitionTime: "2025-06-11T22:53:21Z"
    message: ProjectConfig is synced and ready for use
    observedGeneration: 1
    reason: Synced
    status: "True"
    type: Ready
  webhookReceivers:
  - name: my-first-receiver
    path: /webhook/github/804b6f6bb40eb1f0e371f971d71dd95549be4bc9cbf868046941115f44073c67
    url: https://kargo.example.com/webhook/github/804b6f6bb40eb1f0e371f971d71dd95549be4bc9cbf868046941115f44073c67
  - name: my-second-receiver
    path: /webhook/gitlab/0eba9ff2a91f04f7787404b8f8f0edaf8cf8c39add34082651a474803cc99015
    url: https://kargo.example.com/webhook/gitlab/0eba9ff2a91f04f7787404b8f8f0edaf8cf8c39add34082651a474803cc99015
```

Above, you can see the URLs that can be registered with GitHub and GitLab as
endpoints to receive webhook requests from those platforms.

:::info
For more information about registering these endpoints with specific senders,
refer to
[each receiver type's own page](../50-user-guide/60-reference-docs/80-webhook-receivers).
:::

### Receivers in Action

Once a webhook receiver has been assigned a URL and that URL has been registered
with a compatible sender, the receiver will begin receiving webhook requests in
response to events in your repositories. The payload (body) of such a request
contains structured information (usually JSON) the sender wishes to share about
some event. Invariably, among this information, is the URL of the repository
from which the event originated.

A webhook receiver's only job is to extract a repository URL from the webhook
request's payload, query for all `Warehouse` resources across all Projects
having subscriptions to that repository, and request each to execute their
artifact discovery process.
