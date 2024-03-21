---
sidebar_label: Key Concepts
description: Find out more about key Kargo concepts - stages, freight, warehouses, promotions, and more
---
# Key Kargo Concepts

## The Basics

### What is a Project

A **project** is a collection of related Kargo resources that describe one or
more delivery pipelines and is the basic unit of organization and tenancy in
Kargo.

RBAC rules are also defined at the project level and project administrators
may use projects to define policies, such as whether a **stage** is eligible
for automatic promotions of new **freight**.

### What is a Stage?

When you hear the term “environment”, what you envision will depend
significantly on your perspective. To eliminate confusion, Kargo avoids the term
"environment" altogether in favor of **stage**. The important feature of a stage
is that its name ("test" or "prod," for instance) denotes an application
instance's _purpose_ and not necessarily its _location_.
[This blog post](https://akuity.io/blog/kargo-stage-not-environment/) discusses
the rationale behind this choice.

_Stages are Kargo's most important concept._ They can be linked together in a
directed acyclic graph to describe a delivery pipeline. Typically, such a
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

## Corresponding Resource Types

Each of Kargo's fundamental concepts maps directly onto a custom Kubernetes
resource type.

### `Project` Resources

As of Kargo `v0.4.0`, each Kargo project is represented by a cluster-scoped
Kubernetes resource of type `Project`. Reconciliation of such a resource effects
all boilerplate project initialization, including the creation of a
specially-labeled `Namespace` with the same name as the `Project`. All resources
belonging to a given `Project` should be grouped together in that `Namespace`.

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

* In future releases, _additional_ boilerplate configuration will be created at
  the time of `Project` creation. This will include things such as project-level
  RBAC resources and `ServiceAccount` resources.
:::

:::info
In future releases, the team also expects to also aggregate project-level status
and statistics in `Project` resources.
:::

:::info
The `Project` resource expects to assume sole ownership of the `Namespace`
resource it represents.  However, there may be scenarios where shared ownership
is desired, such as another controller creating the namespace and maintaining
ownership.  To allow shared ownership between the other controller and
`Project`, add `kargo.akuity.io/allow-shared-ownership: "true"` to the
`Namespace` labels.
:::

#### Promotion Policies

A `Project` resource can additionally define project-level configuration. At
present, this only includes **promotion policies** that describe which `Stage`s
are eligible for automatic promotion of newly qualified `Freight`.

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
promotion of newly qualified `Freight`, but any other `Stage`s in the `Project`
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

### `Stage` Resources

Each Kargo stage is represented by a Kubernetes resource of type `Stage`.

A `Stage` resource's `spec` field decomposes into three main areas of concern:

* Subscriptions

* Promotion mechanisms

* Verification

The following sections will explore each of these in greater detail.

#### Subscriptions

The `spec.subscriptions` field is used to describe the sources from which a
`Stage` obtains `Freight`. These subscriptions can be to a single `Warehouse` or
to one or more "upstream" `Stage` resources.

For each `Stage`, the Kargo controller will periodically check for `Freight`
resources that are newly qualified for promotion to that `Stage`.

For any `Stage` subscribed directly to a `Warehouse`, _any_ new `Freight`
resource from that `Warehouse` is tacitly consider to have been _verified_
upstream, and is therefore immediately qualified for promotion to such a
`Stage`.

For a `Stage` subscribed to one or more "upstream" `Stage` resources, `Freight`
is qualified for promotion to that `Stage` after being _verified_ in at least
one of the upstream `Stage`s. Alternatively, users with adequate permissions may
manually _approve_ `Freight` for promotion to any given `Stage` without
requiring upstream verification.

:::tip
Explicit approvals are a useful method for applying the occasional "hotfix"
without waiting for a `Freight` resource to traverse the entirety of a pipeline.
:::

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

#### Verifications

The `spec.verification` field is used to describe optional verification
processes that should be executed after a `Promotion` has successfully deployed
`Freight` to a `Stage`, and if applicable, after the `Stage` has reached a
healthy state.

Verification processes are defined through _references_ to one or more 
[Argo Rollouts `AnalysisTemplate` resources](https://argoproj.github.io/argo-rollouts/features/analysis/)
that reside in the same `Project`/`Namespace` as the `Stage` resource.

:::info
Argo Rollouts `AnalysisTemplate` resources (and the `AnalysisRun` resources that
are spawned from them) were intentionally built to be re-usable in contexts
other than Argo Rollouts. Re-using this resource type to define verification
processes means those processes benefit from this rich and battle-tested feature
of Argo Rollouts.
:::

The following example depicts a `Stage` resource that references an
`AnalysisTemplate` named `kargo-demo` to validate the `test` `Stage` after any
successful `Promotion`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
  verification:
    analysisTemplates:
    - name: kargo-demo
```

It is also possible to specify additional labels, annotations, and arguments
that should be applied to `AnalysisRun` resources spawned from the referenced
`AnalysisTemplate`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  # ...
  verification:
    analysisTemplates:
    - name: kargo-demo
    analysisRunMetadata:
      labels:
        foo: bar
      annotations:
        bat: baz
    args:
    - name: foo
      value: bar
```

An `AnalysisTemplate` could be as simple as the following, which merely executes
a Kubernetes `Job` that is defined inline:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  metrics:
  - name: test
    provider:
      job:
        metadata:
        spec:
          backoffLimit: 1
          template:
            spec:
              containers:
              - name: test
                image: alpine:latest
                command:
                - sleep
                - "10"
              restartPolicy: Never
```

:::note
Please consult the
[relevant sections](https://argoproj.github.io/argo-rollouts/features/analysis/)
of the Argo Rollouts documentation for comprehensive coverage of the full range
of `AnalysisTemplate` capabilities.
:::

#### Status

A `Stage` resource's `status` field records:

* The current phase of the `Stage` resource's lifecycle.

* Information about any in-progress `Promotion`.

* The `Freight` currently deployed to the `Stage`.

* History of `Freight` that has been deployed to the `Stage`. (From most to
  least recent.)

* The health status any any associated Argo CD `Application` resources.

* The status of any in-progress of completed verification processes.

For example:

```yaml
status:
  phase: Steady
  currentFreight:
    id: 47b33c0c92b54439e5eb7fb80ecc83f8626fe390
    images:
    - repoURL: nginx
      tag: 1.25.3
    commits:
    - repoURL: https://github.com/example/kargo-demo.git
      id: 1234abc
    verificationResult:
      analysisRun:
        namespace: kargo-demo
        name: test.ab85b188-0ad5-43d9-a36d-ddcf63666183.47b33c0
        phase: Successful
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
    verificationResult:
      analysisRun:
        namespace: kargo-demo
        name: test.ab85b188-0ad5-43d9-a36d-ddcf63666183.47b33c0
        phase: Successful
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
which the `Freight` has been _verified_ and a separate list of `Stage` resources
for which the `Freight` has been manually _approved_.

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
  verifiedIn:
    test: {}
  approvedFor:
    prod: {}
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
:::info
Kargo uses [semver](https://github.com/masterminds/semver#checking-version-constraints) to handle semantic versioning constraints.
:::

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
