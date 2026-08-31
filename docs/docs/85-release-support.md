---
title: Release Support
sidebar_label: Release Support
description: Kargo release dates, latest patch versions, and when Critical CVE backport coverage ends for each Kargo release line.
---

import ReleaseSupportTable from '@site/src/components/ReleaseSupportTable';

# Release Support

Kargo's maintainers publish a new minor release roughly four times a year. This
page records when each minor release was published, its most recent patch, and
how long Akuity backports fixes for Critical vulnerabilities to it.

<ReleaseSupportTable />

## What these dates mean

The final column is the end of the Critical CVE backport window for **AKP
commercial builds of Kargo** — the `ak`-tagged images distributed as part of
the Akuity Platform. Akuity backports fixes for Critical vulnerabilities to
AKP builds of any Kargo release published within the previous 12 months.

Coverage is bound by severity as well as by time:

| Severity | Backported to AKP builds of releases published within |
|----------|-------------------------------------------------------|
| Critical | the last 12 months |
| High | the last 6 months |
| Medium / Low | the current minor release |

Each window is measured from the affected release's publication date — the
date in the <Hlt>Released</Hlt> column above — and a Kargo instance must be
running a release still inside the applicable window to be eligible. A
Critical vulnerability affecting a release published thirteen months ago falls
outside the window; the fix for it lands in supported releases, and the
instance running the older release upgrades to receive it.

Critical and High severity fixes in Kargo's own code may be delivered as
out-of-band, security-only patch releases rather than waiting for the next
minor release.

## Open source Kargo

The open-source distribution — Apache 2.0, published to
`ghcr.io/akuity/kargo` — carries no time- or severity-bound backport
guarantee. Security fixes land in the latest release, and open-source users
upgrade to receive them. Fixes are not backported to older release lines.
Maintenance is best-effort and the software is provided "as is" under its
license.

:::note
The dates in the table above therefore describe AKP coverage. If you run the
open-source distribution, the supported version is the latest release, whatever
the table says about the line you are on.
:::

Open-source and commercial builds are produced from the same source on the same
release schedule, so a fix lands in both at the same time. What differs is the
commitment around it and the range of releases it is backported to.

Open-source users receive the same transparency artifacts Akuity publishes with
every release:

- A **Software Bill of Materials (SBOM)** enumerating the components in the
  image.
- **Signed VEX statements** recording whether each known vulnerability is
  actually exploitable in Kargo. Scanners commonly flag components that are
  present in an image but never invoked; VEX dispositions those findings
  formally so your scanner can filter them. Point your scanner at
  [vex.akuity.io](https://vex.akuity.io) to ingest them, or read them from the
  Sigstore attestations attached to each released image.
- Public security advisories.
