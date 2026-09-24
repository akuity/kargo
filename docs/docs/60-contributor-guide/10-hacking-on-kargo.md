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

:::info

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

1. Launch or re-use an existing local Kubernetes cluster.

    Any of the following options are viable:

    <Tabs groupId="cluster-start">
    <TabItem value="docker-desktop" label="Docker Desktop">

    If you are a
    [Docker Desktop](https://www.docker.com/products/docker-desktop/)
    user, you can follow
    [these instructions](https://docs.docker.com/desktop/kubernetes/) to enable
    its built-in Kubernetes support. If it's already enabled, you're ready to
    go.

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
   
    </TabItem>
    <TabItem value="orbstack" label="OrbStack">

    [OrbStack](https://orbstack.dev/) is a fast, lightweight, drop-in replacement
    for Docker Desktop for Mac OS only. You can follow
    [these instructions](https://docs.docker.com/desktop/kubernetes/) to enable
    its built-in Kubernetes support. If it's already enabled, you're ready to
    go.

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

    </TabItem>
    <TabItem value="kind" label="kind">

    If you have any Docker-compatible container runtime installed (including
    native Docker, Docker Desktop, or OrbStack), you can easily launch a
    disposable cluster to facilitate Kargo development using
    [kind](https://kind.sigs.k8s.io/#installation-and-usage). You do not need to
    install it in advance.

    The following `make` target will launch a kind cluster with a local image
    registry wired into it, various port-forwarding rules pre-configured, and
    Kargo's prerequisites installed:

    ```shell
    make hack-kind-up
    ```

    :::info

    The `hack-kind-up` target will ensure suitable versions of `kind` and
    [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it) (used for
    declarative kind configuration) are available. Both are pinned as `go tool`
    dependencies in the root `go.mod` and built on demand -- no manual
    installation is needed.

    :::

    :::info

    While this option is a bit more complex than using Docker Desktop or
    OrbStack directly, it offers the advantage of being fully-disposable. If
    your cluster reaches a state you are dissatisfied with, you can simply
    destroy it and launch a new one.
    :::

    </TabItem>
    <TabItem value="k3d" label="k3d">

    If you have any Docker-compatible container runtime installed (including
    native Docker, Docker Desktop, or OrbStack), you can easily launch a
    disposable cluster to facilitate Kargo development using
    [k3d](https://k3d.io). You do not need to install it in advance.

    The following `make` target will launch a kind cluster with a local image
    registry wired into it, various port-forwarding rules pre-configured, and
    Kargo's prerequisites installed:

    ```shell
    make hack-k3d-up
    ```

    :::info

    The `hack-k3d-up` target will ensure suitable versions of `k3d` and
    [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it) (used for
    declarative k3d configuration) are available. `k3d` is installed into
    `hack/bin/`, while `ctlptl` is pinned as a `go tool` dependency in the root
    `go.mod` and built on demand.

    :::

    :::info

    While this option is a bit more complex than using Docker Desktop or
    OrbStack directly, it offers the advantage of being fully-disposable. If
    your cluster reaches a state you are dissatisfied with, you can simply
    destroy it and launch a new one.
    :::

    </TabItem>
    </Tabs>

1. Optional: Configure and start a tunnel for the external webhooks server:

    If you are working on or testing the external webhooks server, you will
    benefit from configuring a tunnel from the outside world so that traffic
    originating from platforms like GitHub, Docker Hub, and others can reach the
    server. It is easy to accomplish this using [ngrok](https://ngrok.com/).
    If you wish to do so, you must first
    [install ngrok](https://ngrok.com/downloads) yourself.

    With `ngrok` installed, you can conveniently open a tunnel to
    `localhost:30083` (where the next step will run the external webhooks
    server) using:

    ```shell
    make hack-ngrok
    ```

    Allow this process to run while you are working on or testing the external
    webhooks server. Interrupt it with `ctrl + c` when you are done.

    If you have a paid ngrok account that allows you to use a custom domain name
    for your tunnels, you can specify that domain name using the
    `KARGO_EXTERNAL_WEBHOOKS_SERVER_HOSTNAME` and (to properly set the protocol
    scheme) `KARGO_EXTERNAL_WEBHOOKS_SERVER_TLS_TERMINATED_UPSTREAM` environment
    variables before running the make target:

    ```shell
    export KARGO_EXTERNAL_WEBHOOKS_SERVER_HOSTNAME=my-tunnel.ngrok.io
    export KARGO_EXTERNAL_WEBHOOKS_SERVER_TLS_TERMINATED_UPSTREAM=true
    make hack-ngrok
    ```

    If the `KARGO_EXTERNAL_WEBHOOKS_SERVER_HOSTNAME` environment variable is
    undefined, the tunnel will utilize a dynamically-generated subdomain of
    `ngrok.io`.

1. Build and deploy Kargo from source:

    [Tilt](https://docs.tilt.dev/#macoslinux) is a convenient tool that builds
    container images from source and seamlessly deploys them to a local
    Kubernetes cluster. More importantly, it enables developers to rapidly
    rebuild and replace running components with the click of a button. You do
    not need to install it in advance.

    ```shell
    make hack-tilt-up
    ```

    :::info

    The `hack-tilt-up` target will ensure the installation of a suitable
    versions of `tilt` and `helm` into `hack/bin/`.

    When run for the first time on a new development cluster, `make
    hack-tilt-up` will install suitable versions of
    [cert-manager](https://cert-manager.io/),
    [Argo CD](https://argoproj.github.io/cd/), and
    [Argo Rollouts](https://argoproj.github.io/rollouts/) into your cluster
    using `helm`.

    The Argo CD dashboard will be exposed at
    [localhost:30080](https://localhost:30080).

    The username and password are both `admin`.

    You may safely ignore any certificate warnings.
    :::

    Tilt will launch a web-based UI running at
    [http://localhost:10350](http://localhost:10350). Visit this in your web
    browser to view the build and deployment status of each Kargo component as
    well as the logs from each component.

    :::info

    Tilt compiles the back end automatically when source files change. Updating
    the running back-end components requires a manual trigger in the Tilt UI.
    The UI, local PostgreSQL resource, and database migrations update
    automatically. The web UI shows which components have pending changes.
    :::

    :::info

    If you specified a custom domain name for a tunnel to the external webhooks
    server in the previous step by defining a value for the
    `KARGO_EXTERNAL_WEBHOOKS_SERVER_HOSTNAME` environment variable, you should
    export the same value for that environment variable before running `make
    tilt up` as well:

    ```shell
    export KARGO_EXTERNAL_WEBHOOKS_SERVER_HOSTNAME=my-tunnel.ngrok.io
    make hack-tilt-up
    ```

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

1. When you are done with Tilt, interrupt the running `make hack-tilt-up`
   process with `ctrl + c`. Components _will remain running in the cluster_, but
   Tilt will no longer be in control. If Tilt is restarted later, it will retake
   control of the already-running components.

    If you wish to undeploy everything Tilt has deployed for you (except for
    prerequisites), use `make hack-tilt-down`.

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

## Working with PostgreSQL

Tilt starts a PostgreSQL instance for local development and forwards
`127.0.0.1:15432` to its port `5432`. The database, username, and password are
all `kargo`. Inside the cluster, the address is `kargo-postgres.kargo.svc:5432`.
Kargo's application components do not use this database yet.

The `db-migrate` Tilt resource waits for PostgreSQL to accept a connection,
then runs the pinned Goose tool to apply pending SQL migrations from
`db/migrations/`. It runs on startup and whenever that directory changes.
Migration failures appear in Tilt; failed migrations are not automatically
retried by the migration script. An empty directory is supported while the
initial schema is being developed.

To run the same migration command manually while Tilt is running:

```shell
make db-migrate
```

To inspect the database or run other Goose commands, configure your shell:

```shell
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING='postgres://kargo:kargo@127.0.0.1:15432/kargo?sslmode=disable'
export GOOSE_MIGRATION_DIR=db/migrations

go tool goose version
# Requires the PostgreSQL client to be installed locally:
psql "$GOOSE_DBSTRING"
```

`make db-migrate` uses these values by default. Set the same environment
variables before starting Tilt to override them, for example when using a
separate development database.

Tilt also configures the management controller's `DATABASE_URL` environment
variable using the in-cluster PostgreSQL address and waits for `db-migrate`
before the controller's initial startup. Configure `DATABASE_URL` separately
when using another database; the Goose variables only configure migrations.

### Generating database code

`sqlc.yaml` configures sqlc to read Goose migrations from `db/migrations` and
SQL query files from `db/queries`, then generate pgx/v5 Go code in `pkg/database`.
After adding or changing SQL, regenerate the code:

```shell
make codegen-db
```

The existing `make codegen` target includes this step. Generation does nothing
until `db/queries` contains a `.sql` query file, so database tooling can be set
up before adding a resource's schema and queries.

### Creating a migration

Create and edit migration drafts outside the watched directory. For example:

```shell
draft_dir=$(mktemp -d)
go tool goose -dir "$draft_dir" create create_widgets sql
```

Replace the generated SQL with:

```sql
-- +goose Up
CREATE TABLE widgets (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL
);

-- +goose Down
DROP TABLE widgets;
```

Once the migration is complete, move it into the watched directory:

```shell
mv "$draft_dir"/*.sql db/migrations/
```

Tilt applies it automatically. You can then inspect the result:

```shell
go tool goose status
psql "$GOOSE_DBSTRING" -c '\d widgets'
```

:::note

Creating a migration directly in `db/migrations/` can cause Tilt to apply the
generated template before you finish editing it. Goose tracks migration
versions; editing an already-applied file does not apply it again. Add a new
migration for subsequent schema changes.

:::

Rollback and reset are explicit operations. With the environment variables
above set, `go tool goose down` rolls back the latest migration, and
`go tool goose reset` rolls back all migrations. Both may delete data, depending
on the migrations' down statements. Run `make db-migrate` to reapply them.

PostgreSQL uses a persistent volume from the cluster's default StorageClass.
Data survives pod replacement, stopping/restarting Tilt, and
`make hack-tilt-down`: Tilt retains namespaces by default, and the StatefulSet
retains its volume claim when deleted. Deleting the `kargo` namespace or the
`data-kargo-postgres-0` volume claim removes this persistence; the StorageClass's
reclaim policy determines whether the backing volume is also deleted. Treat
this database as disposable development data.

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

If you wish to opt-out of executing code-generation within a container (for
performance reasons, perhaps), drop the `hack-` prefix from the target to run the docs natively on your system:

```shell
make serve-docs
```

This will require quite a variety of tools to be installed locally, so we do not
recommend this if you can avoid it.

:::
