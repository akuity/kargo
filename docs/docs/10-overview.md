---
slug: /
sidebar_label: Overview
description: Find out more about Kargo - a next-generation continuous delivery and application lifecycle orchestration platform for Kubernetes
---

# What is Kargo?

Kargo is a next-generation continuous delivery and application lifecycle
orchestration platform for Kubernetes. It builds upon
[GitOps](https://opengitops.dev/) principles and integrates with existing
technologies, like [Argo CD](https://argoproj.github.io/cd/), to streamline and
automate the progressive rollout of changes across the many stages of an
application's lifecycle.

![Screenshot](../static/img/screenshot.png)

Kargo's goal is to provide an intuitive and flexible layer "above" existing GitOps tooling, wherein you can describe the relationships between various application instances deployed to different environments as well as procedures for progressing changes from one application instance's source of truth to the next.

:::info
Watch the *Multi-Stage Deployment Pipelines the GitOps Way* talk by Jesse Suen & Kent Rancourt of Akuity at GitOpsCon EU 2024.

<center>
<iframe width="560" height="315" src="https://www.youtube.com/embed/tl8w72tehZ0?si=fdSx63nsCQHR1eP0" title="YouTube video player" frameBorder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture;" allowFullScreen></iframe>
</center>
:::

:::caution
Kargo is undergoing active development and everyone is invited to [join us](https://github.com/akuity/kargo) in the journey to a GA release (`v1.0.0`)! Please expect breaking changes between pre-GA releases (`v0.x.x`).
:::

:::info
Join the Akuity Community [Discord server](https://discord.gg/dHJBZw6ewT)!
:::

## Next Steps

To learn more about Kargo, consider checking out our
[concepts doc](./concepts) or get hands-on right away with our
[quickstart](./quickstart)!
