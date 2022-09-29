---
description: Hacking on K8sTA
---

# Hacking on K8sTA

K8sTA is implemented in Go. For maximum productivity in your text editor or IDE,
it is recommended that you have installed the latest stable releases of Go and
applicable editor/IDE extensions, however, this is not strictly required to be
successful.

## Containerized tests

In order to minimize the setup required to successfully apply small changes and
in order to reduce the incidence of “it worked on my machine,” wherein changes
that pass tests locally do not pass the same tests in CI due to environmental
differences, K8sTA has adopted a “container-first” approach to testing. This
is to say we have made it the default that unit tests, linters, and a variety of
other validations, when executed locally, automatically execute in a Docker
container that is maximally similar to the container in which those same tasks
will run during the continuous integration process.

To take advantage of this, you only need to have
[Docker](https://docs.docker.com/engine/install/) and `make` installed.

If you wish to opt-out of tasks automatically running inside a container, you
can set the environment variable `SKIP_DOCKER` to the value `true`. Doing so
will require that any tools involved in tasks you execute have been installed
locally. 

## Working with Go code

If you make modifications to Go code, it is recommended that you run
corresponding unit tests and linters before opening a PR.

To run all unit tests:

```shell
make test-unit
```

To run lint checks:

```shell
make lint
```

## Iterating quickly

This section focuses on the best approaches for gaining rapid feedback on
changes you make to K8sTA's code base.

By far, the fastest path to learning whether changes you have applied work as
desired is to execute unit tests as described in previous sections. If, however,
the changes you are applying are not well-covered by unit tests, it can become
advantageous to build K8sTA from source, including your changes, and deploy it
to a live Kubernetes cluster. After doing so, you can test changes manually.
Under these circumstances, a pressing question is one of how K8sTA can be
built/re-built and deployed as quickly as possible.

Building and deploying K8sTA as quickly as possible requires minimizing the
process' dependency on remote systems – including Docker registries and
Kubernetes. To that end, we recommend a specific configuration wherein Docker
images are built and pushed to a local image registry and a local Kubernetes
cluster is configured such that it can pull images from that local registry. To
achieve this with minimal effort, you will need to install the latest stable
versions of:

* [kind](https://kind.sigs.k8s.io/#installation-and-usage): Runs
  development-grade Kubernetes clusters in Docker.

  :::note
  If you strongly prefer [k3d](https://k3d.io), please consider opening a PR to
  add support.
  :::

* [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it): Launches
  development-grade Kubernetes clusters (in kind, for instance) that are
  pre-connected to a local image registry.

* [Tilt](https://docs.tilt.dev/#macoslinux): Builds components from source and
  deploys them to a development-grade Kubernetes cluster. More importantly, it
  enables developers to rapidly rebuild and replace running components with the
  click of a button.

* [Helm](https://helm.sh/docs/intro/install/): The package manager for
  Kubernetes. Tilt will use this to help deploy K8sTA from source.

Follow the installation instructions for each of the above.

:::info
Once these tools are installed, you can be up and running with just a few
commands!
:::

To launch a brand new Kind cluster pre-connected to a local image registry:

```shell
make hack-kind-up
```

Because K8sTA integrates directly with Argo CD, the above command will _also_
install a recent, stable version of that.

:::info
The Argo CD dashboard will be exposed at `localhost:30081`.

The username and password are both `admin`.
:::

K8sTA has no _direct_ dependency on Argo Rollouts and no dependency at all on
Istio, but because one or both of these are often required to enable test
applications, they can be easily added to the local development cluster.

To add Argo Rollouts:

```shell
make hack-add-rollouts
```

To add Istio:

```shell
make hack-add-istio
```

:::info
The Istio ingress controller / gateway will be exposed at `localhost:30080`.
:::

To build and deploy K8sTA from source:

```shell
tilt up
```

Tilt will also launch a web-based UI running at
[http://localhost:10350](http://localhost:10350). Visit this in your web browser
and you will be able to see the build and deployment status of each K8sTA
component.

:::info
Tilt is often configured to watch files and automatically rebuild and replace
running components when their source code is changed. This is deliberately
disabled for K8sTA since the Docker image takes long enough to build that it’s
better to conserve system resources by only rebuilding when you choose. The web
UI makes it easy to identify components whose source has been altered. They can
be rebuilt and replaced with one mouse click.
:::

When you are done with Tilt, interrupt the running `tilt up` process with
`ctrl + c`. Components _will remain running in the cluster_, but Tilt will no
longer be in control. If Tilt is restarted later, it will retake control of the
already-running components.

If you wish to undeploy everything Tilt has deployed for you, use `tilt down`.

To destroy your kind cluster, use `make hack-kind-down`.

:::info
`make hack-kind-down` deliberately leaves your local registry
running so that if you resume work later, you are doing so with a local
registry that’s already primed with most layers of K8sTA’s image.

If you wish to destroy the registry, use:

```shell
docker rm -f k8sta-dev-registry
```
:::

## Receiving webhooks

Making the K8sTA server visible to Docker Hub such that it can successfully
receive webhooks can be challenging. To help ease this process, our `Tiltfile`
has built-in support for exposing your local K8sTA server using
[ngrok](https://ngrok.com/). To take advantage of this:

1. [Sign up](https://dashboard.ngrok.com/signup) for a free ngrok account.

1. Follow ngrok
   [setup & installation instructions](https://dashboard.ngrok.com/get-started/setup)

1. Set the environment variable `ENABLE_NGROK_EXTENSION` to a value of `1`
   _before_ running `tilt up`.

1. After running `tilt up`, the option should become available in the Tilt UI at
  [http://localhost:10350/](http://localhost:10350/) to expose the K8sTA server
   using ngrok. After going so, the applicable ngrok URL will be displayed in
   the server's logs in the Tilt UI.

1. Configure any Docker Hub repository you own to deliver webhooks to
   `<ngrok URL>/dockerhub?access_token=insecure-dev-token`.

:::cation
We cannot guarantee that ngrok will work in all environments, especially if you
are behind a corporate firewall.
:::
