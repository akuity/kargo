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

Iteration 3 will explore _another type of change_ that needs to be progressed
along a `Track` (or series of interconnected `Track`s), with a _different
precipitating event_ -- namely, changes that have been committed to the "source
branch" by a human operator.

:::note
To clarify the meaning of "source branch": K8sTA heavily leans into the
"rendered YAML branches pattern" wherein a single branch (typically `main`)
houses base configuration as well as environment-specific overlays. The base
configuration and applicable overlays can be rendered into plain YAML manifests
that are stored in environment-specific branches. I have been calling the branch
containing base configuration and overlays the "source branch" to distinguish it
from the environment-specific branches.
:::

Iteration 3 will focus on detecting changes to the source branch (that were not
initiated by K8sTA itself as part of the image update process already
implemented), determining whether those changes affect base configuration (and
therefore, potentially, all environments) and in such a case, proceeding to
progress those changes down applicable `Track`s.

Further:

* Changes that are deemed to affect only specific overlays / environments are,
  for now, out of scope (although I anticipate these being addressed in the
  next iteration).

* K8sTA should take responsibility for rendering plain YAML manifests into the
  environment-specific branches so that it is neither the responsibility of a
  human operator to do so manually nor is it their responsibility to automate
  the process themselves through their own esoteric glue code or automated
  processes. (i.e. A human operator should only ever touch the source branch.)

* For the sake of expediency, as with detecting new Docker images, the prototype
  will rely on webhooks for detecting changes in a git repository which, in the
  future, should be detected through other means.
