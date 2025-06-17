---
sidebar_label: GHCR
---

# Receiving Webhooks from the GitHub Container Registry

Webhooks cannot be registered directly on a GHCR image repository. Instead,
`package` events are delivered from an associated source code repository as if
the event precipitating the delivery had occurred there.

Refer to documentation for the
[GitHub Webhooks Receiver](./github.md) for further instructions.

:::note
If your GHCR image repository has not yet been associated with a source code
repository,
[refer to these instructions](https://docs.github.com/en/packages/learn-github-packages/connecting-a-repository-to-a-package).
:::

:::note
GitHub can deliver webhooks _only_ for events occurring in a container image
repository associated with a source code repository.

This is a limitation of GitHub/GHCR and not a limitation of Kargo.
:::
