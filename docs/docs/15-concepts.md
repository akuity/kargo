---
title: Concepts
description: Concepts
---

This section covers important Kargo concepts.

## The Basics

### What is a Stage?

When you hear the term “environment”, what you envision will depend
significantly on your perspective. To eliminate confusion, Kargo avoids the term
"environment" altogether in favor of **stage**. The important feature of a stage
is that its name ("test" or "prod," for instance) denotes an application
instance's _purpose_ and not necessarily its _location_.
[This blog post](https://akuity.io/blog/kargo-stage-not-environment/) discusses
the rationale behind this choice.

_Stages are Kargo's most important concept._ They can be linked together in a
directed asyclic graph to describe a delivery pipeline. Typically, such a
pipeline may feature a "test" or "dev" stage as its starting point, with one or
more "prod" stages at the end.

### What is Freight?

**Freight** is Kargo's second most important concept. A single "piece of
freight" is a set of references to one or more versioned artifacts, which may
include one or more:

* Container images (from image repositories)

* Kubernetes manifests (from Git repositories)

* Helm charts (from chart repositories)

Freight can therefore be thought of as a sort of meta-artifact. Freight is what
Kargo seeks to progress from one stage to another.

### What is a Warehouse?

A **warehouse** is a _source_ of freight. A warehouse subscribes to one or more:

* Container image repositories

* Git repositories

* Helm charts repositories

Anytime something new is discovered in any repository to which a warehouse
subscribes, the warehouse produces a new piece of freight.

### What is a Promotion?

A **promotion** is a request to move a piece of freight into a specified stage.

### What is a Promotion Policy?

A **promotion policy**, at present, determines only whether a specific stage is
eligible to for new freight to be automatically promoted into it.

## Corresponding Resource Types

Each of Kargo's fundamental concepts maps directly onto a custom Kubernetes
resource type.

:::info
Related resources must be grouped together in a single project, which is a
specially labeled Kubernetes `Namespace`. In our examples, we group all of our
resources together in a `kargo-demo` namespace.
:::

### `Stage` Resources

Each Kargo stage is represented by a Kubernetes resource of type `Stage`.

A `Stage` resource's `spec` field decomposes into two main areas of concern:

* Subscriptions

* Promotion mechanisms

The following sections will explore each of these in greater detail.

#### Subscriptions

The `spec.subscriptions` field is used to describe the sources from which a
`Stage` obtains `Freight`. These subscriptions can be to a single `Warehouse` or
to one or more "upstream" `Stage` resources.

For each `Stage`, the Kargo controller will periodically check its subscriptions
for new _qualified_ `Freight`.

For a `Stage` subscribed directly to a `Warehouse`, _any_ new `Freight` resource
from that `Warehouse` is considered qualified.

For a `Stage` subscribed to one or more "upstream" `Stage`s, `Freight` is
qualified only after those "upstream" `Stage` resources have reached a healthy
state while hosting that `Freight`.

In the following example, the `test` `Stage` subscribes to a single `Warehouse`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  subscriptions:
    warehouse: my-warehouse
  # ...
```

In this example, the `uat` `Stage` subscribes to the `test` `Stage`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: uat
  namespace: kargo-demo
spec:
  subscriptions:
    stages:
    - test
  # ...
```

#### Promotion Mechanisms

The `spec.promotionMechanisms` field is used to describe _how_ to transition
`Freight` into the `Stage`.

There are two general methods of accomplishing this:

* Committing changes to a GitOps repository.

* Making changes to an Argo CD `Application` resource. (Often, the only change
  is to force a sync and refresh of the `Application`.)

These two approaches are, in many cases, used in conjunction with one another.
The Kargo controller always applies Git-based promotion mechanisms first _then_
Argo CD-based promotion mechanisms.

Included among the Git-based promotion mechanisms is specialized support for:

* Running `kustomize edit set image` for a specified directory, then committing
  the changes, if any.

* Updating the value of a key in a Helm values file, then committing the
  changes, if any.

* Updating a `Chart.yaml` file in a Helm "umbrella chart," then committing the
  changes, if any.

And among the Argo CD-based promotion mechanisms, there is specialized support
for:

* Updating image overrides in the `kustomize` section of a specified Argo CD
  `Application` resource.

* Updating the value of a key in the `helm` section of a specified Argo CD
  `Application` resource to point at a new Docker image.

* Updating a specified Argo CD `Application` resource's `targetRevision`
  field(s) to point at a specific commit in a Git repository or a specific
  version of a Helm chart.

* Forcing a specified Argo CD `Application` to refresh and sync. (This is
  automatic for any `Application` resource a `Stage` interacts with.)

:::info
Additionally, interaction with any Argo CD `Application` resources(s) as
described above implicitly results in periodic evaluation of `Stage` health by
aggregating the results of sync/health state for all such `Application`
resources(s).
:::

The following example, shows that transitioning `Freight` into the `test`
`Stage` requires:

1. Updating the `https://github.com/example/kargo-demo.git` repository by
   running `kustomize edit set image` in the `stages/test` directory and
   committing those changes to a stage-specific `stages/test` branch.

1. Forcing the Argo CD `Application` named `kargo-demo-test` in the `argocd`
   namespace to refresh and sync.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
  promotionMechanisms:
    gitRepoUpdates:
    - repoURL: https://github.com/example/kargo-demo.git
      writeBranch: stages/test
      kustomize:
        images:
        - image: nginx
          path: stages/test
    argoCDAppUpdates:
    - appName: kargo-demo-test
      appNamespace: argocd
```

#### Status

A `Stage` resource's `status` field records:

* The `Freight` currently deployed to the `Stage`.

* History of `Freight` that has been deployed to the `Stage`. (From most to
  least recent.)

* The health status any any associated Argo CD `Application` resources.

For example:

```yaml
status:
  currentFreight:
    id: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
    images:
    - repoURL: nginx
      tag: 1.25.3
    commits:
    - repoURL: https://github.com/example/kargo-demo.git
      id: 1234abc
  health:
    argoCDApps:
    - healthStatus:
        status: Healthy
      name: kargo-demo-test
      namespace: argocd
      syncStatus:
        revision: 4b1bd08ffbaecf0961e1877d7f2cc8bde7090575
        status: Synced
    status: Healthy
  history:
  - id: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
    images:
    - repoURL: nginx
      tag: 1.25.3
    commits:
    - repoURL: https://github.com/example/kargo-demo.git
      id: 1234abc
```

### `Freight` Resources

Each piece of Kargo freight is represented by a Kubernetes resource of type
`Freight`. `Freight` resources are immutable except for their `status` field.

A single `Freight` resource references one or more versioned artifacts, such as:

* Container images (from image repositories)

* Kubernetes manifests (from Git repositories)

* Helm charts (from chart repositories)

A `Freight` resource's `id` field _and_ `metadata.name` field are both (for now)
set to the same value, which is a SHA1 hash of a canonical representation of the
artifacts referenced by the `Freight` resource. (This is enforced by an
admission webhook.) The `id` and `metadata.name` fields, therefore, are both
"fingerprints," deterministically derived from the `Freight`'s contents.

A `Freight` resource's `status` field records a list of `Stage` resources in
which the `Freight` has been _qualified_. A `Freight` resource is qualified in
any `Stage` that reached a healthy state while hosting it.

`Freight` resources look similar to the following:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Freight
metadata:
  name: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
  namespace: kargo-demo
id: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
images:
- repoURL: nginx
  tag: 1.25.3
commits:
- repoURL: https://github.com/example/kargo-demo.git
  id: 1234abc
status:
  qualifications:
    test: {}
```

### `Warehouse` Resources

Each Kargo warehouse is represented by a Kubernetes resource of type
`Warehouse`.

A `Warehouse` resource's most important field is its `spec.subscriptions` field,
which is used to subscribe to one or more:

* Container image repositories

* Git repositories

* Helm charts repositories

The following example shows a `Warehouse` resource that subscribes to a
container image repository and a Git repository:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - image:
      repoURL: nginx
      semverConstraint: ^1.24.0
  - git:
      repoURL: https://github.com/example/kargo-demo.git
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

### `PromotionPolicy` Resources

Each Kargo promotion policy is represented by a Kubernetes resource of type
`PromotionPolicy`.

A `PromotionPolicy` resource's two most important fields are its `stage` and
`enableAutoPromotion` fields, which, respectively, identify a `Stage` and
indicate whether that `Stage` should be eligible to automatically receive new
`Freight`.

The following example shows a `PromotionPolicy` resource that enables automatic
promotions to the `test` `Stage`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionPolicy
metadata:
  name: test
  namespace: kargo-demo
stage: test
enableAutoPromotion: true
```

Kargo considers auto-promotion disabled by default for any `Stage` that does not
have a corresponding `PromotionPolicy` resource. If a `Stage` has multiple
corresponding `PromotionPolicy` resources, then the policy is considered
ambiguous and auto-promotion will be considered disabled by default.

:::note
Promotion policies are represented by a separate resource type (instead of being
a field on a `Stage` resource) because the users with authority to decide what
`Stage` resources should be eligible for auto-promotion may not be the same
users with authority to define the `Stage` resources themselves.
:::

## Role-Based Access Control

As with all resource types in Kubernetes, permissions to perform various actions
on resources of different types are governed by
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

For all Kargo resource types, Kubernetes RBAC functions exactly as one would
expect, with one notable exception.

Often, it is necessary to grant a user permission to create `Promotion`
resources that reference certain `Stage`s, but not others. To address this,
Kargo utilizes an admission control webhook that conducts access reviews to
determine if a user creating a `Promotion` resource has the virtual `promote`
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
