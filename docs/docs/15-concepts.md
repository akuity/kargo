---
title: Concepts
description: Concepts
---

This section covers important Kargo concepts.

## What is a `stage`?

When you hear the term “environment”, what you envision will depend significantly
on your perspective. To eliminate confusion, Kargo avoids the term "environment"
altogether in favor of something more precise: _stage_. The important feature of
a _stage_ is that its name ("test" or "prod," for instance) denotes an application
instance's _purpose_ and not its _location_. [This blog post](https://akuity.io/blog/kargo-stage-not-environment/) discusses the rationale behind this.

The progression of new materials from stage-to-stage can be fully automated or
manually triggered, as called for by your use cases or preferences.

:::info
As a matter of convention, throughout this documentation, we are careful to use
`Stage` (capitalized and monospaced) when we're referring specifically to that
custom resource type and "stage" in standard typeface when referring to the
concept.
:::

## `Stage` Resources

Like many Kubernetes resource types, the `Stage` resource type is decomposed
into three main sections:

* `metadata` that describes the resource's identifying information, such as
  name, namespace, labels, etc.

  :::info
  Related `Stage` resources should be grouped together in a single, dedicated
  namespace. In our examples, we group all of our `Stage` resources together in
  a `kargo-demo` namespace.
  :::

* `spec` that encapsulates the user-defined particulars of each resource.

* `status` that encapsulates resource state.

At this highest level, a manifest describing a `Stage` resource may appear as
follows:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
status:
  # ...
```

A `Stage` resource's `spec` field further decomposes into two main areas of
concern:

* Subscriptions

* Promotion mechanisms

The following sections will explore each of these in greater detail.

### Subscriptions

The `spec.subscriptions` field is used to describe the sources from which a
`Stage` obtains *materials*. Materials include any combination of the following:

* Manifests from a Git repository. These can be plain YAML or rendered with the
  assistance of configuration management tools like
  [Kustomize](https://kustomize.io/) or [Helm](https://helm.sh/).

* Docker images from an image repository.

* Helm charts from a chart repository.

Alternatively, instead of subscribing directly to repositories, a `Stage` may
subscribe to another, "upstream" `Stage`.

For each `Stage`, the Kargo controller will periodically check all subscriptions
for the latest available materials. A single set of materials is known as a
*freight* (or _piece_ of freight). The controller produces a canonical
representation of each piece of freight and uses that to calculate a SHA1 hash
which becomes its ID. Because freight IDs are calculated deterministically from
the underlying materials, each ID is intrinsically a _fingerprint_. Two pieces
of freight having the same materials have the same ID. (This enables cheap
comparisons.) This ID is compared to those in a stack of known, *available
freight* stored in the `Stage` resource's `status` field. If a piece of freight
is new, it is pushed onto the stack and becomes *available.*

In the following example, the `test` `Stage` subscribes to manifests from a Git
repository _and_ images from an image repository:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  subscriptions:
    repos:
      git:
      - repoURL: https://github.com/example/kargo-demo.git
        branch: main
      images:
      - repoURL: nginx
        semverConstraint: ^1.24.0
  # ...
```

This is how the `test` `Stage` resource's `status` field may appear after
polling the two repositories to which it subscribes:

```yaml
status:
  availableFreight:
  - id: 51636b9332d5938b9f2d382e9713b54ceb62a323
    firstSeen: "2023-04-21T18:34:56Z"
    commits:
    - id: dd8dc6a021d9d6c42e937f8b8f221a838342ec2a
      repoURL: https://github.com/example/kargo-demo.git
    images:
    - repoURL: nginx
      tag: 1.24.0
```

### Promotion Mechanisms

The `spec.promotionMechanisms` field is used to describe _how_ to move freight
into the `Stage`.

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

In the following example, the `test` `Stage` subscribes to manifests from a Git
repository _and_ images from an image repository, as in the previous section.
The example has now been amended to also show that transitioning freight into
the `test` `Stage` requires:

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
  subscriptions:
    repos:
      git:
      - repoURL: https://github.com/example/kargo-demo.git
        branch: main
      images:
      - repoURL: nginx
        semverConstraint: ^1.24.0
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

If you commit changes to the Git repository's `main` branch _or_ if a new
version of the `nginx` image were published to Docker Hub, these mechanisms
provide the recipe for applying those changes to our `test` `Stage`.

:::note
Promotion mechanisms describe _how_ to move freight into a `Stage`, but they say
nothing of _which_ piece of freight or _when_ to do this. Keep reading. These
will be covered in the next section.
:::

:::info
In the example above, you may have noticed the use of a stage-specific
branch in the Git repository. Since we _subscribe_ to the Git repository's
`main` branch, we could create an undesired loop in our automation if it also
_writes_ to that same branch. Combining manifests from `main` with the desired
images and then writing those changes to the `stages/test` branch (which the
corresponding Argo CD `Application` would reference as its `targetRevision`) is
a strategy to prevent such a loop from ever forming.
:::

The application of any `Stage` resource's promotion mechanisms transitions a
piece of freight into the `Stage` and updates the `Stage`'s `status` field
accordingly.

Continuing with our example, our `test` `Stage`'s `status` may appear as follows
after its first promotion:

```yaml
status:
  availableFreight:
  - id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    firstSeen: "2023-04-21T19:05:36Z"
    commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/example/kargo-demo.git
    images:
    - repoURL: nginx
      tag: 1.24.0
  currentFreight:
    id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    firstSeen: "2023-04-21T19:05:36Z"
    commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/example/kargo-demo.git
    images:
    - repoURL: nginx
      tag: 1.24.0
    health:
      status: Healthy
  history:
  - id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    firstSeen: "2023-04-21T19:05:36Z"
    commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/example/kargo-demo.git
    images:
    - repoURL: nginx
      tag: 1.24.0
    health:
      status: Healthy
```

Above, we can see that the piece of freight currently deployed to the `Stage` is
recorded in the `currentFreight` field. The `history` field duplicates this
information, but as the freight in a stage continues to change over time, each
new piece of freight will be _pushed_ onto the `history` collection, making that
field a historic record of of the freight that has moved through the `Stage`.

## `Promotion` resources

In the previous section, we discussed _how_ promotion mechanisms move freight
from one `Stage` to another, but we have not yet discussed what actually
triggers that process.

Kargo `Promotion` resources are used as _requests_ to progress a piece of
freight from one `Stage`to another.

`Promotion` resources may be created either automatically or manually, depending
on policies (covered in the next section).

Regardless of whether it is created manually or automatically, a `Promotion`
looks like this:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Promotion
metadata:
  name: test-to-404df86560cab5d515e7aa74653e665c1dc96e1c
  namespace: kargo-demo
spec:
  stage: test
  freight: 404df86560cab5d515e7aa74653e665c1dc96e1c
```

`Promotion` resources are simple. Their `spec.stage` and `spec.freight`
fields specify a `Stage` by name and one of its available pieces of freight,
which should be moved into that `Stage`.

:::info
The name in a `Promotion`'s `metadata.name` field is inconsequential. Only
the `spec` matters.
:::

When a `Promotion` has concluded -- whether successfully or unsuccessfully --
the `Promotion`'s `status` field is updated to reflect the outcome.

_So, who can create `Promotion` resources? And when does Kargo create them
automatically?_

## Creating `Promotion`s Manually

As with all resource types in Kubernetes, permissions to perform various actions
on `Promotion` resources, including creating new ones, are governed by
[RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

Kubernetes RBAC, alone, cannot address one particular concern, however. Often,
it is necessary to grant a user permission to create `Promotion` resources for
some particular `Stage`s but not for others. To address this, Kargo utilizes an
admission control webhook that conducts access reviews to determine if a user
creating a `Promotion` resource has the virtual `promote` verb for the `Stage`
referenced by the `Promotion` resource.

:::info
[This blog post](https://blog.aquasec.com/kubernetes-verbs) is an excellent
primer on virtual verbs in Kubernetes RBAC.
:::

The pre-defined `kargo-promoter` `ClusterRole` grants the ability to create,
read, update, delete, and list `Promotion` resources and also grants the virtual
`promote` ability for all `Stages`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kargo-promoter
  labels:
    # ...
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
- # ...
- apiGroups:
  - kargo.akuity.io
  resources:
  - stages
  verbs:
  - promote
```

To grant a fictional user `alice` the ability to create `Promotion`s for all
`Stage`s in a given namespace, such as `kargo-demo`, a `RoleBinding` (_not_ a
`ClusterRoleBinding`) such as the following may be created:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alice-promoter
  namespace: kargo-demo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kargo-promoter
subjects:
- kind: User
  name: alice
```

Suppose, however, that a fictional user `bob` should be permitted to create
`Promotion` resources that reference the `UAT` `Stage`, but not any other.
The following `Role` and `RoleBinding` would address that need:

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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: bob-uat-promoter
  namespace: kargo-demo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: uat-promoter
subjects:
- kind: User
  name: bob
```

## Auto-promotions

At times, it may be desirable for Kargo itself to create a new `Promotion`
resource to _automatically_ transition new freight into a certain `Stage`.

Enabling this requires the creation of a `PromotionPolicy` resource. The
following example demonstrates how a `test` `Stage` can take advantage of
auto-promotions:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionPolicy
metadata:
  name: test
  namespace: kargo-demo
stage: test
enableAutoPromotion: true
```

:::info
Why isn't `enableAutoPromotion` a field on the `Stage` resource type itself?

It is entirely plausible that the users with permission to define a `Stage`
aren't intended to have the authority to execute promotions _to_ that `Stage`.
If `enableAutoPromotion` were a field on the `Stage` resource type, then users
with permission to create and update `Stage`s could enable auto-promotion to
effect a promotion they themselves could not otherwise have performed manually.

By utilizing a separate `PromotionPolicy` resource to enable auto-promotion for
a given `Stage`, this would-be method of privilege escalation is eliminated.
:::
