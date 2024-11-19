---
sidebar_label: Key Concepts
description: Find out more about key Kargo concepts - stages, freight, warehouses, promotions, and more
---
# Key Kargo Concepts

:::note
We're currently reorganizing and updating the documentation.
During this process, you may encounter sections that are incomplete,
repetitive, or fragmented. Please bear with us as we work to make improvements.
:::

## The Basics

### What is a Project

A **project** is a collection of related Kargo resources that describe one or
more delivery pipelines and is the basic unit of organization and tenancy in
Kargo.

RBAC rules are also defined at the project level and project administrators
may use projects to define policies, such as whether a **stage** is eligible
for automatic promotions of new **freight**.

:::note
For technical information on the corresponding `Project` resource
type, refer to [Working with Projects](./30-how-to-guides/11-working-with-projects.md).
:::

### What is a Stage?

When you hear the term “environment”, what you envision will depend
significantly on your perspective. A developer, for example, may think of
an "environment" as a specific _instance_ of an application they work on,
while a DevOps engineer, may think of an "environment" as a particular segment
of the infrastructure they maintain.

To eliminate confusion, Kargo avoids the term "environment" altogether in favor of **stage**.
The important feature of a stage is that its name ("test" or "prod," for instance)
denotes an application instance's _purpose_ and not necessarily its _location_.
[This blog post](https://akuity.io/blog/kargo-stage-not-environment/) discusses
the rationale behind this choice.

_Stages are Kargo's most important concept._ They can be linked together in a
directed acyclic graph to describe a delivery pipeline. Typically, such a
pipeline may feature a "test" or "dev" stage as its starting point, with one or
more "prod" stages at the end.

:::note
For technical details of the corresponding `Stage` resource type,
refer to [Working with Stages](./30-how-to-guides/14-working-with-stages.md).
:::

### What is Freight?

**Freight** is Kargo's second most important concept. A single "piece of
freight" is a set of references to one or more versioned artifacts, which may
include one or more:

* Container images (from image repositories)

* Kubernetes manifests (from Git repositories)

* Helm charts (from chart repositories)

Freight can therefore be thought of as a sort of meta-artifact. Freight is what
Kargo seeks to progress from one stage to another.
For detailed guidance on working with Freight, refer to
[this guide](./30-how-to-guides/15-working-with-freight.md).

### What is a Warehouse?

A **warehouse** is a _source_ of freight. A warehouse subscribes to one or more:

* Container image repositories

* Git repositories

* Helm charts repositories

Anytime something new is discovered in any repository to which a warehouse
subscribes, the warehouse produces a new piece of freight.

:::note
For technical details of the corresponding `Warehouse` resource type,
refer to [Working with Warehouses](./30-how-to-guides/16-working-with-warehouses.md).
:::

### What is a Promotion?

A **promotion** is a request to move a piece of freight into a specified stage.

## Corresponding Resource Types

Each of Kargo's fundamental concepts maps directly onto a custom Kubernetes
resource type.

### `Freight` Resources

Each piece of Kargo freight is represented by a Kubernetes resource of type
`Freight`. `Freight` resources are immutable except for their `alias` field
and `status` subresource (mutable only by the Kargo controller).

A single `Freight` resource references one or more versioned artifacts, such as:

* Container images (from image repositories)

* Kubernetes manifests (from Git repositories)

* Helm charts (from chart repositories)

A `Freight` resource's `metadata.name` field is a SHA1 hash of a canonical
representation of the artifacts referenced by the `Freight` resource. (This is
enforced by an admission webhook.) The `metadata.name` field is therefore a
"fingerprint", deterministically derived from the `Freight`'s contents.

To provide a human-readable identifier for a `Freight` resource, a `Freight`
resource has an `alias` field. This alias is
a human-readable string that is unique within the `Project` to which the
`Freight` belongs. Kargo automatically generates unique aliases for all
`Freight` resources, but users may update them to be more meaningful.

:::tip
Assigning meaningful and recognizable aliases to important pieces of `Freight`
traversing your pipeline(s) can make it easier to track their progress from one
`Stage` to another.
:::

:::note
For more information on aliases, refer to the [aliases](./30-how-to-guides/15-working-with-freight.md#aliases)
and [updating aliases](./30-how-to-guides/15-working-with-freight.md#updating-aliases)
sections of the "Working with Freight" how-to guide.
:::

A `Freight` resource's `status` field records a list of `Stage` resources in
which the `Freight` has been _verified_ and a separate list of `Stage` resources
for which the `Freight` has been manually _approved_.

`Freight` resources look similar to the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Freight
metadata:
  name: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
  namespace: kargo-demo
  labels:
    kargo.akuity.io/alias: fruitful-ferret
alias: fruitful-ferret
images:
- digest: sha256:b2487a28589657b318e0d63110056e11564e73b9fd3ec4c4afba5542f9d07d46
  repoURL: public.ecr.aws/nginx/nginx
  tag: 1.27.0
commits:
- repoURL: https://github.com/example/kargo-demo.git
  id: 1234abc
warehouse: my-warehouse
status:
  verifiedIn:
    test: {}
  approvedFor:
    prod: {}
```

### `Promotion` Resources

Each Kargo promotion is represented by a Kubernetes resource of type
`Promotion`.

A `Promotion` resource's two most important fields are its `spec.freight` and
`spec.stage` fields, which respectively identify a piece of `Freight` and a
target `Stage` to which that `Freight` should be promoted.

`Promotions` are, in some cases, created automatically by Kargo. In other cases,
they are created manually by users. In either case, a `Promotion` resource
resembles the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390-to-test
  namespace: kargo-demo
spec:
  stage: test
  freight: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
```

:::info
The name in a `Promotion`'s `metadata.name` field is inconsequential. Only
the `spec` matters.
:::

When a `Promotion` has concluded -- whether successfully or unsuccessfully --
the `Promotion`'s `status` field is updated to reflect the outcome. For example:

```yaml
status:
  phase: Succeeded
```

## Role-Based Access Control

As with all resource types in Kubernetes, permissions to perform various actions
on resources of different types are governed by
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

For all Kargo resource types, Kubernetes RBAC functions exactly as one would
expect, with one notable exception.

Often, it is necessary to grant a user permission to create `Promotion`
resources that reference certain `Stage` resources, but not others. To address
this, Kargo utilizes an admission control webhook that conducts access reviews
to determine if a user creating a `Promotion` resource has the virtual `promote`
verb for the `Stage` referenced by the `Promotion` resource.

:::info
[This blog post](https://blog.aquasec.com/kubernetes-verbs) is an excellent
primer on virtual verbs in Kubernetes RBAC.
:::

The following `Role` resource describes permissions to create `Promotion`
references that reference the `uat` `Stage` only:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: uat-promoter
  namespace: kargo-demo
rules:
- apiGroups:
  - kargo.akuity.io
  resources:
  - promotions
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - kargo.akuity.io
  resources:
  - stages
  resourceNames:
  - uat
  verbs:
  - promote
```

To grant a fictional user `alice`, in the QA department, the ability to promote
to `uat` only, create a corresponding `RoleBinding`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alice-uat-promoter
  namespace: kargo-demo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: uat-promoter
subjects:
- kind: User
  name: alice
```
