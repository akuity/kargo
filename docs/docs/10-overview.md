---
slug: /
title: Overview
Description: What is Kargo?
---

# What is Kargo?

Kargo is a next-generation continuous delivery (CD) platform for Kubernetes. It
builds upon established practices (like [GitOps](https://opengitops.dev/)) and
existing technology (like [Argo CD](https://argoproj.github.io/cd/)) to
streamline, or even automate, the progressive rollout of changes across multiple
environments.

:::caution
Kargo is still in its early stages and undergoing heavy development, so we
discourage anyone from using it in production environments at this time, but all
are invited to [join us in improving Kargo](https://github.com/akuity/kargo).
:::

## Tell me more!

At [Akuity](https://akuity.io/), we've heard one consistent thing from our
customers -- they want a sensible and mostly automated means of progressing
changes through a series of environments.

* Have you ever wanted your "test" or "dev" environment to update automatically
  as new images or Kubernetes manifests become available?

* Have you ever wanted your "integration" or "stage" environment to update
  automatically as soon as new images or Kubernetes manifests have proven
  themselves stable in "test" or "dev?"

* Have you ever wanted to _promote_ images and Kubernetes manifests from your
  "stage" environment to your "prod" environment with just a few clicks or
  keystrokes?

* Have you struggled to automate these sort of workflows? Have you leaned on
  platforms like GitHub Actions and Jenkins to facilitate that automation and
  found they weren't designed with these use cases in mind?

If you've answered "yes" to any of these questions, Kargo might be right for
you.

## Our goal

Kargo's goal is to provide an intuitive and flexible layer "above" your GitOps
repositories and platforms, wherein you can describe the relationships between
environments, sources of materials (such as container images or Kubernetes
manifests), _how_ to apply those materials to each environment (typically by
interacting with repositories, configuration management tools, and Argo CD), and
the conditions under which new materials may progress from one logical
environment to the next.

## Next steps

To learn more about Kargo, consider checking out our
[concepts doc](./concepts) or get hands-on right away with our
[quickstart](./quickstart)!
