# K8sTA Contribution Guide

This documentation provides a very brief overview of:

* How to get a development environment for K8sTA set up quickly and easily.

* The requirements for signing your commits.

## The Development Environment

K8sTA is implemented in Go. For maximum productivity in your text editor or IDE,
it is recommended that you have installed the latest stable releases of Go and
applicable editor/IDE extensions, however, this is not strictly required to be
successful.

### Containerized Tests

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

### Working with Go Code

If you make modifications to Go code, it is recommended that you run
corresponding unit tests and linters before opening a PR.

To run all unit tests:

```shell
$ make test-unit
```

To run lint checks:

```shell
$ make lint
```

### Building & Pushing Docker Images from Source

You will rarely, if ever, need to directly / manually build Docker images from
source. This is because of tooling we use (see next section) that does this for
you. Unless you have a specific need for doing this, you can safely skip this
section.

In the event that you do need to manually build images from source you can
execute the same make targets that are used by CI and our release process, but
be advised that this involves
[multiarch builds using buildx](https://www.docker.com/blog/multi-arch-build-and-images-the-simple-way/).
This can be somewhat slow and is not guaranteed to be supported on all systems.

First, list all available builders:

```shell
$ docker buildx ls
```

You will require a builder that lists both `linux/amd64` and `linux/arm64` as
supported platforms. If one is present, select it using the following command:

```shell
$ docker buildx use <NAME/NODE>
```

If you do not have an adequate builder available, you can try to launch one:

```shell
$ docker buildx create --use 
```

Because buildx utilizes a build server, the images built will not be present
locally. (Even though your build server is running locally, it’s remote from the
perspective of your local Docker engine.) To make them available for use, you
must push them somewhere. The following environment variables give you control
over where the images are pushed to:

* `DOCKER_REGISTRY`: Host name of an OCI registry. If this is unset, Docker Hub
  is assumed.

* `DOCKER_ORG`: For multi-tenant registries, set this to a username or
  organization name for which you have permission to push images. This is not
  always required for private registries, but if you’re pushing to Docker Hub,
  for instance, you will want to set this.

If applicable, you MUST log in to whichever registry you are pushing images to
in advance.

The example below shows how to build and push it to Docker Hub:

```shell
$ DOCKER_ORG=<Docker Hub username or org name> make push
```

### Iterating Quickly

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

* [KinD](https://kind.sigs.k8s.io/#installation-and-usage): Runs
  development-grade Kubernetes clusters in Docker.

  > 📝&nbsp;&nbsp;If you strongly prefer [k3d](https://k3d.io), please consider
  > opening a PR to add support.

* [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it): Launches
  development-grade Kubernetes clusters (in KinD, for instance) that are
  pre-connected to a local image registry.

* [Tilt](https://docs.tilt.dev/#macoslinux): Builds components from source and
  deploys them to a development-grade Kubernetes cluster. More importantly, it
  enables developers to rapidly rebuild and replace running components with the
  click of a button.

* [Helm](https://helm.sh/docs/intro/install/): The package manager for
  Kubernetes. Tilt will use this to help deploy K8sTA from source.

Follow the installation instructions for each of the above.

> 🟢&nbsp;&nbsp;Once these tools are installed, you can be up and running with
> just two commands!

To launch a brand new Kind cluster pre-connected to a local image registry:

```shell
$ make hack-kind-up
```

Because K8sTA augments Argo CD, the above command will _also_ install a recent,
stable version of Argo CD in the cluster's `argocd` namespace.

To build and deploy K8sTA from source:

```shell
$ tilt up
```

> 📝&nbsp;&nbsp;Temporarily, the K8sTA Helm chart that Tilt installs also
> includes K8sTA configuration that, in the future, will be user-defined.

Tilt will also launch a web-based UI running at
[http://localhost:10350](http://localhost:10350). Visit this in your web browser
and you will be able to see the build and deployment status of each K8sTA
component.

> 📝&nbsp;&nbsp;Tilt is often configured to watch files and automatically
> rebuild and replace running components when their source code is changed. This
> is deliberately disabled for K8sTA since the Docker image takes long enough to
> build that it’s better to conserve system resources by only rebuilding when
> you choose. The web UI makes it easy to identify components whose source has
> been altered. They can be rebuilt and replaced with one mouse click.

When you are done with Tilt, interrupt the running `tilt up` process with
`ctrl + c`. Components _will remain running in the cluster_, but Tilt will no
longer be in control. If Tilt is restarted later, it will retake control of the
already-running components.

If you wish to undeploy everything Tilt has deployed for you, use `tilt down`.

To destroy your KinD cluster, use `make hack-kind-down`.

> 📝&nbsp;&nbsp;`make hack-kind-down` deliberately leaves your local registry
> running so that if you resume work later, you are doing so with a local
> registry that’s already primed with most layers of K8sTA’s image.
> If you wish to destroy the registry, use:
>
> ```shell
> $ docker rm -f k8sta-dev-registry
> ```

### Receiving Webhooks

Making the K8sTA server visible to Docker Hub such that it can successfully
receive webhooks can be challenging. To help ease this process, our Tiltfile has
built-in support for exposing your local K8sTA server using
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

> 🟡&nbsp;&nbsp;We cannot guarantee that ngrok will work in all environments,
> especially if you are behind a corporate firewall.

## Signing Your Commits

All commits merged into K8sTA's main branch MUST bear a DCO (Developer
Certificate of Origin) sign-off. This is a line placed at the end of a commit
message containing a contributor’s “signature.” In adding this, the contributor
certifies that they have the right to contribute the material in question.

Here are the steps to sign your work:

1. Verify the contribution in your commit complies with the
   [terms of the DCO](https://developercertificate.org/).

1. Add a line like the following to your commit message:

   ```
   Signed-off-by: Joe Smith <joe.smith@example.com>
   ```

   You MUST use your legal name – handles or other pseudonyms are not permitted.

   While you could manually add DCO sign-off to every commit, there is an easier
   way:

   1. Configure your git client appropriately. This is one-time setup.

      ```shell
      $ git config user.name <legal name>
      $ git config user.email <email address you use for GitHub>
      ```

      If you work on multiple projects that require a DCO sign-off, you can
      configure your git client to use these settings globally instead of only
      for K8sTA:

      ```shell
      $ git config --global user.name <legal name>
      $ git config --global user.email <email address you use for GitHub>
      ```

   1. Use the --signoff or -s (lowercase) flag when making each commit. For
      example:

      ```shell
      $ git commit --message "<commit message>" --signoff
      ```

      If you ever make a commit and forget to use the `--signoff` flag, you can
      amend your commit with this information before pushing:

      ```shell
      $ git commit --amend --signoff
      ```

   1. You can verify the above worked as expected using `git log`. Your latest
      commit should look similar to this one:

      ```shell
      Author: Joe Smith <joe.smith@example.com>
      Date:   Thu Feb 2 11:41:15 2018 -0800

      Update README

      Signed-off-by: Joe Smith <joe.smith@example.com>
      ```

      Notice the `Author` and `Signed-off-by` lines match. If they do not, the
      PR will be rejected by the automated DCO check.
