# K8sTA -- The Kubernetes Transit Authority

![CI](https://github.com/akuityio/k8sta-prototype/actions/workflows/ci.yaml/badge.svg)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

K8sTA is a _prototype_ progressive delivery tool built on top of Argo CD for the
purpose of coordinating the rollout of changes through a series of environments,
ending with the change reaching production.

> 🟡&nbsp;&nbsp;K8sTA is highly experimental at this time and breaking changes
> should be anticipated between pre-GA minor releases.

## The Tracks & Stations Paradigm

The term "environment" is dangerously overloaded -- meaning different things to
different people. Suppose, for instance, that "production" is sharded into
discrete application instances in multiple geographic regions. While some may
argue that "production," in its entirety, is a single "environment," others may
regard individual shards in "prod-us-east" and "prod-us-west" as discrete
"environments." To sidestep any such confusion, K8sTA all but entirely shuns the
"environment" nomenclature and deals instead with "stations" -- _as in stops
along a railroad track._

A K8sTA `Station` references zero or more existing Argo CD `Application`
resources and a `Track` enumerates a series of such `Station`s that a change
must traverse, _in order_, on its path to a terminal `Station` (e.g.
production). Proverbially "pulling into" a station is effected by a series of
Git commits that cause each of the referenced Argo CD `Application`s to re-sync
(i.e. GitOps). Changes do not depart for the next `Station` until all the
`Application`s referenced by the previous `Station` are both synced to the
relevant commits and healthy.

Conceptualizing the path to production as a railroad track has other advantages
as well:

1. In the real world, two trains may simultaneously traverse the same railroad
   track, but one can never pass the other, and so it is with K8sTA's `Track`s.
   One change can not overtake another -- which is a sensible limitation given
   that the Argo CD `Application` resources referenced by each `Station` can
   only manage the deployment of a _single_ revision of your software at any
   given moment.

1. Changes traversing a `Track` can be conceptualized as ticketed passengers. A
   K8sTA `Ticket` resource describes a change bound for the terminal `Station`
   (e.g. production). A `Ticket`'s `Status` sub-resource reflects the change's
   progress on that journey. This is similar in some respects to a conductor
   punching a passenger's ticket.

1. In the real world, multiple railroad tracks may converge at a station, making
   stations _junctions_ for multiple tracks. K8sTA `Station`s behave similarly.
   In addition to referencing zero or more Argo CD `Application` resources,
   `Station`s may also reference zero or more `Tracks`. Proverbially "pulling
   into" such a station not only results in a a series of Git commits that cause
   each of the referenced Argo CD `Application`s to re-sync, it _also_ results
   in the creation of a new `Ticket` resource for each referenced `Track`. This
   effectively allows changes to fan out and progress independently along
   multiple linear `Track`s.

## Design Philosophy and the KCR Stack

K8sTA embraces the
[Unix philosophy of doing one thing and doing it well](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well).
K8sTA also avoids re-inventing the wheel. Specifically, this means relying on
Argo CD and, to a lesser but still-significant extent, Argo Rollouts to do what
they already do so well. This makes K8sTA the final component of the vertically
integrated <b><u>K</u></b>8sTA + Argo <b><u>C</u></b>D + 
Argo <b><u>R</u></b>ollouts (KCR) stack.

While this may initially seem complex, the KCR stack decomposes quite cleanly
with each component fulfilling a discrete purpose. From the bottom up:

* __Argo Rollouts:__ Manages progressive delivery of services _within_ the
  context of an Argo CD `Application` using battle-tested patterns like canary
  or blue/green deployments. Argo Rollouts can also execute user-defined
  analyses to automatically complete or roll back changes. This feature
  implicitly provides the KCR stack with "quality gates" that must be cleared
  in order for a change to progress from one `Station` to the next. Argo
  Rollouts can reasonably be omitted from your own stack if its features are not
  useful to you.

* __Argo CD:__ Manages application deployment by continuously syncing with
  configuration stored in a Git repository (i.e. GitOps). This feature
  implicitly provides the KCR stack with the means of deploying changes.

* __K8sTA:__ Understands the relationship _between_ multiple Argo CD
  `Application`s and automates the Git operations that migrate changes from one
  `Application` (or set of `Application`s) to the next.

K8sTA also aims to not _interfere_ with the normal function of either Argo CD or
Argo Rollouts. Although many changes, such as new Docker images or changes to
base configuration may need to progress along a `Track` to a terminal `Station`
(i.e. production), many other changes (like updates to an environment variable,
perhaps) are `Application`-specific and can be dealt with by Argo CD without any
K8sTA involvement.

## Opinions

_K8sTA is also opinionated._ Rather than providing users with a litany of
options, it favors support for specific best-in-class tools and patterns, and
embraces convention over configuration. Its goal is to "just work."

Here is a small set of assumptions we are embracing:

* You intend to continuously deliver a first-party applications using a SaaS
  model. You're interested in getting your _own_ code into a your _own_ live
  environments as quickly as possible (without sacrificing quality). You're
  _not_ "shrink wrapping" software (in the form of a Helm chart, for instance)
  for another party to install in _their_ environments.

* Your applications are hosted in Kubernetes clusters.

* You're using GitOps to describe your infrastructure, applications,
  configuration, etc. as code -- specifically, _declarative_ code in the
  form of Kubernetes manifests (or something that can be rendered into
  Kubernetes manifests).

* You're using [kustomize](https://github.com/kubernetes-sigs/kustomize) to
  render `Application`-specific Kubernetes manifests from base manifests
  overlaid with `Application`-specific configuration and the rendered YAML is
  stored in `Application`-specific branches of a Git repository.

  > 📝&nbsp;&nbsp;We anticipate supporting
  > [ytt](https://github.com/vmware-tanzu/carvel-ytt) as well.

* You're using Argo CD to sync `Application`-specific branches of a Git
  repository with your Kubernetes cluster(s).

## Getting Started

We have a
[quickstart](https://docs-k8sta-akuity-io.netlify.app/getting-started/quickstart)
now! (The password is `akuitydocs`.)

This documentation is very new, so please open issues against this repository if
you encounter any difficulties with it.

## Roadmap

Visit our dedicated [Roadmap](metadocs/ROADMAP.md) doc for details about what we've
accomplished so far and what we're currently working on.

## Contributing

The K8sTA project accepts contributions via GitHub pull requests.

Visit our [K8sTA Contribution Guide](metadocs/CONTRIBUTING.md) for more info on how
to get started quickly and easily.

## Support & Feedback

To report an issue, request a feature, or ask a question, please open an issue
[here](https://github.com/akuityio/k8sta-prototype/issues).

## Code of Conduct

Participation in the K8sTA project is governed by the
[Contributor Covenant Code of Conduct](metadocs/CODE_OF_CONDUCT.md).
