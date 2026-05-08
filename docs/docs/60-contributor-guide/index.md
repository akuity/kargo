---
description: Policies and procedures for contributing to the Kargo project
sidebar_label: Contributor Guide
---

# Contributor Guide

This guide defines the policies and procedures for contributing to Kargo. Please
read it in full before opening issues or pull requests.

## Process Overview

Kargo follows an **issue-first contribution model**. Unless authored by a
maintainer, all code contributions require a linked issue that has been reviewed
and unblocked by maintainers in advance.

:::caution

Ideas and contributions from the community are always welcome, however, items
not appearing on the [roadmap](../100-roadmap.md) are reviewed on a best-effort
basis. While we cannot guarantee specific timelines or outcomes, we value and
consider every submission.

:::

1. **Open an issue** using the
   [Bug Report](https://github.com/akuity/kargo/issues/new?template=bug_report.yml)
   or
   [Feature Request](https://github.com/akuity/kargo/issues/new?template=feature_request.yml)
   template.

1. **Wait for maintainer review.** Maintainers will triage the issue, ask
   clarifying questions, and determine whether the work aligns with the
   project's priorities and [roadmap](../100-roadmap.md).

1. **Wait for the issue to be unblocked.** When an issue is ready for external
   contribution, a maintainer will remove all blocking labels. **Do not begin
   work while any blocking labels are present.**

1. **Open a pull request** that references the unblocked issue using
   `Closes #<number>` in the PR body.

## Blocking Labels

The following labels indicate an issue is **not ready for external
contribution**. Pull requests linked to issues carrying any of these labels are
automatically converted to a draft until the issue is unblocked.

| Label | Meaning |
| ----- | ------- |
| `kind/proposal` | Feature request under consideration. Not yet unblocked. |
| `needs discussion` | Needs further discussion before any work begins. |
| `needs research` | Needs investigation or research before work begins. |
| `maintainer only` | Reserved for maintainers due to size, complexity, or sensitivity. |
| `area/security` | Involves security-sensitive code. Maintainer-coordinated only. |
| `size/large` | Large scope. Requires maintainer coordination. |
| `size/x-large` | Very large scope. Requires maintainer coordination. |
| `size/xx-large` | Extremely large scope. Requires maintainer coordination. |

An issue is ready for external contribution when **none** of these labels are
present.

### Exceptions

A few kinds of contribution are exempt from the issue-first requirement and can
be opened as pull requests directly, without a corresponding issue:

- **Drive-by fixes.** Pull requests of five or fewer lines (added + removed,
  combined). This covers typo corrections, comment fixes, and small, obvious bug
  fixes.

- **Documentation-only changes.** Pull requests whose changes are limited to
  `README.md` and Markdown files anywhere under `docs/`.

Exempt pull requests must still adhere to the
[Quality Expectations](#quality-expectations) below — the issue requirement is
the only thing waived.

## Quality Expectations

All pull requests must:

- Be authored and reviewed by a human. AI-assisted coding is acceptable, but
  every line must be understood and verified by the submitter. Submissions that
  appear to be generated without meaningful human review will be closed.

- Include appropriate tests for new and modified code.

- Follow existing code conventions and patterns.

- Include DCO sign-off on all commits (see [Signing Commits](signing-commits)).

- Include documentation updates where user-facing behavior has changed.

Cryptographic signing of all commits is strongly encouraged.

## Specific Topics

import DocCardList from '@theme/DocCardList';

<DocCardList />
