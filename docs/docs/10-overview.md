---
slug: /
title: Overview
Description: What is Kargo?
---

# What is Kargo?

Kargo is a next-generation continuous delivery and application lifecycle
orchestration platform for Kubernetes. It builds upon
[GitOps](https://opengitops.dev/) principles and integrates with existing
technologies, like [Argo CD](https://argoproj.github.io/cd/), to streamline and
automate the progressive rollout of changes across the many stages of an
application's lifecycle.

:::caution
Kargo is still undergoing heavy development and is not yet ready for production,
but all are invited to
[join us in improving Kargo](https://github.com/akuity/kargo) to help it get
there.

In the meantime, breaking changes should be anticipated between pre-GA minor
releases.
:::

## Tell me more!

At [Akuity](https://akuity.io/), we've heard one consistent thing from GitOps
practitioners -- they want a sensible and mostly automated means of progressing
changes through a series of environments.

Have you ever:

* Wanted applications instances in a "test" environment to update automatically
  as new images or Kubernetes manifests become available?

* Wanted applications instances in a "UAT" environment to update automatically
  after new images or Kubernetes manifests have proven themselves stable in a
  "test" environment?

* Wanted to _promote_ images and Kubernetes manifests from an application
  instance in your "UAT" environment to an application instance in your
  "prod" environment with just a few clicks or keystrokes?

* Struggled to automate these sort of workflows? Have you over-leveraged
  CI platforms like GitHub Actions and Jenkins to facilitate that automation and
  found they weren't designed with these use cases in mind?

If you've answered "yes" to any of these questions, Kargo might be right for
you.

## Our goal

Kargo's goal is to provide an intuitive and flexible layer "above" your existing
GitOps tooling, wherein you can describe the relationships between various
application instances deployed to different environments as well as procedures
for progressing changes (such as new container images or updated Kubernetes
manifests), from one application instance's source of truth to the next.

## Next steps

To learn more about Kargo, consider checking out our
[concepts doc](./concepts) or get hands-on right away with our
[quickstart](./quickstart)!
