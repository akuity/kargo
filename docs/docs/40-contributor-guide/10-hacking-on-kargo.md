---
description: Learn how to set up a development environment to begin contributing to Kargo
sidebar_label: Hacking on Kargo
---

# Hacking on Kargo

Kargo is implemented in Go. For maximum productivity in your text editor or IDE,
it is recommended that you have installed the latest stable releases of Go and
applicable editor/IDE extensions, however, this is not strictly required to be
successful.

## Running Tests

In order to minimize the setup required to apply small changes and to reduce the
incidence of tests passing locally, but failing during the continuous
integration process due to environmental differences, we've made it easy to
execute tests within a container that is maximally similar to those used in CI.

To take advantage of this, you only need `make` and
[Docker](https://docs.docker.com/engine/install/) (or a Docker-compatible 
container-runtime).

To run all unit tests:

```shell
make hack-test-unit
```

:::info
If you wish to opt-out of executing the tests within a container (for
performance reasons, perhaps), drop the `hack-` prefix from the target:

```shell
make test-unit
```

This will require Go to be installed locally.
:::

## Running Linters

It is also possible to execute a variety of different linters that perform
static code analysis, detect code hygiene issues, assert adherence to project
standards, etc. As with unit tests, we've made it easy to execute linters within
a container that is maximally similar to those used in CI.

To lint Go code only:

```shell
make hack-lint-go
```

To lint generated protobuf definitions only:

```shell
make hack-lint-proto
```

To lint Helm charts only:

```shell
make hack-lint-charts
```

To run _all_ linters with one command:

```shell
make hack-lint
```

:::info
If you wish to opt-out of executing any or all linters within a container (for
performance reasons, perhaps), drop the `hack-` prefix from the desired target.

This will require quite a variety of tools to be installed locally, so we do not
recommend this if you can avoid it.
:::

## Executing Code Generation

Anytime the contents of the `api/` directory have been modified, a code
generation process must be manually executed. As with tests and linters, this
process is easy to execute within a container, which eliminates the need to
install various tools or specific versions thereof:

```shell
make hack-codegen
```

:::info
If you wish to opt-out of executing code-generation within a container (for
performance reasons, perhaps), drop the `hack-` prefix from the target:

```shell
make codegen
```

This will require quite a variety of tools to be installed locally, so we do not
recommend this if you can avoid it.
:::

## Building the Image

To build source into a Docker image that will be tagged as `kargo:dev`,
execute the following:

```shell
make hack-build
```

:::info
There is seldom a need to do this, as the next section will cover a better
option for rapidly building and deploying Kargo from source.
:::

:::tip
The [Docker buildx](https://github.com/docker/buildx) machine required by the
build process has to be created with the `--driver-opt network=host` option to
allow it to access the (temporary) local image registry used for the base image.

If you encounter an error during the build process (e.g. `failed to resolve
source metadata for localhost:5001/kargo-base:latest-arm64` or `granting
entitlement network.host is not allowed by build daemon configuration`), you
may need to (re)create the machine using `docker buildx create` with this
option set.
:::

## Iterating Quickly

This section focuses on the best approaches for gaining rapid feedback on
changes you make to Kargo's code base.

The fastest path to learning whether changes you have applied work as desired is
to execute unit tests as described in previous sections. If the changes you are
applying are complex, it can also be advantageous to exercise them, end-to-end,
as a user would. Because Kargo is dependent on a Kubernetes cluster, this raises
the question of how Kargo can not only be built from source, but also deployed
to a live Kubernetes cluster efficiently enough to enable a tight feedback loop
as you continue iterating on your changes.

The remainder of this section covers the approaches we recommend for enabling
this.

:::info
We may eventually provide convenient methods of running _some_ Kargo components
as native processes.
:::

1. Launch or re-use an existing local Kubernetes cluster.

    Any of the following options are viable:

    <Tabs groupId="cluster-start">
    <TabItem value="docker-desktop" label="Docker Desktop">

    If you are a
    [Docker Desktop](https://www.docker.com/products/docker-desktop/)
    user, you can follow
    [these instructions](https://docs.docker.com/desktop/kubernetes/) to enable
    its built-in Kubernetes support.

    :::info
    A specific benefit of this option is that nothing special is required in
    terms of creating a local image registry connected to the cluster.
    Additionally, this approach requires no specific port-forwarding rules to be
    defined.
    :::
    :::info
    Although this is one of the fastest paths to a local Kubernetes cluster, be
    aware that Docker Desktop supports only a _single_ Kubernetes cluster. If
    that cluster reaches a state you are dissatisfied with, resetting it will
    remove not just Kargo-related resources, but _all_ your workloads and data.
    :::

    To install Kargo's prerequisites, you will need
    [Helm](https://helm.sh/docs/intro/install/) installed first, and can then
    execute a convenient `make` target:

    ```shell
    make hack-install-prereqs
    ```
   
    </TabItem>
    <TabItem value="orbstack" label="OrbStack">

    [OrbStack](https://orbstack.dev/) is a fast, lightweight, drop-in replacement
    for Docker Desktop for Mac OS only. You can follow
    [these instructions](https://docs.docker.com/desktop/kubernetes/) to enable
    its built-in Kubernetes support.

    :::info
    A specific benefit of this option is that nothing special is required in
    terms of creating a local image registry connected to the cluster.
    Additionally, this approach requires no specific port-forwarding rules to be
    defined.
    :::
    :::info
    Although this is one of the fastest paths to a local Kubernetes cluster, be
    aware that OrbStack supports only a _single_ Kubernetes cluster. If
    that cluster reaches a state you are dissatisfied with, resetting it will
    remove not just Kargo-related resources, but _all_ your workloads and data.
    :::

    To install Kargo's prerequisites, you will need
    [Helm](https://helm.sh/docs/intro/install/) installed first, and can then
    execute a convenient `make` target:

    ```shell
    make hack-install-prereqs
    ```

    </TabItem>
    <TabItem value="kind" label="kind">

    If you have any Docker-compatible container runtime installed (including
    native Docker, Docker Desktop, or OrbStack), you can easily launch a
    disposable cluster to facilitate Kargo development using
    [kind](https://kind.sigs.k8s.io/#installation-and-usage).

    This option also requires
    [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it) and
    [Helm](https://helm.sh/docs/intro/install/) to be installed.

    The following `make` target will launch a kind cluster with a local image
    registry wired into it, various port-forwarding rules pre-configured, and
    Kargo's prerequisites installed:

    ```shell
    make hack-kind-up
    ```

    :::info
    While this option is a bit more complex than using Docker Desktop or OrbStack
    directly, it offers the advantage of being fully-disposable. If your cluster
    reaches a state you are dissatisfied with, you can simply destroy it and
    launch a new one.
    :::

    </TabItem>
    <TabItem value="k3d" label="k3d">

    If you have any Docker-compatible container runtime installed (including
    native Docker, Docker Desktop, or OrbStack), you can easily launch a
    disposable cluster to facilitate Kargo development using
    [k3d](https://k3d.io).

    This option also requires
    [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it) and
    [Helm](https://helm.sh/docs/intro/install/) to be installed.

    The following `make` target will launch a kind cluster with a local image
    registry wired into it, various port-forwarding rules pre-configured, and
    Kargo's prerequisites installed:

    ```shell
    make hack-k3d-up
    ```

    :::info
    While this option is a bit more complex than using Docker Desktop or OrbStack
    directly, it offers the advantage of being fully-disposable. If your cluster
    reaches a state you are dissatisfied with, you can simply destroy it and
    launch a new one.
    :::

    </TabItem>
    </Tabs>

    Whichever approach you choose, your cluster will end up with recent, stable
    versions of [cert-manager](https://cert-manager.io/) and 
    [Argo CD](https://argoproj.github.io/cd/) installed.

    :::info
    The Argo CD dashboard will be exposed at
    [localhost:30080](https://localhost:30080).

    The username and password are both `admin`.

    You may safely ignore any certificate warnings.
    :::

1. Build and deploy Kargo from source:

    [Tilt](https://docs.tilt.dev/#macoslinux) is a convenient tool that builds
    container images from source and seamlessly deploys them to a local
    Kubernetes cluster. More importantly, it enables developers to rapidly
    rebuild and replace running components with the click of a button.

    :::warning
    If using OrbStack, be advised it is only compatible with Tilt as of Tilt
    v0.33.6. Please use that version or greater.
    :::

    ```shell
    tilt up
    ```

    Tilt will also launch a web-based UI running at
    [http://localhost:10350](http://localhost:10350). Visit this in your web
    browser to view the build and deployment status of each Kargo component as
    well as the logs from each component.

    :::info
    Tilt is often configured to watch files and automatically rebuild and replace
    running components when their source code is changed. This is deliberately
    disabled for Kargo since the Docker image takes long enough to build that
    it’s better to conserve system resources by only rebuilding when you choose.
    The web UI makes it easy to identify components whose source has been
    altered. They can be rebuilt and replaced with a single click.
    :::

1. If necessary, build the CLI from source:

    ```shell
    make hack-build-cli
    ```

    This will produce an executable at `bin/kargo-<os>-<arch>`.

    You can log in using:

    ```shell
    bin/kargo-<os>-<arch> login http://localhost:30081 \
      --admin \
      --password admin \
      --insecure-skip-tls-verify
    ```

1. If necessary, access the Kargo UI at
   [localhost:30082](http://localhost:30082).

    The admin account password is `admin`.

    You may safely ignore any certificate warnings.

1. When you are done with Tilt, interrupt the running `tilt up` process with
   `ctrl + c`. Components _will remain running in the cluster_, but Tilt will no
   longer be in control. If Tilt is restarted later, it will retake control of
   the already-running components.

    If you wish to undeploy everything Tilt has deployed for you, use `tilt
    down`.

1. Clean up your local Kubernetes cluster.

    <Tabs groupId="cluster-start">
    <TabItem value="docker-desktop" label="Docker Desktop">

    Docker Desktop supports only a _single_ Kubernetes cluster. If you are
    comfortable deleting not just just Kargo-related resources, but _all_ your
    workloads and data, the cluster can be reset from the Docker Desktop
    Dashboard.

    If, instead, you wish to preserve non-Kargo-related workloads and data, you
    will need to manually uninstall Kargo's prerequisites:

    ```
    make hack-uninstall-prereqs
    ```
   
    </TabItem>
    <TabItem value="orbstack" label="OrbStack">

    OrbStack supports only a _single_ Kubernetes cluster. If you are
    comfortable deleting not just just Kargo-related resources, but _all_ your
    workloads and data, you can destroy the cluster with:

    ```shell
    orb delete k8s
    ```

    If, instead, you wish to preserve non-Kargo-related workloads and data, you
    will need to manually uninstall Kargo's prerequisites:

    ```
    make hack-uninstall-prereqs
    ```

    </TabItem>
    <TabItem value="kind" label="kind">

    To destroy the cluster, use:

    ```shell
    make hack-kind-down
    ```

    :::info
    This command deliberately leaves your local image registry running so that if
    you resume work later, you are doing so with a local registry that’s already
    primed with most layers of Kargo’s image.

    If you wish to stop the registry, use:

    ```shell
    docker stop kargo-dev-registry
    ```

    To destroy it, use:

    ```shell
    docker rm -f kargo-dev-registry
    ```
    :::

    </TabItem>
    <TabItem value="k3d" label="k3d">

    To destroy the cluster, use:

    ```shell
    make hack-k3d-down
    ```

    :::info
    This command deliberately leaves your local image registry running so that if
    you resume work later, you are doing so with a local registry that’s already
    primed with most layers of Kargo’s image.

    If you wish to stop the registry, use:

    ```shell
    docker stop kargo-dev-registry
    ```

    To destroy it, use:

    ```shell
    docker rm -f kargo-dev-registry
    ```
    :::

    </TabItem>
    </Tabs>

## Contributing to Documentation

Contributors should ensure that their changes are accompanied by relevant documentation
updates. This helps maintain the project's sustainability. Pull requests with
corresponding documentation updates are more likely to be merged faster.

To make this process smoother, you can refer to [Docusaurus](https://docusaurus.io/docs)
for guidance on writing and maintaining docs effectively.

### Previewing Doc Changes Locally

After making your changes, preview the documentation locally to ensure everything renders
correctly. You can either run it in a container or natively on your system.

To build and serve the docs inside a container:

```shell
make hack-serve-docs
```

:::info
If you want to build and serve the docs on your local machine, run the following command:

```shell
make serve-docs
```

This will require you to install the [`pnpm`](https://pnpm.io/installation) tool locally on your machine.
:::
