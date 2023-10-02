---
description: Hacking on Kargo
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

To take advantage of this, you only need to have
[Docker](https://docs.docker.com/engine/install/) and `make` installed.
> you can use podman by creating an eenvironment variable *export CONTAINER_RUNTIME=podman*, [How to config podman socket](https://docs.podman.ioen/latest/markdown/podman-system-service.1.html#examples)

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

The remainder of this section covers the tools we recommend for enabling this.

You will need:

* [Docker](https://docs.docker.com/engine/install/)

* [kind](https://kind.sigs.k8s.io/#installation-and-usage) or
  [k3d](https://k3d.io): Runs development-grade Kubernetes clusters within
  Docker containers.

* [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it): Launches
  kind or k3d clusters that are pre-connected to a local image registry.

* [Tilt](https://docs.tilt.dev/#macoslinux): Builds components from source and
  deploys them to a development-grade Kubernetes cluster. More importantly, it
  enables developers to rapidly rebuild and replace running components with the
  click of a button.

* [Helm](https://helm.sh/docs/intro/install/): The package manager for
  Kubernetes. Tilt will use this to help deploy Kargo from source.

Follow the installation instructions for each of the above. Once these are
installed, you can be up and running with just a few commands!

1. Launch a kind or k3d cluster pre-connected to a local image registry:

   ```shell
   make hack-kind-up
   ```

   Or:

   ```shell
   make hack-k3d-up
   ```

   Either of these commands will _also_ install recent, stable versions of
   [cert-manager](https://cert-manager.io/) and
   [Argo CD](https://argoproj.github.io/cd/).

   :::info
   The Argo CD dashboard will be exposed at
   [localhost:30080](https://localhost:30080).

   The username and password are both `admin`.

   You may safely ignore any certificate warnings.
   :::

1. Build and deploy Kargo from source:

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
   bin/kargo-<os>-<arch> login https://localhost:30081 \
     --admin \
     --password admin \
     --insecure-skip-tls-verify
   ```

1. If necessary, access the Kargo UI at
   [localhost:30081](https://localhost:30081).

   The admin account password is `admin`.

   You may safely ignore any certificate warnings.

1. When you are done with Tilt, interrupt the running `tilt up` process with
   `ctrl + c`. Components _will remain running in the cluster, but Tilt will no
   longer be in control. If Tilt is restarted later, it will retake control of
   the already-running components.

   If you wish to undeploy everything Tilt has deployed for you, use `tilt
   down`.

1. If you wish to destroy your local kind or k3d cluster, use:

   ```shell
   make hack-kind-down
   ```

   Or:

   ```shell
   make hack-k3d-down
   ```

   :::info
   `make hack-kind-down` and `make hack-k3d-down` deliberately leave your local image registry
   running so that if you resume work later, you are doing so with a local
   registry that’s already primed with most layers of Kargo’s image.

   If you wish to stop the registry, use:

   ```shell
   docker stop kargo-dev-registry
   ```

   To destroy it, use:

   ```shell
   docker rm -f kargo-dev-registry
   ```

   :::
