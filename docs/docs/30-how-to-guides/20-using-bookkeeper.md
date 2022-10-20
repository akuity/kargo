---
description: Using Bookkeeper
---

# Using Bookkeeper

K8sTA's architecture separates components into two main areas of concern:

1. Support for the "rendered YAML branches" pattern. i.e. Using your preferred
   configuration management tool to render plain YAML from the default branch of
   your GitOps repository and storing that in an environment-specific branch.

2. Managing the progressive deployment of application changes through a series
   of environments (Argo CD Applications).

Apart from the the common motivations for separating concerns, the K8sTA team
chose to do this because the first problem is better understood and easier to
solve than the second. The separation enables the team to realize value from the
components that address the first problem _independently of the rest of K8sTA_.
We call those components "Bookkeeper" and the rest of this page covers exactly
how to use Bookkeeper by itself, without the rest of K8sTA.

## Repository layout

Bookkeeper relies heavily on the principle of
[convention over configuration](https://en.wikipedia.org/wiki/Convention_over_configuration)
and supports three configuration management tools: `kustomize`, `ytt`, and
`helm`.

Regardless of which configuration management tool you select, Bookkeeper imposes
one consistent convention: That common configuration resides in a `base/`
directory at the root of your repository's default branch, while any
environment-specific configuration resides in a directory whose path (relative
to the root of the repository) matches the name of the environment-specific
branch in which that configuration will be stored once rendered.

:::note
This means branch names will include slashes if the corresponding paths are
multiple levels deep.
:::

### kustomize

To use Bookkeeper with `kustomize`, the default branch of your repository should
be laid out as follows:

```plain
.
├── base
│  ├── <manifests>             # plain YAML
│  └── kustomization.yaml      # references manifests
└── env                        # "env" is suggested, but not required; this could also be many levels deep
    ├── <env 1 name>
    │   ├── <manifests>        # supplements base manifests
    │   └── kustomization.yaml # references base and manifests; patches base manifests
    ├── <env 2 name>
    │   ├── <manifests>        # supplements base manifests
    │   └── kustomization.yaml # references base and manifests; patches base manifests
    │
    ├── ...
    │
    └── <env n name>
        ├── <manifests>        # supplements base manifests
        └── kustomization.yaml # references base and manifests; patches base manifests
```

This layout is thought to be compatible with typical use of `kustomize`.

`kustomize` is Bookkeeper's default configuration management tool, so no
additional configuration is required.

:::info
To render configuration for an environment-specific branch, Bookkeeper will
execute `kustomize build` from within the corresponding directory.
:::

Refer directly to
[kustomize's documentation](https://kubectl.docs.kubernetes.io/references/kustomize/)
for more information.

### ytt

To use Bookkeeper with `ytt`, the default branch of your repository should be
laid out as follows:

```plain
.
├── Bookfile.yaml
├── base
│   └── <manifest templates>
└── env                        # "env" is suggested, but not required; this could also be many levels deep
    ├── <env 1 name>
    │   └── <files>            # fills in blanks in base and/or patches base
    ├── <env 2 name>
    │   └── <files>            # fills in blanks in base and/or patches base
    │
    ├── ...
    │
    └── <env n name>
        └── <files>            # fills in blanks in base and/or patches base
```

This layout is thought to be compatible with typical use of `ytt`.

`ytt` is not Bookkeeper's default configuration management tool, so a small bit
of configuration in `Bookfile.yaml` is required to specify the use of `ytt`
across all branches:

```yaml
defaultBranchConfig:
  configManagement:
    ytt: {}
```

:::info
To render configuration for an environment-specific branch, Bookkeeper will
execute a command resembling the following from the root of the repository:

```shell
ytt --file base/ --file env/<env name>/
```
:::

Refer directly to [ytt's documentation](https://carvel.dev/ytt/) for more
information.

### helm

To use Bookkeeper with `helm`, the default branch of your repository should be
laid out as follows:

```plain
.
├── Bookfile.yaml
├── base                       # a proper Helm chart
│   ├── Chart.yaml
│   ├── templates
│   │   └── <manifest templates>
│   └── values.yaml            # default values
└── env                        # "env" is suggested, but not required; this could also be many levels deep
    ├── <env 1>
    │   └── values.yaml        # supplement or override base values
    ├── <env 2>
    │   └── values.yaml        # supplement or override base values
    │
    ├── ...
    │
    └── <env n>
        └── values.yaml        # supplement or override base values
```

This layout is thought to be compatible with typical use of `helm` in a GitOps
context.

`helm` is not Bookkeeper's default configuration management tool, so a small bit
of configuration in `Bookfile.yaml` is required to specify the use of `helm`
across all branches. When using `helm`, it is also required to specify a release
name:

```yaml
defaultBranchConfig:
  configManagement:
    helm:
      releaseName: <release name>
```

:::info
To render configuration for an environment-specific branch, Bookkeeper will
execute a command resembling the following from the root of the repository:

```shell
helm template <release name> base/ --values env/<env name>/values.yaml
```
:::

Refer directly to [helm's documentation](https://helm.sh/docs/) for more
information.

## The Bookkeeper CLI

### Installing the CLI

The `bookkeeper` CLI can be downloaded directly from the K8sTA
[releases page](https://github.com/akuityio/k8sta-prototype/releases).

:::note
You may need to update permissions on the downloaded binary to enable your
system to execute it. You should also consider renaming the binary to
`bookkeeper` and moving it to a location on your `PATH`.
:::

### Enabling the server

Bookkeeper has dependencies on `git`, `kustomize`, `helm`, and `ytt`. Rather
than requiring its users to have compatible versions of those dependencies, the
`bookkeeper` offloads all work to a Bookkeeper server.

The rest of K8sTA utilizes Bookkeeper packages directly and does not require a
Bookkeeper server to be running in order to function. Because of this, the
Bookkeeper server is not enabled by default when installing K8sTA and needs to
be explicitly enabled if you wish to use the CLI. To do this, set
`bookkeeper.server.enabled` to `true` during `helm install` or
`helm upgrade` of K8sTA. The documentation on
[Installing K8sTA](./10-installing-k8sta.md) covers this and other relevant
installation options in greater detail.

### Basic CLI usage

Once your Bookkeeper server is running, rendering configuration using the
`bookkeeper render` command is straightforward. The required inputs for the
command, each with their own flag, include:

* `--server`: The address of the Bookkeeper server

  :::note
  If you use the `bookkeeper` CLI frequently, you can specify a persistent
  server address using the `BOOKKEEPER_SERVER` environment variable.
  :::

* `--repo`: The URL for cloning your GitOps repository

* `--repo-username`: Username for reading from and writing to your GitOps
  repository

* `--repo-password`: Password (or personal access token) for reading from and
  writing to your GitOps repository

  :::caution
  In some contexts, it is not secure to specify a password or token directly in
  a command. To accommodate such circumstances, the repository password can also
  be specified using the `BOOKKEEPER_REPO_PASSWORD` environment variable. Since
  username and password are complementary pieces of information, the repository
  username may also be specified in this manner, using the
  `BOOKKEEPER_REPO_USERNAME` environment variable.
  
  The above may be especially relevant if incorporating the `bookkeeper` CLI
  into any sort of automated processes. 
  :::

* `--target-branch`: The environment-specific branch in which to store rendered
  configuration
  
  :::info
  Bookkeeper also uses this value to locate the environment-specific paths in
  your repository's default branch.
  :::

Example usage:

```shell
bookkeeper render \
  --server https://bookkeeper.example.com \
  --repo https://github.com/<your GitHub handle>/bookkeeper-demo-deploy \
  --repo-username <your GitHub handle> \
  --repo-password <a GitHub personal access token> \
  --target-branch env/dev
```

### Advanced CLI usage

This section covers additional `bookkeeper render` options that are relevant for
certain scenarios.

* Use `--commit` followed by a commit ID (sha) if you wish to specify a commit
  in your GitOps repository for Bookkeeper to render from (instead of defaulting
  to the commit at the head of the default branch).
  
* Use `--image` to specify a new version of a Docker image to replace an older
  version of the same image. As a convenience, Bookkeeper can accomplish this
  itself in the "last mile" of configuration rendering, regardless of which
  configuration management tool you use and regardless of whether that tool
  supports similar functionality. This option can be specified multiple times if
  your configuration references multiple images.

* Use `--pr` to specify that Bookkeeper should open a pull request against the
  target branch instead of committing directly to it.

  :::note
  At present, this feature only works for GitHub repositories. It does not work
  with other common Git providers, nor does it work with GitHub enterprise.

  Support for Azure DevOps, Bitbucket, GitLab, and GitHub Enterprise is planned.
  :::

  :::note
  This option will result in an error if the target branch does not exist, as
  PRs cannot be opened against non-existent branches.
  :::

## The Bookkeeper action

If you are integrating Bookkeeper into automated processes that are implemented
via GitHub Actions, Bookkeeper can be run as an action. Unlike the `bookkeeper`
CLI, the Bookkeeper action does not require a Bookkeeper server to support it.

:::info
The Bookkeeper action utilizes the official K8sTA Docker image and therefore has
guaranteed access to compatible versions of dependencies like `git`, `helm`,
`kustomize`, and `ytt`, which are included on that image. This is what obviates
the need to offload rendering to a Bookkeeper server.

Because it does not require a Bookkeeper server, the Bookkeeper action is an
inherently lighter weight and superior option to the `bookkeeper` CLI and should
be favored wherever possible.
:::

### Installing the action

:::info
As of this writing, GitHub Actions does not have good support for _private_
actions. This being the case, some extra setup is currently required in order to
use the Bookkeeper action.
:::

Paste the following YAML, verbatim into `.github/actions/bookkeeper` in your
GitOps repository:

```yaml
name: 'Bookkeeper'
description: 'Publish rendered config to an environment-specific branch'
inputs:
  personalAccessToken:
    description: 'A personal access token that allows Bookkeeper to write to your repository'
    required: true
  commitSHA:
    description: 'The ID of the commit from which you want to render configuration'
    required: true
  targetBranch:
    description: 'The environment-specific branch for which you want to render configuration'
    required: true
  openPR:
    description: 'Whether to open a PR instead of committing directly to the target branch'
    required: false
    default: 'false'
runs:
  using: 'docker'
  image: 'krancour/mystery-image:v0.1.0-alpha.2'
  entrypoint: 'bookkeeper-action'
```

:::note
The odd-looking reference to a Docker image named
`krancour/mystery-image:v0.1.0-alpha.2` is not a mistake. As previously
noted, GitHub support for private actions is very poor. Among other things, this
means there is no method of authenticating to a Docker registry to pull private
images. `krancour/mystery-image:v0.1.0-alpha.2` is a public copy of the
official K8sTA image. We hope that its obscure name prevents it from attracting
much notice.
:::

### Using the action

Because the action definition exists within your own repository (see previous
section), you must utilize
[actions/checkout](https://github.com/marketplace/actions/checkout) to ensure
that definition is available during the execution of your workflow. After doing
so, the Bookkeeper action is as easy to use as if it had been sourced from the
GitHub Actions Marketplace.

:::info
In the future, this step will not be required.
:::

Example usage:

```yaml
jobs:
  render-dev-manifests:
    name: Render dev manifests
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Render manifests
        uses: ./.github/actions/bookkeeper/
        with:
          personalAccessToken: ${{ secrets.GITHUB_TOKEN }}
          commitSHA: ${{ github.sha }}
          targetBranch: env/dev
```

In the example `render-dev-manifests` job above, you can see that rendering
configuration into an environment-specific branch requires little more than
specifying the commit to render configuration from, providing a token, and
specifying the branch name. The action takes care of the rest.

:::caution
`github.sha` is the ID of the commit that triggered the workflow. Depending on
what your workflow looks like, this may or may not be the right value to provide
for the `commitSHA` input. For instance, this value would be incorrect if your
workflows "cascade" and rendering environment-specific configuration into a
given branch is triggered by merge/push of environment-specific configuration
into some logically "preceding" environment-specific branch. In such a case
`github.sha` will point to a commit in an environment-specific branch and
_won't_ point to the commit in the default branch from which configuration
should be rendered.

Bookkeeper has no convenient answer yet for how to determine the correct
`commitSHA` value to use for cases such as this one, but this _will_ be
addressed in an upcoming release.
:::

:::note
`secrets.GITHUB_TOKEN` is automatically available in every GitHub Actions
workflow and should have sufficient permissions to both read from and write to
your repository.
:::

## The K8sTA image

The official K8sTA Docker image contains a "thick" variant of the Bookkeeper CLI
that does not require a Bookkeeper server to support it.

If you are integrating Bookkeeper into automated processes that are implemented
with something other than GitHub Actions and those processes permit execution of
commands within a Docker container (much as a GitHub action does), then
utilizing the K8sTA image and this variant of the CLI is a convenient option.

The thick CLI's interface is identical to that of the thin CLI's except that all
server-related flags are absent. Example usage equivalent to the thin client
example, therefore, resembles that example with the `--server` flag omitted:

```shell
docker run -it ghcr.io/akuityio/k8sta-prototype:v0.1.0-alpha.2 \
  bookkeeper render \
  --repo https://github.com/<your GitHub handle>/bookkeeper-demo-deploy \
  --repo-username <your GitHub handle> \
  --repo-password <a GitHub personal access token> \
  --target-branch env/dev
```

:::tip
Although the exact procedure for emulating the example above with vary from one
automation platform to the next, the K8sTA image and this variant of the CLI
should permit you to integrate Bookkeeper with a broad range of automation
platforms including, but not limited to, popular choices such as
[CircleCI](https://circleci.com/) or [Travis CI](https://www.travis-ci.com/).
:::

:::caution
This variant of the Bookkeeper CLI is not designed to be run anywhere except
within a container based on the official K8sTA image.
:::
