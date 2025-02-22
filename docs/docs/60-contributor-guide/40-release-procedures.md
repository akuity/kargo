---
description: Steps required by Kargo maintainers to orchestrate a minor or major release of Kargo.
sidebar_label: Release Procedures
---

# Release Procedures

This document outlines the few manual steps that Kargo maintainers must follow
to orchestrate a minor or major release of Kargo.

## Conventions

Throughout this document, we will use:

* `M` and `m` to denote the major and minor version numbers of the upcoming
  release (i.e. the one being performed), respectively. 

* `M-1` and `m-1` to denote the _previous_ major and minor version numbers,
  respectively.

* `M+1` and `m+1` to denote the _next_ major and minor version numbers,
  respectively.

* `L` to denote the latest minor version of the `M-1` line.

## Timeline

The steps outlined below should be started on the Monday preceding the expected
release date -- which is always a Friday.

## Steps

1. Open a PR [similar to this one](https://github.com/akuity/kargo/pull/1932)
   to revise the roadmap.

    * The roadmap should be updated to reflect that work for the `vM.m.0`
      release is complete.
    * Planned work that was not completed should be moved to a future release.
    * The next release (`vM.m+1.0` or `vM+1.0.0`) should be updated to reflect
      that work is in-progress, along with the expected release date.

      :::note
      "Edge" documentation at
      [main.kargo.akuity.io](https://main.docs.kargo.io) is continuously
      published from the `main` branch and production documentation at
      [kargo.akuity.io](https://docs.kargo.io) is continuously published from
      the previous release branch (`release-M.m-1` or `release-M-1.L`). There
      are two consequences of this:

      * The production documentation will _not_ immediately reflect changes made
        to the `main` branch, nor will it reflect changes made to the release branch
        for the upcoming release. (Which does not exist yet. See next step.)

      * This step should ideally be performed _prior_ to the creation of the
        release branch (see next step) in order to avoid the need for two separate
        PRs to update both branches.
      :::

1. Create a release branch of the form `release-M.m`.

    This can be done locally by a maintainer. Presuming `upstream` is a remote
    pointing to the main Kargo repository:

      ```shell
      git checkout main
      git pull upstream main
      git checkout -b release-M.m
      git push upstream release-M.m
      ```

    :::note
    After the creation of this branch, anything merged to `main` is excluded
    from the upcoming release unless explicitly cherry-picked into the
    `release-M.m branch`. As such, this step should ideally be performed
    _after_ the majority of work for the upcoming release is complete.

    In some cases, this may be performed early to:

      * Un-block work on the next release.
      * Facilitate the creation of a release candidate for use by non-engineers
        while work on the upcoming release continues.
    :::

1. Merge any release-specific upgrade logic into the `release-M.m` branch.

    :::info
    Pre-`v1.0.0`, we are making a best effort to automatically compensate for
    breaking changes between minor releases for users upgrading _directly_ from
    any release in the `v0.m-1` line. This means release-specific upgrade
    logic does not need to be merged into `main`.
    :::

1. Open a PR [similar to this one](https://github.com/akuity/kargo/pull/1925)
   against the previous release branch (`release-M.m-1` or `release-M-1.L`) to
   lock production documentation (e.g. for download and installation procedures)
   into permanently reflecting the latest stable release.

    :::note
    Production documentation is continuously published from the previous
    release branch, so this step is necessary to ensure that the production
    documentation is not inadvertently broken by any subsequent steps.

    This step will also ensure that when the current production documentation
    is archived, it will reflect the latest release to which that documentation
    was applicable.
    :::

1. Cut `vM.m.0-rc.1` from the Kargo
   [release page](https://github.com/akuity/kargo/releases/new).

    * The release process itself is fully-automated.
    * Be certain to reference the head of the `release-M.m` branch and _not_ `main`.
    * Be sure to check the __"Set as a pre-release"__ box.
    * Wait for the
      [automated release process](https://github.com/akuity/kargo/actions/workflows/release.yaml)
      to complete.

1. Open a PR [like this one](https://github.com/akuity/kargo/pull/1926) against
   `main` to make the edge documentation (e.g. for download and installation
   procedures) reflect the recently built release candidate.

    :::info
    The edge documentation is continuously published from the `main` branch, so
    this step makes it easy for non-engineers to test the release candidate by
    adhering to instructions in the edge documentation, without any need to
    compensate for the release candidate not being counted as "latest" on
    account of being a pre-release.
    :::

1. Alert non-engineer stakeholders to the availability of the release candidate.

1. Bug fixes and last minute features should be merged to `main` and backported
   to the `release-M.m` (in bulk, when possible).

1. Repeat steps 5-8 as necessary until the release candidate is deemed stable
   by relevant stakeholders.

1. Draft release notes for the upcoming release.

    :::info
    This can be done concurrently with the previous steps.

    Some stakeholders may desire early access to these notes to inform blog
    posts, marketing materials, etc.
    :::

1. Cut `vM.m.0` from the Kargo
   [release page](https://github.com/akuity/kargo/releases/new).

    * Be certain to reference the head of the `release-M.m` branch and _not_ `main`.
    * Be certain to include the final draft of the release notes.
    * Be sure to check the __"Set as the latest release"__ box.
    * Wait for the
      [automated release process](https://github.com/akuity/kargo/actions/workflows/release.yaml)
      to complete.

1. Mark the release branch (`release-M.m`) as the __"Production branch"__
   [in Netlify](https://app.netlify.com/sites/docs-kargo-akuity-io/configuration/deploys#branches-and-deploy-contexts).

    * Also add the previous release branch (`release-M.m-1` or
      `release-M-1.L`) to __"Branch deploys"__.
    * After changing the __"Production branch"__, it will be necessary to
      [manually trigger a deployment](https://app.netlify.com/sites/docs-kargo-io/deploys)
      of the production documentation.

1. Open a PR to revert the changes from step 6.

1. Inform relevant stakeholders that the release is complete.

1. 🎉 Celebrate!
