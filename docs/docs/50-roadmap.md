---
description: Roadmap
---

# Roadmap

:::caution
K8sTA is highly experimental at this time and breaking changes should be
anticipated between pre-GA minor releases.
:::

The current goal with K8sTA is to discover and add value in small increments. If
the road ahead were more clear, we'd document it here -- and someday we will. In
the meantime, we'll document the road _behind us_ and _currently under our
tires_ to highlight what we've already experimented with, or are experimenting
with currently, what has worked, and what hasn't.

## Iteration 1

In the first iteration, our goal was only to move a new Docker image through a
series of environments _with no manual intervention and no "glue code"
required._

We introduced a CRD, `Track`, instances of which subscribe to new images pushed
to different image repositories (in Docker Hub only for now). Instances of a
`Track` also define an ordered list of environments to progress new images
through. In this initial iteration, an environment is just a reference to an
existing Argo CD `Application`.

When an inbound webhook from Docker Hub is received by the server component and
indicates a new image is to be progressed along a subscribed `Track`, an
instance of a `Ticket` CRD is created. A `TicketReconciler` in the controller
component manages progressive deployment of the image along the `Track`.

Argo CD operates 100% as normal and K8sTA is purely complementary.

The results of the first iteration were successfully demoed on 2022-08-03 and
a recording of that demo is available to Akuity employees
[here](https://drive.google.com/file/d/1HfAaS9tky3QVof9xTvYugr55CwIhCOSJ/view?usp=sharing).

## Iteration 2

Our goal for the second iteration were to further explore what a reasonable API
for K8sTA looks like, with no expectation of immediately implementing all
aspects of that API. The effort to do so was a great success in that it
uncovered all of the following:

* K8sTA will likely need to assimilate much of the functionality of the
  [Argo CD Image Updater](https://argocd-image-updater.readthedocs.io/en/stable/)
  -- namely its ability to subscribe to an image repository and be notified of
  new images meeting user-defined criteria. Long term, this will mean K8sTA will
  not be dependent on webhooks, although the K8sTA prototype _does_ remain
  dependent on webhooks at this time. Fields that permit users to define those
  subscriptions have already been added to the `Track` API for illustrative
  purposes, but are mostly unused at this time.

* Stations on a `Track` can now reference _multiple_ Argo CD `Application`
  resources. By doing so, it is possible to roll changes out to multiple
  environments at once. Progressing farther down the `Track` requires _all_ of
  those multiple `Application`s to sync and reach a heathy state. This is
  useful (for instance) if one wished to treat `Application`s managing
  deployment to different availability zones as a single logical environment,
  deploying to all simultaneously, and weighing the success or failure of all
  as an atomic unit. This feature is fully-integrated into the prototype, but
  isn't well-covered by tests.

* Stations on a `Track` can now behave as "junctions" by referencing one or more
  other `Track`s. When the change represented by a `Ticket` is progressed to
  such a station, it results in the creation of a new `Ticket` per referenced
  `Track` to progress the same change, independently, along each. This
  capability makes it possible to compose complex, tree-like tracks from many
  sections of simple, linear `Track`. This is useful (for instance) if one
  wished to progress changes through a series of "preliminary" environments like
  "dev" and "int" before _independently_ progressing the change through multiple
  environments in different geographic regions or even different clouds. This
  feature is fully-integrated into the prototype, but isn't well-covered by
  tests.

* Some work was performed to allow for user-defined "quality gates" in the
  `Track` API. Such gates would have either permitted or halted progress to the
  next station after every `Application` referenced by the previous station was
  synced and healthy. This was abandoned after @jessesuen and @alexmt pointed
  out that [Argo Rollouts](https://argoproj.github.io/argo-rollouts/) already
  contained similar functionality, which turned out to be a pivotal moment in
  understanding where K8sTA is headed.

  Because Argo CD already integrates so well with Argo Rollouts, a single Argo
  CD `Application`'s health can already be pegged to the success or failure of
  user-defined tests, and because K8sTA already treats `Application` health as a
  criterion for progressing a change farther down a `Track`, K8sTA gets
  "quality gate" functionality for free. Or more precisely, the stack of K8sTA +
  Argo CD + Argo Rollouts has this capability. _This was a major revelation._

  K8sSTA is but the top tier of the <b><u>KCR</u></b> (<b><u>K</u></b>8sTA +
  Argo <b><u>C</u></b>D + Argo <b><u>R</u></b>ollouts) stack and only needs to
  provide the little functionality those other tiers lack -- namely it needs
  only to understand the relationship between environments and orchestrate
  deployments across them while relying on the lower tiers of the stack to
  continue doing what they already so remarkably well. This seems an eminently
  practical proposition since, at Akuity, we have the luxury of knowing that
  every one of our customers is already using Argo CD, and given Argo CD's
  excellent integration with Argo Rollouts, many Akuity customers are likely
  using it already or could easily be convinced to do so.

## Iteration 3

The first two iterations focused _exclusively_ on the use case of progressing a
new image along a `Track` (or series of interconnected `Track`s), with the
precipitating event being the publication (push) of that image.

Iteration 3 explored _another type of change_ that needs to be progressed
along a `Track` (or series of interconnected `Track`s), with a _different
precipitating event_ -- namely, changes that have been committed to the default
(e.g. `main`) branch of a GitOps repository by a human operator.

The K8sTA controller now monitors the default branches of repositories
referenced by K8sTA `Track` resources for new commits. When such changes
detected _and_ K8sTA determined to likely impact multiple Argo CD `Application`s
(because files under `base/` were added, modified, or deleted), K8sTA creates a
`Ticket` representing the changes.

Changes that are deemed to affect only specific overlays / Argo CD
`Application`s (because _no_ files under `base/` were added, modified, or
deleted) are, for now, ignored, and will likely be the subject of some future
iteration.

This iteration also was partly interrupted by Akuity Platform launch and
resources were, largely, reallocated to creating and editing Akuity Platform
documentation. Owing to this, any concurrent K8sTA work was limited to "low
hanging fruit," but several noteworthy advancements were made despite the simple
nature of the work:

* Ability to disable entire `Track`s or constituent elements such as `Station`s
  or Argo CD `Application`. This is useful when wishing to temporarily bypass
  some element without permanently removing it.

* Support Argo CD `Application`s in any namespace. Argo CD itself supports this
  as of just recently, which invalidated our previous assumption that all
  `Application` resources existed in the same namespace as Argo CD itself.

* Support for additional configuration management tools:

  * ytt
  * Helm

* Automated CI and release processes implemented.
    * Images are signed during the release process.
    * SBOMs are published during the release process.

* Docs were created, including a quickstart.

This iteration also included our first release -- v0.1.0-alpha.1, which permits
Akuity staff who are not actively contributing to K8sTA to test drive K8sTA
without the need to build it from source.

## Iteration 4

This iteration aligns with the start of Akuity dev cycle 2.

Iteration 4 will focus on decomposing K8sTA's two main responsibilities into
separate components that can (potentially) be used independently of one another.
By doing this, we hope to begin dogfooding and gaining value from the more
stable and straightforward elements of K8sTA _sooner_ while work on the more
novel and experimental elements continues at its own pace.

The responsibilities in question are:

1. GitOps "bookkeeping": These are the tedious operations required for
   implementing GitOps with Argo CD and the "rendered YAML branches" pattern.
   
   This includes:

     * Integrating with configuration management tools to render plain YAML.
     * Committing or PR'ing changes to `Application`-specific branches.

   Overall, these functions are well understood and comparatively easy to
   implement.

2. Progressive delivery: This includes detecting changes and progressing them
   through the `Station`s enumerated by a `Track`.

   Overall, this is more novel and experimental.

Iteration 4 will attempt to incorporate all logic related to no. 1 above into a
standalone service. We hope that by the end of the iteration, we will be able to
leverage that service to eliminate a substantial amount of bespoke "glue code"
from some of AKuity's own internal processes.
