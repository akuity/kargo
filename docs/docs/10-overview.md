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
