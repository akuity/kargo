---
description: Learn how to work effectively with freight
sidebar_label: Working with Freight
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
see [Core Concepts](../10-core-concepts/index.md).
:::

The remainder of this page describes features of freight that will enable you
to work more effectively.

## Names

Like all Kubernetes resources, Kargo `Freight` resources have a `metadata.name`
field, which uniquely identifies each resource of that type within a given Kargo
project (a specially labeled Kubernetes namespace). When a `Warehouse` produces
a new `Freight` resource, it will compute a canonical representation of the
artifacts referenced by that resource and use that, in turn, to compute a SHA-1
hash. This becomes the value of the `metadata.name` field. The deterministic
method of computing this value makes it a unique "fingerprint" of the
collection of artifacts referenced by the `Freight` resource.

## Aliases

While the `metadata.name` field contains a predictably computed SHA-1 hash,
such identifiers are, unarguably, not very user-friendly.
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
resources.

Kubernetes
[labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/),
by contrast, are both mutable and indexed, which makes them ideal for use as
secondary identifiers.
:::

When using the Kargo CLI to query for `Freight` resources, the `alias` field is
always displayed:

```shell
kargo get freight --project kargo-demo
```

Sample output:

```shell
NAME                                       ALIAS              AGE
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
only, but a `Freight` resource's `name` can always be discovered by hovering
over its alias.
:::

:::note
Kargo CLI commands will accept `Freight` aliases as an alternative to a
`Freight` name. Refer to the help text for the `kargo` command for more
information.
:::

### Updating Aliases

While every `Freight` resource is automatically assigned an alias, users may
sometimes wish to override that alias with one of their own choosing. This can
make it easier to identify a particularly important (or problematic) `Freight`
resource as it progresses through the `Stage`s of a pipeline.

To update the `Freight` Alias:

<Tabs groupId="update-alias">
<TabItem value="ui" label="Using the UI" default>

1. Click the three dots on the `Freight` in the <Hlt>Freight Timeline</Hlt>,
    then select <Hlt>Change Alias</Hlt>:

    ![update-freight-alias](img/freight-alias.png)
1. Update Alias and submit it.

    ![update-freight-alias](img/freight-alias-2.png)

</TabItem>

<TabItem value="cli" label="Using the CLI">

```shell
kargo update freight \
  --project=kargo-demo \
  --name=f5f87aa23c9e97f43eb83dd63768ee41f5ba3766 \
  --new-alias=frozen-tauntaun
```
Alternatively, you can reference the `Freight` to which you want to assign a new alias using its existing alias:

```shell
kargo update freight \
  --project=kargo-demo \
  --old-alias=mortal-dragonfly \
  --new-alias=frozen-tauntaun
```

This can also be accomplished via `kubectl` commands `apply`, `edit`, `patch`,
etc. by updating the `alias` field of the `Freight` resource.

</TabItem>
</Tabs>

:::info
The `alias` field is a convenient way to update the `Freight` resource's
`kargo.akuity.io/alias` label, which causes a webhook to sync the field value
to the label value. The precedence rules for syncing between the field and
label values are as follows:

- If the field has a non-empty value, the label will assume the field's value.
- If the field has an empty value, the field will assume the label's value.

It's worth noting that removing an alias entirely requires clearing both the
field and label values, but this is expected to be a rare occurrence.
:::

## Manual Approvals

:::note
**Maintainers Note**

This section references docs that have been moved or not written yet.
:::

The [Core Concepts](../10-core-concepts/index.md) describes the
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
resource for promotion to any given `Stage`. To manually approve the `Freight`:

<Tabs groupId="manual-approval">
<TabItem value="ui" label="Using the UI" default>

1. Click the three dots on the `Freight` in the <Hlt>Freight Timeline</Hlt>,
    then select <Hlt>Manually Approve</Hlt>:

    ![update-freight-alias](img/freight-approval.png)

1. Select <Hlt>Approve</Hlt> on the `Stage` where you'd want to promote the `Freight`.

    ![update-freight-alias](img/freight-approval-2.png)

</TabItem>

<TabItem value="cli" label="Using the CLI">

```shell
kargo approve \
  --freight f5f87aa23c9e97f43eb83dd63768ee41f5ba3766 \
  --stage prod \
  --project kargo-demo
```

:::note
Manual approvals cannot be granted via `kubectl` due to technical factors
preventing `kubectl` from updating `status` subresources of Kargo resources.
:::

</TabItem>
</Tabs>

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
bypassed:

<Tabs groupId="freight-status">
<TabItem value="ui" label="To check using the UI" default>

1. Click on the `Freight` in the <Hlt>Freight Timeline</Hlt>:

    ![freight-status](img/freight-status.png)

</TabItem>

<TabItem value="cli" label="To check using the CLI">

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

</TabItem>
</Tabs>

## Promoting a Freight to a Stage

<Tabs groupId="promoting-freight-to-a-stage">
<TabItem value="ui" label="Using the UI" default>

1. Click the three dots on the `Freight` in the <Hlt>Freight Timeline</Hlt>,
    then select <Hlt>Promote</Hlt>:

    ![Promote Freight to a Stage](img/freight-promotion.png)

2. Select the `Stage` and click <Hlt>Promote</Hlt>:

    ![Promote Freight to a Stage](img/freight-promotion-2.png)

</TabItem>

<TabItem value="cli" label="Using the CLI">

To promote `Freight` to a `Stage` using the CLI, run:

```shell
kargo promote --stage <stage> --project <project> --freight <freight>
```

</TabItem>
</Tabs>
