---
slug: /
title: Overview
description: Overview
---

# K8sTA -- The Kubernetes Transit Authority

K8sTA is a _prototype_ progressive delivery tool built on top of Argo CD for
teams that love Kubernetes and GitOps. By automating away much of the tedium
inherent to GitOps, it is able to orchestrate the rollout of changes through a
series of environments, usually ending with changes reaching production, all
while observing good GitOps practices and maintaining an impeccable paper trail.

:::caution
K8sTA is highly experimental at this time and breaking changes should be
anticipated between pre-GA minor releases.
:::

## The Tracks & Stations paradigm

The term "environment" is ambiguous -- meaning different things to different
people. Is it a single instance of your application? Is it multiple instances of
your application distributed across different zones or regions? It's not worth
debating. To sidestep such ambiguity, K8sTA avoids the term "environment"
altogether and deals instead with "stations" -- _as in stops along a railroad
track._

A K8sTA `Station` resource references zero or more existing Argo CD
`Application` resources. A `Track` resource enumerates a series of such
`Station`s that changes must traverse, _in order_. Usually, the last `Station`
on a `Track` represents production. This means that for a change to reach
production, it must pass through prior `Station`s like "dev," "int," or
"staging" on its way.

At each station, K8sTA creates the necessary Git commits that cause the
referenced Argo CD `Application`s to re-sync. Changes do not progress to the
next `Station` until all the `Application`s referenced by the current `Station`
are both synced to the relevant commits and healthy.

Changes traversing a `Track` can be conceptualized as ticketed passengers. A
K8sTA `Ticket` resource describes a change bound for the terminal `Station`
(e.g. production). A `Ticket`'s `Status` sub-resource reflects the change's
progress on that journey. This is similar in some respects to a conductor
punching a passenger's ticket.

In the real world, multiple railroad tracks may converge at a station, making
stations _junctions_ for multiple tracks. K8sTA `Station`s behave similarly. In
addition to referencing Argo CD `Application` resources, `Station`s may also
reference other `Tracks`. Arriving at such a `Station` results in the creation
of a new `Ticket` resource for each referenced `Track`. This permits
changes to fan out and progress concurrently and independently along multiple
`Track`s.

## The KCR Stack

K8sTA embraces the
[Unix philosophy of doing one thing and doing it well](https://en.wikipedia.org/wiki/Unix_philosophy#Do_One_Thing_and_Do_It_Well)
and also avoids re-inventing the wheel. Specifically, this means relying on
Argo CD and Argo Rollouts to do what they already do so well. This makes K8sTA
the final component of the vertically
integrated <b><u>K</u></b>8sTA + Argo <b><u>C</u></b>D + Argo <b><u>R</u></b>ollouts
(KCR) Stack.

The KCR Stack decomposes cleanly with each component fulfilling a discrete
purpose. From the bottom up:

* __Argo Rollouts:__ Manages progressive delivery of services _within_ the
  context of a single Argo CD `Application` using battle-tested patterns like
  canary or blue/green deployments. Argo Rollouts can also execute user-defined
  analyses to automatically complete or roll back changes. This feature
  effectively provides the KCR Stack with "quality gates" that must be cleared in
  order for a change to progress from one `Station` to the next.
  
  :::info
  Argo Rollouts can safely be omitted from your own stack if its features are
  not useful to you.
  :::

* __Argo CD:__ Manages application deployment by continuously syncing with
  configuration stored in a Git repository (i.e. GitOps). This feature
  provides the KCR Stack with the means of deploying changes.

* __K8sTA:__ Understands the relationship _between_ multiple Argo CD
  `Application`s and automates the tedious Git operations required to migrate
  changes from one `Application` (or set of `Application`s) to the next.

## Opinions

K8sTA is _opinionated._ Rather than providing users with a litany of options, it
favors support for specific best-in-class tools and patterns, and embraces
convention over configuration. Its goal is to "just work."

Here is a small set of assumptions K8sTA embraces:

* Your goal is to continuously deliver first-party applications using a SaaS
  model. In other words, you're deploying your _own_ software on your _own_
  infrastructure. You're _not_ "shrink wrapping" software for someone else to
  install on _their_ infrastructure.

* You're using GitOps to describe your infrastructure, applications,
  configuration, etc. as code -- specifically, _declarative_ code in the
  form of Kubernetes manifests (or something that can be rendered into
  Kubernetes manifests).

* You're using Argo CD to sync "rendered YAML branches" (i.e.
  `Application`-specific branches of your GitOps repository) with your
  Kubernetes cluster(s).
