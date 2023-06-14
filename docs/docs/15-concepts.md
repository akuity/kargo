---
title: Concepts
description: Concepts
---

This section covers important Kargo concepts.

## What is an environment?

_And what is an `Environment`?_

In general, the word "environment" is poorly defined and severely overloaded. To
some, an "environment" may be a particular Kubernetes cluster or namespace
therein hosting a number of applications. To others, an "environment" could be a
particular instance of one application -- or multiple applications deployed as a
single unit. It could be an entire failure domain.

Kargo is un-opinionated and unconcerned with what "environment" means to you.
However _you_ define "environment," Kargo's `Environment` custom resource type
helps you describe _how_ changes to Kubernetes manifests, new versions of
Docker images, or even new versions of Helm charts can be rolled out in a
controlled and progressive fashion from one "environment" to the next. The
transitions from environment-to-environment can be automated or manually
triggered, as called for by your use cases or preferences.

:::info
As a matter of convention, throughout this documentation, we are careful to use
`Environment` (capitalized and monospaced) when we're referring specifically to
that custom resource type and "environment" (with or without quotes) when
referring to the considerably more vague concept.
:::

## `Environment` resources

Like many Kubernetes resource types, an `Environment` resource is decomposed
into three main sections:

* `metadata` that describes the resource's identifying information, such as
  names, namespace, labels, etc.

  :::info
  It is a suggested practice to co-locate related `Environment` resources in
  a single, dedicated namespace. In our examples, we group our `Environment`
  resources together in a `kargo-demo` namespace.
  :::

* `spec` that encapsulates the user-defined particulars of each resource.

* `status` that encapsulates resource state.

At this highest level, a manifest describing an `Environment` resource may
appear as follows:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Environment
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
status:
  # ...
```

An `Environment` resource's `spec` field further decomposes into two main areas
of concern:

* Subscriptions

* Promotion mechanisms

The following sections will explore each of these in greater detail.

### Subscriptions

The `spec.subscriptions` field is used to describe the sources from which an
`Environment` obtains *materials*. Materials include any combination of the
following:

* Manifests from a Git repository. These can be plain YAML or rendered with the
  assistance of configuration management tools like
  [Kustomize](https://kustomize.io/) or [Helm](https://helm.sh/).

* Docker images from an image repository.

* Helm charts from a chart repository.

Alternatively, instead of subscribing directly to repositories, an `Environment`
may subscribe to another, "upstream" `Environment`.

For each `Environment`, the Kargo controller will periodically check all
subscriptions for the latest available materials. A single set of materials is
known, internally, as a *state*. The controller produces a canonical
representation of each state and uses that to calculate a SHA1 hash which
becomes the state's ID. Because state IDs are calculated deterministically from
the underlying materials, each state ID is also a fingerprint of sorts. This ID
is compared to those in a stack of known, *available states* stored in the
`Environment` resource's `status` field. If a state is new, it is pushed onto
the stack and becomes *available.*

In the following example, the `Environment` subscribes to manifests from a Git
repository _and_ images from an image repository:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Environment
metadata:
  name: test
  namespace: kargo-demo
spec:
  subscriptions:
    repos:
      git:
      - repoURL: https://github.com/example/kargo-demo-gitops.git
        branch: main
      images:
      - repoURL: nginx
        semverConstraint: ^1.24.0
  # ...
```

This is how the `test` `Environment` resource's `status` field may appear after
polling the two repositories to which it subscribes:

```yaml
status:
  availableStates:
  - commits:
    - id: dd8dc6a021d9d6c42e937f8b8f221a838342ec2a
      repoURL: https://github.com/example/kargo-demo-gitops.git
    firstSeen: "2023-04-21T18:34:56Z"
    id: 51636b9332d5938b9f2d382e9713b54ceb62a323
    images:
    - repoURL: nginx
      tag: 1.24.0
```

### Promotion mechanisms

The `spec.promotionMechanisms` field is used to describe how to transition an
environment into a new state.

There are two general methods of accomplishing this:

* Committing changes to a Git repository.

* Making changes to an Argo CD `Application` resource.

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
  automatic for any `Application` resource the `Environment` interacts with.)

:::info
Additionally, interaction with any Argo CD `Application` resources(s) as
described above implicitly results in periodic evaluation of `Environment`
health by aggregating the results of sync/health state for all such
`Application` resources(s).
:::

In the following example, the `Environment` subscribes to manifests from a Git
repository _and_ images from an image repository, as in the previous section.
Our example also now states that transitioning the environment to a new state
requires:

1. Updating the `https://github.com/example/kargo-demo-gitops.git` repository by
   running `kustomize edit set image` in the `env/test` directory and committing
   those changes to an environment-specific `env/test` branch.

1. Forcing the Argo CD `Application` named `kargo-demo-test` in the `argocd`
   namespace to refresh and sync.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Environment
metadata:
  name: test
  namespace: kargo-demo
spec:
  subscriptions:
    repos:
      git:
      - repoURL: https://github.com/example/kargo-demo-gitops.git
        branch: main
      images:
      - repoURL: nginx
        semverConstraint: ^1.24.0
  promotionMechanisms:
    gitRepoUpdates:
    - repoURL: https://github.com/example/kargo-demo-gitops.git
      writeBranch: env/test
      kustomize:
        images:
        - image: nginx
          path: env/test
    argoCDAppUpdates:
    - appName: kargo-demo-test
      appNamespace: argocd
```

If you commit changes to the Git repository's `main` branch _or_ if a new
version of the Nginx image were published to Docker Hub, these mechanisms
provide the recipe for transitioning those changes into our test environment.

:::note
Promotion mechanisms describe _how_ to transition an `Environment` into a new
state, but they say nothing of _which_ state or _when_ to make the transition.
Keep reading. These will be covered in the next section.
:::

:::info
In the example above, you may have noticed the use of an environment-specific
branch in the Git repository. Since we _subscribe_ to the Git repository's
`main` branch, we could create an undesired loop in our automation if it also
_writes_ to that same branch. Combining manifests from `main` with the desired
images and then writing those changes to `env/test` branch (which the
corresponding Argo CD `Application` would reference as its `targetRevision`) is
a strategy to prevent such a loop from ever forming.
:::

The application of any `Environment` resource's promotion mechanisms transitions
the `Environment` into a new state and updates the `Environment`'s `status`
field accordingly.

Continuing with our example, our `test` `Environment`'s `status` may appear as
follows after its first promotion:

```yaml
status:
  availableStates:
  - commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/krancour/kargo-demo-gitops-2.git
    firstSeen: "2023-04-21T19:05:36Z"
    id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    images:
    - repoURL: nginx
      tag: 1.24.0
  currentState:
    commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/krancour/kargo-demo-gitops-2.git
    firstSeen: "2023-04-21T19:05:36Z"
    id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    images:
    - repoURL: nginx
      tag: 1.24.0
    health:
      status: Healthy
  history:
  - commits:
    - id: 02d153f75e5c042d576c713be52b57e1db8ddc97
      repoURL: https://github.com/krancour/kargo-demo-gitops-2.git
    firstSeen: "2023-04-21T19:05:36Z"
    id: 404df86560cab5d515e7aa74653e665c1dc96e1c
    images:
    - repoURL: nginx
      tag: 1.24.0
    health:
      status: Healthy
```

Above, we can see that the state currently deployed to the environment is
recorded in the `currentState` field. The `history` field duplicates this
information, but as state continues to change over time, each new state will be
_pushed_ onto the `history` collection, making that field a historic record of
what's been deployed to the environment.

## `Promotion` resources

In the previous section, we discussed _how_ promotion mechanisms transition
`Environment`s from one state to another, but we have not yet discussed what
actually triggers that process.

Kargo `Promotion` resources are used as _requests_ to transition an
`Environment` from its current state to any of its available states.

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
  environment: test
  state: 404df86560cab5d515e7aa74653e665c1dc96e1c
```

`Promotion` resources are simple. Their `spec.environment` and `spec.state`
fields specify an `Environment` by name and one of its available states, into
which that `Environment` should be transitioned.

:::info
While the name in a `Promotion`'s `metadata.name` field is inconsequential (only
the `spec` matters), it is recommended that they be named using the following
pattern: `<environment>-to-<state>`.

This is the same naming convention that the Kargo controller itself will observe
in cases where it does create `Promotion` resources automatically.
:::

When the state transition specified by a `Promotion` has concluded -- whether
successfully or unsuccessfully -- the `Promotion`'s `state` field is updated
to reflect the outcome.

_So, who can create `Promotion` resources? And when does Kargo cerate them
automatically?_

## `PromotionPolicy` resources

`PromotionPolicy` resources describe who may create `Promotion` resources for
each `Environment`s. They also specify whether the Kargo controller may
automatically create a `Promotion` resource when the `Environment`
reconciliation loop discovers a new available state.

For example:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: PromotionPolicy
metadata:
  name: test
  namespace: kargo-demo
environment: test
authorizedPromoters:
- subjectType: User
  name: kubernetes-admin
enableAutoPromotion: true
```

The above example indicates that authenticated users of the Kubernetes cluster
identified by username `kubernetes-admin` may create `Promotion` resources
referencing the `test` `Environment`. It also specifies that auto-promotion is
enabled -- meaning that the Kargo controller will automatically create a
`Promotion` resource to transition the `test` `Environment` into any newly
discovered state.

:::note
Authorized promoters do not need to be identified by username. There is also
support for identifying authorized `ServiceAccount`s, and human users and 
`ServiceAccount`s alike can both be authorized indirectly through bindings to
a specific role or membership in s specific group.
:::

:::info
_What about Kubernetes RBAC?_

Kubernetes RBAC works for Kargo resource types, of course, however, Kubernetes
RBAC is only sophisticated enough to establish who may or may not create
`Promotion` resources (or `Promotion` resources in a particular namespace).

With Kargo, it is likely that a single Kubernetes namespace may contain multiple
`Environment` resources. It is also likely that not all such resources are
treated with equal degrees of rigor. For instance, it may be permissible for any
developer on one's team to manually promote to a `test` or `stage` environment,
however, authority to promote to `prod` might be vested only in the team lead.

`PromotionPolicy` resources, therefore, permit someone such as a team lead to,
for instance, opt-in to auto-promotions for the `test` `Environment` and permit
any developer to promote manually to the `stage` `Environment` while reserving
the power to promote to the `prod` `Environment` for themselves.
:::

:::note
To be effective, the ability to create, edit, and delete `PromotionPolicy`
resources should be restricted to the same set of users who are authorized to
promote to production. Doing this precludes the possibility of a users _not_
authorized to promote to some environment(s) from creating or editing
`PromotionPolicy` resources in a manner that elevates their own privileges.
:::

:::info
When installed to your Kubernetes cluster via its official Helm chart, Kargo
includes three `ClusterRoleBinding` resources:

* `kargo-admin`: Can list, create, read, update, and delete all Kargo resource
  types.

* `kargo-developer`: Can list, create, read, update, and delete Kargo
  `Environment` resources. Can list and read `Promotion` and `PromotionPolicy`
  resources.

* `kargo-promoter`: Can list, create, read, update, and delete Kargo `Promotion`
  resources. Can list and read `Environment` and `PromotionPolicy` resources.

It is recommended that applicable users, `ServiceAccount`s, groups, etc. be
bound to these `ClusterRoles` on a namespace-by-namespace basis. (Kubernetes
does permit namespace-scoped `RoleBinding`s to non-namespaced `ClusterRoles`).
:::
