---
sidebar_label: Key Concepts
description: Find out more about key Kargo concepts - stages, freight, warehouses, promotions, and more
---
# Key Kargo Concepts

:::note
Documentation is Under Active Development

We're currently reorganizing and updating the documentation. During this process, you may encounter sections that are incomplete, repetitive, or fragmented. Please bear with us as we work to make improvements.
:::

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

#### Promotion Policies

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

### `Stage` Resources

Each Kargo stage is represented by a Kubernetes resource of type `Stage`.

A `Stage` resource's `spec` field decomposes into three main areas of concern:

* Requested freight

* Promotion template

* Verification

The following sections will explore each of these in greater detail.

#### Requested Freight

The `spec.requestedFreight` field is used to describe one or more "types" of
`Freight`, as specified by an `origin`, that the `Stage`'s promotion process, as
specified by `spec.promotionTemplate`, will operate on, and the acceptable
sources from which to obtain that `Freight`. Those sources may include the
origin itself (e.g. a `Warehouse`) and/or any number of "upstream" `Stage`
resources.

:::info
`Warehouse`s are the only type of origin at present, but it is anticipated that
future versions of Kargo will introduce additional origin types. This is why
"types" of `Freight` are described by an `origin` field having `kind` and `name`
subfields instead of being described only by the name of a `Warehouse`.
:::

For each `Stage`, the Kargo controller will periodically check for `Freight`
resources that are newly available for promotion to that `Stage`.

When a `Stage` accepts `Freight` directly from its origin, _all_ new `Freight`
created by that origin (e.g. a `Warehouse` ) are immediately available for
promotion to that `Stage`.

When a `Stage` accepts `Freight` from one or more "upstream" `Stage` resources,
`Freight` is considered available for promotion to that `Stage` only after being
_verified_ in at least one of the upstream `Stage`s. Alternatively, users with
adequate permissions may manually _approve_ `Freight` for promotion to any given
`Stage` without requiring upstream verification.

:::tip
Explicit approvals are a useful method for applying the occasional "hotfix"
without waiting for a `Freight` resource to traverse the entirety of a pipeline.
:::

In the following example, the `test` `Stage` requests `Freight` that has
originated from the `my-warehouse` `Warehouse` and indicates that it will accept
new `Freight` _directly_ from that origin:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  requestedFreight:
  - origin:
      kind: Warehouse
      name: my-warehouse
    sources:
      direct: true
  # ...
```

In this example, the `uat` `Stage` requests `Freight` that has originated from
the `my-warehouse` `Warehouse`, but indicates that it will accept such `Freight`
only after it has been _verified_ in the `test` `Stage`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: uat
  namespace: kargo-demo
spec:
  requestedFreight:
  - origin:
      kind: Warehouse
      name: my-warehouse
    sources:
      stages:
      - test
  # ...
```

Stages may also request `Freight` from multiple sources. The following example
illustrates a `Stage` that requests `Freight` from both a `microservice-a` and
`microservice-b` `Warehouse`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-demo
spec:
  requestedFreight:
  - origin:
      kind: Warehouse
      name: microservice-a
    sources:
      direct: true
  - origin:
      kind: Warehouse
      name: microservice-b
    sources:
      direct: true
  # ...
```

:::tip
By requesting `Freight` from multiple sources, a `Stage` can effectively
participate in _multiple pipelines_ that may each deliver different collections
of artifacts independently of the others. At present, this is most useful for
the delivery of microservices that are developed and deployed in parallel,
although other uses of this feature are anticipated in the future.
:::

#### Promotion Templates

The `spec.promotionTemplate` field is used to describe _how_ to transition
`Freight` into the `Stage`. The `spec.promotionTemplate.steps` field describes
the discrete steps of a promotion process in detail.

In the following, very common example, the `promotionTemplate` describes steps
to:

1. Clone a Git repository containing Kubernetes manifests and Kustomize
   configuration, checking out two different branches to two different
   directories.

1. Clears the contents of one working tree, with intentions to fully replace its
   contents.

1. Runs the equivalent of `kustomize edit set image` to update a
   `kustomization.yaml` file with a reference to an updated
   `public.ecr.aws/nginx/nginx` container image.

1. Renders the updated manifests using the equivalent of `kustomize build`.

1. Commits the updated manifests and pushes them to the `stage/test` of the
   remote repository.

1. Forces Argo CD to sync the `kargo-demo-test` application to the latest commit
   of the `stage/test` branch.

```yaml
promotionTemplate:
  spec:
    steps:
    - uses: git-clone
      config:
        repoURL: https://github.com/example/repo.git
        checkout:
        - fromFreight: true
          path: ./src
        - branch: stage/test
          create: true
          path: ./out
    - uses: git-clear
      config:
        path: ./out
    - uses: kustomize-set-image
      as: update-image
      config:
        path: ./src/base
        images:
        - image: public.ecr.aws/nginx/nginx
    - uses: kustomize-build
      config:
        path: ./src/stages/test
        outPath: ./out
    - uses: git-commit
      as: commit
      config:
        path: ./out
        messageFromSteps:
        - update-image
    - uses: git-push
      config:
        path: ./out
        targetBranch: stage/test
    - uses: argocd-update
      config:
        apps:
        - name: kargo-demo-test
          sources:
          - repoURL: https://github.com/example/repo.git
            desiredCommitFromStep: commit
```

__For complete documentation of all of Kargo's built-in promotion steps, refer
to the [Promotion Steps Reference](./35-references/10-promotion-steps.md).__

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

* Information about the last `Promotion` and any in-progress `Promotion`.

* History of `Freight` that has been deployed to the `Stage` (from most to
  least recent) along with the results of any associated verification processes.

* The health status of any associated Argo CD `Application` resources.

For example:

```yaml
status:
  freightHistory:
  - id: 101bca5b0e18ca7913978a1da956308d2544f741
    items:
      Warehouse/my-warehouse:
        commits:
        - healthCheckCommit: 111eaf55aa41f21bb9bb707ba1baa748b83ec51e
          id: 961cfaedbc53aacdb65110028839a2c1c281290d
          repoURL: https://github.com/example/kargo-demo.git
        images:
        - digest: sha256:b2487a28589657b318e0d63110056e11564e73b9fd3ec4c4afba5542f9d07d46
          repoURL: public.ecr.aws/nginx/nginx
          tag: 1.27.0
        name: 666209fd9755a1e48bec6b27f5f447747410dd9e
        origin:
          kind: Warehouse
          name: my-warehouse
    verificationHistory:
    - analysisRun:
        name: test.01j2w7aknhf3j7jteyqs72hnbg.101bca5
        namespace: kargo-demo-09
        phase: Successful
      finishTime: "2024-07-15T22:13:57Z"
      id: 5535a484-bbd0-4f12-8cf4-be2c8e0041c9
      phase: Successful
      startTime: "2024-07-15T22:13:34Z"
  health:
    argoCDApps:
    - healthStatus:
        status: Healthy
      name: kargo-demo-09-test
      namespace: argocd
      syncStatus:
        revision: 111eaf55aa41f21bb9bb707ba1baa748b83ec51e
        status: Synced
    status: Healthy
  lastPromotion:
    finishedAt: "2024-07-15T22:13:25Z"
    freight:
      commits:
      - healthCheckCommit: 111eaf55aa41f21bb9bb707ba1baa748b83ec51e
        id: 961cfaedbc53aacdb65110028839a2c1c281290d
        repoURL: https://github.com/example/kargo-demo.git
      name: 666209fd9755a1e48bec6b27f5f447747410dd9e
      origin:
        kind: Warehouse
        name: kargo-demo
    name: test.01j2w7a15cxjjgejresfyw6ysp.666209f
    status:
      finishedAt: "2024-07-15T22:13:25Z"
      freight:
        commits:
        - healthCheckCommit: 111eaf55aa41f21bb9bb707ba1baa748b83ec51e
          id: 961cfaedbc53aacdb65110028839a2c1c281290d
          repoURL: https://github.com/example/kargo-demo.git
        name: 666209fd9755a1e48bec6b27f5f447747410dd9e
        origin:
          kind: Warehouse
          name: kargo-demo
      freightCollection:
        id: 101bca5b0e18ca7913978a1da956308d2544f741
        items:
          Warehouse/kargo-demo:
            commits:
            - healthCheckCommit: 111eaf55aa41f21bb9bb707ba1baa748b83ec51e
              id: 961cfaedbc53aacdb65110028839a2c1c281290d
              repoURL: https://github.com/example/kargo-demo.git
            name: 666209fd9755a1e48bec6b27f5f447747410dd9e
            origin:
              kind: Warehouse
              name: kargo-demo
        verificationHistory:
        - analysisRun:
            name: test.01j2w7aknhf3j7jteyqs72hnbg.101bca5
            namespace: kargo-demo-09
            phase: ""
          id: 5535a484-bbd0-4f12-8cf4-be2c8e0041c9
          phase: Pending
          startTime: "2024-07-15T22:13:34Z"
      phase: Succeeded
  observedGeneration: 1
  phase: Steady
```

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
      repoURL: public.ecr.aws/nginx/nginx
      semverConstraint: ^1.26.0
  - git:
      repoURL: https://github.com/example/kargo-demo.git
```
:::info
Kargo uses [semver](https://github.com/masterminds/semver#checking-version-constraints) to handle semantic versioning constraints.
:::

#### Git Subscription Path Filtering

In some cases, it may be necessary to constrain the paths within a Git
repository that a `Warehouse` will consider as triggers for `Freight`
production. This is especially useful for GitOps repositories that are
"monorepos" containing configuration for multiple applications.

The paths that may or must not trigger `Freight` production may be specified
using a combination of the `includePaths` and `excludePaths` fields of a Git
repository subscription.

The following example demonstrates a `Warehouse` with a Git repository
subscription that will only produce new `Freight` when the latest commit
(selected by the applicable commit selection strategy) contains changes in the
`apps/guestbook` directory since the last piece of `Freight` produced by the
`Warehouse`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/kargo-demo.git
      includePaths:
      - apps/guestbook
```

The next example demonstrates the opposite: a `Warehouse` with a Git repository
subscription that will only produce new `Freight` when the latest commit
(selected by the applicable commit selection strategy) contains changes to paths
_other than_ the repository's `docs/` directory:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
    subscriptions:
    - git:
        repoURL: https://github.com/example/kargo-demo.git
      excludePaths:
      - docs
```

`includePaths` and `excludePaths` may be combined to include a broad set of
paths and then exclude a subset of those. The following example demonstrates a
`Warehouse` with a Git repository subscription that will only produce new
`Freight` when the latest commit (selected by the applicable commit selection
strategy) contains changes _within_ the `apps/guestbook` directory _other than_
the `apps/guestbook/README.md`:

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Warehouse
metadata:
  name: my-warehouse
  namespace: kargo-demo
spec:
  subscriptions:
  - git:
      repoURL: https://github.com/example/kargo-demo.git
      includePaths:
      - apps/guestbook
      excludePaths:
      - apps/guestbook/README.md
```

:::note
It is important to understand that new `Freight` will be produced when the
latest commit (selected by the applicable commit selection strategy) contains
_even a single change_ that is:

1. Implicitly included via undefined `includePaths`.

   &nbsp;&nbsp;&nbsp;&nbsp;OR

   Explicitly included via `includePaths`.

   AND

2. Not explicitly excluded via `excludePaths`.
:::

:::info
By default, the strings in the `includePaths` and `excludePaths` fields are
treated as exact paths to files or directories. (Selecting a directory will
implicitly select all paths within that directory.)

Paths may _also_ be specified using glob patterns (by prefixing the string with
`glob:`) or regular expressions (by prefixing the string with `regex:` or
`regexp:`).
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
