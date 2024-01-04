---
description: Learn how to work effectively with freight
sidebar_label: Working with freight
---

# Working With Freight

**Freight** is an important Kargo concept. A single "piece of freight" is a set
of references to one or more versioned artifacts, which may include one or more:

* Container images (from image repositories)

* Kubernetes manifests (from Git repositories)

* Helm charts (from chart repositories)

Freight can therefore be thought of as a sort of meta-artifact. Freight is what
Kargo seeks to progress from one stage to another.

:::info
To learn the fundamentals of freight and the warehouses that produce freight,
visit the [concepts doc](./concepts).
:::

The remainder of this page describes features of freight that will enabled you
to work more effectively.

## IDs

Like all Kubernetes resources, Kargo `Freight` resources have a `metadata.name`
field, which uniquely identifies each resource of that type within a given Kargo
project (a specially labeled Kubernetes namespace). When a `Warehouse` produces
a new `Freight` resource, it will compute a canonical representation of the
artifacts referenced by that resource and use that, in turn, to compute a SHA-1
hash. This becomes the value of both the resources's `metadata.name` _and_ `id`
fields. The deterministic method of computing this value makes it a unique
"fingerprint" of the collection of artifacts referenced by the `Freight`
resource.

:::info
Why include the same information in two different fields?

The Kargo team anticipates the possibility of someday permitting users to
manually create their own `Freight` resources -- i.e. `Freight` resources not
created by a `Warehouse`. In such a scenario, the value of the `metadata.name`
field would be user-defined and would not be guaranteed to reflect the artifacts
referenced by the resource.

Because Kargo utilizes a
[mutating admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
to ensure that the value of the `id` field is _always_ set correctly, Kargo can
rely on the value of that field to _always_ reflect the artifacts referenced by
the resource, even when the `metadata.name` field does not.
:::

## Aliases

While the `id` field (and `metadata.name` field, for now) contain predictably
computed SHA-1 hashes, such identifiers are, unarguably, not very user-friendly.
To make `Freight` resources easier for human users to identify, `Warehouse`s
automatically generate a human-friendly alias for every `Freight` resource they
produce and apply it as the value of the `Freight` resource's
`kargo.akuity.io/alias`
[label](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/).

:::info
Generating aliases of the form `<adjective>-<animal>` is a strategy borrowed
from Docker, which generates similar names for containers not explicitly named
by users.
:::

:::info
Why a label?

Kubernetes enforces the immutability of the `metadata.name` field for all
resources and a Kargo webhook enforces the immutability of the `id` field.

Kubernetes
[labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/),
by contrast, are both mutable and indexed, which which makes them ideal for use
as secondary identifiers.
:::

When using the Kargo CLI to query for `Freight` resources, the `alias` field is
always displayed:

```shell
kargo get freight --project kargo-demo
```

Sample output:

```shell
NAME/ID                                    ALIAS              AGE
f5f87aa23c9e97f43eb83dd63768ee41f5ba3766   mortal-dragonfly   35s
```

It is also displayed when using `kubectl` to query for `Freight` resources:

```shell
kubectl get freight --namespace kargo-demo
```

Sample output:

```shell
NAME                                       ALIAS              AGE
f5f87aa23c9e97f43eb83dd63768ee41f5ba3766   mortal-dragonfly   35s
```

:::info
The Kargo UI, to make efficient use of screen real estate, displays aliases
only, but a `Freight` resource's `id` can always be discovered by hovering over
its alias.
:::

:::note
Some, but not all, Kargo CLI commands will accepts `Freight` aliases as an
alternative to `Freights` IDs.

Eventually, all Kargo CLI commands will accept aliases as an alternative.
:::

### Updating Aliases

While every `Freight` resource is automatically assigned an alias, users may
sometimes wish to override that alias with one of their own choosing. This can
make it easier to identify a particularly important (or problematic) `Freight`
resource as it progresses through the `Stage`s of a pipeline.

This is conveniently accomplished via the Kargo CLI:

```shell
kargo update freight-alias \
  f5f87aa23c9e97f43eb83dd63768ee41f5ba3766 \
  frozen-tauntaun \
  --project kargo-demo
```

This can also be accomplished via `kubectl` commands `apply`, `edit`, `patch`,
etc.

:::note
Aliases cannot currently be updated via the Kargo UI, but this will be addressed
in the `v0.4.0` release.
:::

## Manual Approvals

The [concepts doc](http://localhost:3000/concepts#verifications) describes the
usual process by which `Freight` resources are _verified_ at each `Stage` in a
pipeline before becoming available to the next `Stage` or `Stage`s. In brief, it
typically requires the `Stage` to reach a healthy state _and_, if applicable,
any user-defined verification processes to complete with favorable results.

This is suitable for the average case wherein a new `Freight` resource is
expected to traverse the entirety of a pipeline on its way to production,
however, it is nearly inevitable that the occasional need for a "hotfix" will
arise, in which case it may sometimes be desirable to bypass one or more
`Stage`s in the pipeline.

To enable this, Kargo provides the ability to manually approve a `Freight`
resource for promotion to any given `Stage`. At present, this can be
accomplished only through the Kargo UI:

1. Click the three dots in the upper-right corner of any `Freight` resource
   select __Manually Approve__.

1. Click on the `Stage` resource for which you wish to approve the `Freight`.

:::note
Manual approvals cannot currently be granted via the Kargo CLI, but this will be
addressed in the `v0.4.0` release.

Manual approvals also cannot be granted via `kubectl` due to technical factors
preventing `kubectl` from updating `status` subresources of Kargo resources.
:::

:::note
Manually granting approval for a `Freight` resource to be promoted to any given
`Stage` requires the same level of permissions as would be required to carry out
that promotion, although, granting manual approval does _not_ automatically
create a corresponding `Promotion` resource.
:::

After successfully granting manual approval for a `Freight` resource to be
promoted to a given `Stage`, the `Freight` resource's `status` field will
reflect that approval.

The following depicts a `Freight` resource that has been verified in a `test`
`Stage` through the usual process, but has been manually approved for promotion
to the `prod` `Stage`. i.e. Any `Stage`s between `test` and `prod` may be
bypassed.

```shell
kargo get freight \
  --project kargo-demo \
  --output jsonpath-as-json={.status}
```

```shell
[
    {
        "approvedFor": {
            "prod": {}
        },
        "verifiedIn": {
            "test": {}
        }
    }
]
```
