---
description: Learn how to work effectively with Projects
sidebar_label: Working with Projects
---

# Working with Projects

Each Kargo project is represented by a cluster-scoped Kubernetes resource of
type `Project`. Reconciliation of such a resource effects all boilerplate
project initialization, including the creation of a specially-labeled
`Namespace` with the same name as the `Project`. All resources belonging to a
given `Project` should be grouped together in that `Namespace`.

A minimal `Project` resource looks like the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Project
metadata:
name: kargo-demo
```

:::note
Deletion of a `Project` resource results in the deletion of the corresponding
`Namespace`. For convenience, the inverse is also true -- deletion of a
project's `Namespace` results in the deletion of the corresponding `Project`
resource.
:::

:::info
There are compelling advantages to using `Project` resources instead of
permitting users to create `Namespace` resources directly:

* The required label indicating a `Namespace` is a Kargo project cannot be
forgotten or misapplied.

* Users can be granted permission to indirectly create `Namespace` resources for
Kargo projects _only_ without being granted more general permissions to create
_any_ new `Namespace` directly.

* Boilerplate configuration is automatically created at the time of `Project`
creation. This includes things such as project-level RBAC resources and
`ServiceAccount` resources.
:::

:::info
In future releases, the team also expects to also aggregate project-level status
and statistics in `Project` resources.
:::

### Promotion Policies

A `Project` resource can additionally define project-level configuration. At
present, this only includes **promotion policies** that describe which `Stage`s
are eligible for automatic promotion of newly available `Freight`.

:::note
Promotion policies are defined at the project-level because users with
permission to update `Stage` resources in a given project `Namespace` may _not_
have permission to create `Promotion` resources. Defining promotion policies at
the project-level therefore restricts such users from enabling automatic
promotions for a `Stage` to which they may lack permission to promote to
manually. It leaves decisions about eligibility for auto-promotion squarely in
the hands of someone like a "project admin."
:::

In the example below, the `test` and `uat` `Stage`s are eligible for automatic
promotion of newly available `Freight`, but any other `Stage`s in the `Project`
are not:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Project
metadata:
name: kargo-demo
spec:
promotionPolicies:
- stage: test
autoPromotionEnabled: true
- stage: uat
autoPromotionEnabled: true
```

## Namespace Adoption

To ensure compliance with governance policies, you may need
to pre-configure namespaces with specific labels or annotations
required by your organization's policy agents.

Kargo supports the adoption of pre-existing namespaces that are
labeled with `kargo.akuity.io/project: "true"`. This enables you
to pre-configure namespaces according to your organization's requirements.

For example, if your policy agent mandates that all namespaces include
a label identifying the internal organization responsible for them,
you can define the `namespace` in your YAML manifest with
the necessary labels and resources:

* Add the label `kargo.akuity.io/project: "true"` to your `namespace` definition.
* Ensure that in your YAML file, the `namespace` definition is listed above the
Kargo `Project` resource to allow it to be created first.

In this example, the `namespace` is pre-labeled to indicate the responsible
internal organization. When the Kargo `Project` is created, it automatically
adopts this pre-existing `namespace`.

```yaml
apiVersion: v1
kind: Namespace
metadata:
    name: kargo-example
labels:
    kargo.akuity.io/project: "true"
    org.example.com: responsible-org
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Project
metadata:
    name: kargo-example
spec:
# Project specifications go here
```

This setup allows the Kargo `Project` to recognize and utilize your pre-configured `namespace` seamlessly, aligning with your compliance requirements without additional updates.
