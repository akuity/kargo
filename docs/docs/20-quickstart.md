---
description: Learn about Kargo by progressing a change through multiple stages in a local Kubernetes cluster
sidebar_label: Quickstart
---

# Kargo Quickstart

This guide presents a basic introduction to Kargo. Together, we will:

1. Install Kargo and its dependencies into an existing, local Kubernetes
   cluster.

    OR

    Create a new local Kubernetes cluster with Kargo and its dependencies
    already installed.

1. Install the Kargo CLI.

1. Demonstrate how Kargo can progress changes through multiple stages by
   interacting with your GitOps repository and Argo CD `Application` resources.

1. Clean up.

:::info
If you're looking to contribute to Kargo, you may wish to consult the
[contributor guide](./contributor-guide) instead.
:::

### Starting a Local Cluster

Any of the following approaches require [Helm](https://helm.sh/docs/) v3.13.1 or
greater to be installed.

<Tabs groupId="cluster-start">
<TabItem value="docker-desktop" label="Docker Desktop">

If you are a
[Docker Desktop](https://www.docker.com/products/docker-desktop/)
user, you can follow
[these instructions](https://docs.docker.com/desktop/kubernetes/) to enable
its built-in Kubernetes support.

:::info
Although this is one of the fastest paths to a local Kubernetes cluster, be
aware that Docker Desktop supports only a _single_ Kubernetes cluster. If
that cluster reaches a state you are dissatisfied with, resetting it will
remove not just Kargo-related resources, but _all_ your workloads and data.
:::

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/install.sh | sh
```

</TabItem>
<TabItem value="orbstack" label="OrbStack">

[OrbStack](https://orbstack.dev/) is a fast, lightweight, drop-in replacement
for Docker Desktop for Mac OS only. You can follow
[these instructions](https://docs.orbstack.dev/kubernetes/) to enable its
built-in Kubernetes support.

:::info
Although this is one of the fastest paths to a local Kubernetes cluster, be
aware that OrbStack supports only a _single_ Kubernetes cluster. If
that cluster reaches a state you are dissatisfied with, resetting it will
remove not just Kargo-related resources, but _all_ your workloads and data.
:::

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/install.sh | sh
```

</TabItem>
<TabItem value="kind" label="kind">

If you have any Docker-compatible container runtime installed (including native
Docker, Docker Desktop, or OrbStack), you can easily launch a disposable cluster
just for this quickstart using
[kind](https://kind.sigs.k8s.io/#installation-and-usage).

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/kind.sh | sh
```

:::info
While this option is a bit more complex than using Docker Desktop or OrbStack
directly, it offers the advantage of being fully-disposable. If your cluster
reaches a state you are dissatisfied with, you can simply destroy it and
launch a new one.
:::

</TabItem>
<TabItem value="k3d" label="k3d">

If you have any Docker-compatible container runtime installed (including native
Docker, Docker Desktop, or OrbStack), you can easily launch a disposable cluster
just for this quickstart using [k3d](https://k3d.io).

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/k3d.sh | sh
```

:::info
While this option is a bit more complex than using Docker Desktop or OrbStack
directly, it offers the advantage of being fully-disposable. If your cluster
reaches a state you are dissatisfied with, you can simply destroy it and
launch a new one.
:::

</TabItem>
<TabItem value="more-info" label="More Info">

:::info
If you are averse to piping a downloaded script directly into a shell, please
feel free to download the applicable script and inspect its contents prior to
execution.

Any approach you select should only:

1. Launch a new, local Kubernetes cluster, if applicable
1. Install cert-manager
1. Install Argo CD
1. Install Argo Rollouts
1. Install Kargo
:::

</TabItem>
</Tabs>

:::note
If Kargo installation fails with a `401`, verify that you are using Helm v3.13.1
or greater.

If Kargo installation fails with a `403`, it is likely that Docker is configured
to authenticate to `ghcr.io` with an expired token. The Kargo chart and images
are accessible anonymously, so this issue can be resolved simply by logging out:

```shell
docker logout ghcr.io
```
:::

At the end of this process:

* The Argo CD dashboard will be accessible at [localhost:31443](https://localhost:31443).

  The username and password are both `admin`.

* The Kargo dashboard will be accessible at [localhost:31444](https://localhost:31444).

  The admin password is `admin`.

* You can safely ignore all cert errors for both of the above.

### Installing the Kargo CLI

<Tabs groupId="os">
<TabItem value="general" label="Mac, Linux, or WSL" default>

To download the Kargo CLI:

```shell
arch=$(uname -m)
[ "$arch" = "x86_64" ] && arch=amd64
curl -L -o kargo https://github.com/akuity/kargo/releases/download/v0.8.0-rc.1/kargo-"$(uname -s | tr '[:upper:]' '[:lower:]')-${arch}"
chmod +x kargo
```

Then move `kargo` to a location in your file system that is included in the
value of your `PATH` environment variable.

</TabItem>
<TabItem value="windows" label="Windows Powershell">

To download the Kargo CLI:

```shell
Invoke-WebRequest -URI https://github.com/akuity/kargo/releases/download/v0.8.0-rc.1/kargo-windows-amd64.exe -OutFile kargo.exe
```

Then move `kargo.exe` to a location in your file system that is included in the value
of your `PATH` environment variable.

</TabItem>
</Tabs>

## Trying It Out

### Create a GitOps Repository

Let's begin by creating a repository on GitHub to house variations of our
application manifests for three different stages of a sample application: test,
UAT, and production.

1. Visit https://github.com/akuity/kargo-demo and fork the repository into your
   own GitHub account.

1. You can explore the repository and see that the `main` branch contains common
   configuration in a `base/` directory as well as stage-specific overlays in
   paths of the form `stages/<stage name>/`.

    :::note
    This layout is typical of a GitOps repository using
    [Kustomize](https://kustomize.io/) for configuration management and is not
    at all Kargo-specific.

    Kargo also works just as well with [Helm](https://helm.sh).
    :::

1. We'll be using it later, so save the location of your GitOps repository in an
   environment variable:

   ```shell
   export GITOPS_REPO_URL=<your repo URL, starting with https://>
   ```

### Create Argo CD `Application` Resources

In this step, we will use an Argo CD `ApplicationSet` resource to create and
manage three Argo CD `Application` resources that deploy the sample application
at three different stages of its lifecycle, with three slightly different
configurations, to three different namespaces in our local cluster:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: kargo-demo
  namespace: argocd
spec:
  generators:
  - list:
      elements:
      - stage: test
      - stage: uat
      - stage: prod
  template:
    metadata:
      name: kargo-demo-{{stage}}
      annotations:
        kargo.akuity.io/authorized-stage: kargo-demo:{{stage}}
    spec:
      project: default
      source:
        repoURL: ${GITOPS_REPO_URL}
        targetRevision: stage/{{stage}}
        path: stages/{{stage}}
      destination:
        server: https://kubernetes.default.svc
        namespace: kargo-demo-{{stage}}
      syncPolicy:
        syncOptions:
        - CreateNamespace=true
EOF
```

If you visit [your Argo CD dashboard](https://localhost:31443), you will notice
all three Argo CD `Application`s have not yet synced because they're not
configured to do so automatically, and in fact, the branches referenced by their
`targetRevision` fields do not even exist yet.

:::info
Our three stages all existing in a single cluster is for the sake of expediency.
A single Argo CD control plane can manage multiple clusters, so these could just
as easily have been spread across multiple clusters.
:::

### Hands on with the Kargo CLI

Up to this point, we haven't done anything with Kargo -- in fact everything
we've done thus far should be familiar to anyone who's already using Argo CD and
Kustomize. Now it's time to see what Kargo can do!

To get started, you will need a GitHub
[personal access token](https://github.com/settings/tokens)
with adequate permissions to read from and write to the repository you forked in
the previous section.

1. Save your GitHub handle and your personal access token in environment
   variables:

   ```shell
   export GITHUB_USERNAME=<your github handle>
   export GITHUB_PAT=<your personal access token>
   ```

1. Then, log into Kargo:

    ```shell
    kargo login https://localhost:31444 \
      --admin \
      --password admin \
      --insecure-skip-tls-verify
    ```

1. Next, we'll create several Kargo resource:

    1. A `Project`, which, when reconciled, will effect all boilerplate project
       initialization, including the creation of a specially-labeled `Namespace`
       with the same name as the `Project`

    1. A `Secret` containing credentials for our GitOps repository

    1. A `Warehouse` that subscribes to a container image repository

    1. Three `Stage` resources -- `test`, `uat`, and `prod`

    ```shell
    cat <<EOF | kargo apply -f -
    apiVersion: kargo.akuity.io/v1alpha1
    kind: Project
    metadata:
      name: kargo-demo
    ---
    apiVersion: v1
    kind: Secret
    type: Opaque
    metadata:
      name: kargo-demo-repo
      namespace: kargo-demo
      labels:
        kargo.akuity.io/cred-type: git
    stringData:
      repoURL: ${GITOPS_REPO_URL}
      username: ${GITHUB_USERNAME}
      password: ${GITHUB_PAT}
    ---
    apiVersion: kargo.akuity.io/v1alpha1
    kind: Warehouse
    metadata:
      name: kargo-demo
      namespace: kargo-demo
    spec:
      subscriptions:
      - image:
          repoURL: public.ecr.aws/nginx/nginx
          semverConstraint: ^1.26.0
          discoveryLimit: 5
    ---
    apiVersion: kargo.akuity.io/v1alpha1
    kind: Stage
    metadata:
      name: test
      namespace: kargo-demo
    spec:
      requestedFreight:
      - origin:
          kind: Warehouse
          name: kargo-demo
        sources:
          direct: true
      promotionMechanisms:
        gitRepoUpdates:
        - repoURL: ${GITOPS_REPO_URL}
          writeBranch: stage/test
          kustomize:
            images:
            - image: public.ecr.aws/nginx/nginx
              path: stages/test
        argoCDAppUpdates:
        - appName: kargo-demo-test
          appNamespace: argocd
    ---
    apiVersion: kargo.akuity.io/v1alpha1
    kind: Stage
    metadata:
      name: uat
      namespace: kargo-demo
    spec:
      requestedFreight:
      - origin:
          kind: Warehouse
          name: kargo-demo
        sources:
          stages:
          - test
      promotionMechanisms:
        gitRepoUpdates:
        - repoURL: ${GITOPS_REPO_URL}
          writeBranch: stage/uat
          kustomize:
            images:
            - image: public.ecr.aws/nginx/nginx
              path: stages/uat
        argoCDAppUpdates:
        - appName: kargo-demo-uat
          appNamespace: argocd
    ---
    apiVersion: kargo.akuity.io/v1alpha1
    kind: Stage
    metadata:
      name: prod
      namespace: kargo-demo
    spec:
      requestedFreight:
      - origin:
          kind: Warehouse
          name: kargo-demo
        sources:
          stages:
          - uat
      promotionMechanisms:
        gitRepoUpdates:
        - repoURL: ${GITOPS_REPO_URL}
          writeBranch: stage/prod
          kustomize:
            images:
            - image: public.ecr.aws/nginx/nginx
              path: stages/prod
        argoCDAppUpdates:
        - appName: kargo-demo-prod
          appNamespace: argocd
    EOF
    ```
    
  :::info
  Kargo uses [semver](https://github.com/masterminds/semver#checking-version-constraints) to handle semantic versioning constraints.
  :::

1. Use the CLI to view our `Warehouse` resource:

    ```shell
    kargo get warehouses --project kargo-demo
    ```

    Sample output:

    ```shell
    NAME         SHARD   AGE
    kargo-demo           13s
    ```

1. Use the CLI to view our three `Stage` resources:

    ```shell
    kargo get stages --project kargo-demo
    ```

    Sample output:

    ```shell
    NAME   SHARD   CURRENT FREIGHT   HEALTH   PHASE           AGE
    prod                                      NotApplicable   20s
    test                                      NotApplicable   20s
    uat                                       NotApplicable   20s
    ```

1. After a few seconds, our `Warehouse`, which subscribes to the
   `public.ecr.aws/nginx/nginx` container image, also should have already
   produced `Freight`:

    ```shell
    kargo get freight --project kargo-demo
    ```

    Sample output:

    ```shell
    NAME                                       ALIAS               ORIGIN                 AGE
    7a6e91f2d26aa84faadfdc340437bf84a0cc577a   pining-rottweiler   Warehouse/kargo-demo   49s
    ```
    :::info
    `Freight` is a set of references to one or more versioned artifacts, which
    may include:

      * Container images (from image repositories)

      * Kubernetes manifests (from Git repositories)

      * Helm charts (from chart repositories)

    This introductory example has `Freight` that references only a specific
    version of the `public.ecr.aws/nginx/nginx` container image.
    :::

1. We'll use it later, so save the ID of the `Freight` to an environment
   variable:

    ```shell
    export FREIGHT_ALIAS=$(kargo get freight --project kargo-demo --output jsonpath={.alias})
    ```

1. Now, let's _promote_ the `Freight` into the `test` `Stage`:

    ```shell
    kargo promote --project kargo-demo --freight-alias $FREIGHT_ALIAS --stage test
    ```

    Sample output:

    ```shell
    promotion.kargo.akuity.io/test.01j2wbtrym4r3tktv38qzteh5h.7a6e91f promotion created
    ```

    Query for `Promotion` resources within our project to see one has been
    created:

    ```shell
    kargo get promotions --project kargo-demo
    ```

    Our `Promotion` may briefly appear to be in a `Pending` phase, but more than
    likely, it will almost immediately be `Running`, or even `Succeeded`:

    ```shell
    NAME                                      SHARD   STAGE   FREIGHT                                    PHASE       AGE
    test.01j2wbtrym4r3tktv38qzteh5h.7a6e91f           test    7a6e91f2d26aa84faadfdc340437bf84a0cc577a   Succeeded   57s
    ```

    Once the `Promotion` has succeeded, we can again view all `Stage` resources
    in our project, and at a glance, see that the `test` `Stage` is now either
    in a `Progressing` or `Healthy` state.

    ```shell
    kargo get stages --project kargo-demo
    ```

    Sample output:

    ```shell
    NAME   SHARD   CURRENT FREIGHT                            HEALTH    PHASE           AGE
    prod                                                                NotApplicable   3m43s
    test           7a6e91f2d26aa84faadfdc340437bf84a0cc577a   Healthy   Steady          3m43s
    uat                                                                 NotApplicable   3m43s
    ```

    We can repeat the command above until our `test` `Stage` is in a `Healthy`
    state and we can further validate the success of this entire process by
    visiting the test instance of our site at
    [localhost:30081](http://localhost:30081).

    If we once again view the `status` of our `test` `Stage` in more detail, we
    will see that it now reflects its current `Freight`, and the history of all
    `Freight` that have passed through this stage. (The collection is ordered
    most to least recent.)

    ```shell
    kargo get stage test --project kargo-demo --output jsonpath-as-json={.status}
    ```

    Sample output:

    ```shell
    [
        {
            "freightHistory": [
                {
                    "id": "a86e8ac937e6029cbb41b336ececbf490408dacd",
                    "items": {
                        "Warehouse/kargo-demo": {
                            "images": [
                                {
                                    "digest": "sha256:c31bd632509dc8023d68f69a17515a6b830222f948d28f8cd4e279f3b6f4d1ed",
                                    "repoURL": "public.ecr.aws/nginx/nginx",
                                    "tag": "1.27.0"
                                }
                            ],
                            "name": "7a6e91f2d26aa84faadfdc340437bf84a0cc577a",
                            "origin": {
                                "kind": "Warehouse",
                                "name": "kargo-demo"
                            }
                        }
                    }
                }
            ],
            "health": {
                "argoCDApps": [
                    {
                        "healthStatus": {
                            "status": "Healthy"
                        },
                        "name": "kargo-demo-test",
                        "namespace": "argocd",
                        "syncStatus": {
                            "revision": "516ab40cb2b3da9b0a6346d37357905ec05336d5",
                            "status": "Synced"
                        }
                    }
                ],
                "status": "Healthy"
            },
            "lastPromotion": {
                "finishedAt": "2024-07-15T23:32:23Z",
                "freight": {
                    "images": [
                        {
                            "digest": "sha256:c31bd632509dc8023d68f69a17515a6b830222f948d28f8cd4e279f3b6f4d1ed",
                            "repoURL": "public.ecr.aws/nginx/nginx",
                            "tag": "1.27.0"
                        }
                    ],
                    "name": "7a6e91f2d26aa84faadfdc340437bf84a0cc577a",
                    "origin": {
                        "kind": "Warehouse",
                        "name": "kargo-demo"
                    }
                },
                "name": "test.01j2wbtrym4r3tktv38qzteh5h.7a6e91f",
                "status": {
                    "finishedAt": "2024-07-15T23:32:23Z",
                    "freight": {
                        "images": [
                            {
                                "digest": "sha256:c31bd632509dc8023d68f69a17515a6b830222f948d28f8cd4e279f3b6f4d1ed",
                                "repoURL": "public.ecr.aws/nginx/nginx",
                                "tag": "1.27.0"
                            }
                        ],
                        "name": "7a6e91f2d26aa84faadfdc340437bf84a0cc577a",
                        "origin": {
                            "kind": "Warehouse",
                            "name": "kargo-demo"
                        }
                    },
                    "freightCollection": {
                        "id": "a86e8ac937e6029cbb41b336ececbf490408dacd",
                        "items": {
                            "Warehouse/kargo-demo": {
                                "images": [
                                    {
                                        "digest": "sha256:c31bd632509dc8023d68f69a17515a6b830222f948d28f8cd4e279f3b6f4d1ed",
                                        "repoURL": "public.ecr.aws/nginx/nginx",
                                        "tag": "1.27.0"
                                    }
                                ],
                                "name": "7a6e91f2d26aa84faadfdc340437bf84a0cc577a",
                                "origin": {
                                    "kind": "Warehouse",
                                    "name": "kargo-demo"
                                }
                            }
                        }
                    },
                    "phase": "Succeeded"
                }
            },
            "observedGeneration": 1,
            "phase": "Steady"
        }
    ]
    ```

1. If we look at our `Freight` in greater detail, we'll see that by virtue of
   the `test` `Stage` having achieved a `Healthy` state, the `Freight` is now
   _verified_ in `test`, which designates it as eligible for promotion to the
   next `Stage` -- in our case, `uat`.

    :::note
    Although this example does not demonstrate it, it is also possible to verify
    the `Freight` in a `Stage` using user-defined processes. See the
    [relevant section](./15-concepts.md#verifications) of the concepts page to
    learn more.
    :::

    ```shell
    kargo get freight --alias $FREIGHT_ALIAS --project kargo-demo --output jsonpath-as-json={.status}
    ```

    Sample output:

    ```shell
    [
        {
            "verifiedIn": {
                "test": {}
            }
        }
    ]
    ```

### Behind the Scenes

So what has Kargo done behind the scenes?

Visiting our fork of https://github.com/akuity/kargo-demo, we will see that
Kargo has recently created a `stage/test` branch for us. It has taken the latest
manifests from the `main` branch as a starting point, run `kustomize edit set image`
within the `stages/test/` directory, and written the modified configuration to a
stage-specific branch -- the same branch referenced by the `test` Argo CD
`Applicaton`'s `targetRevision` field.

:::info
Although not strictly required for all cases, using stage-specific branches is a
suggested practice that enables Kargo to transition each `Stage` into any new or
previous state, at any time, with a new commit that replaces the entire contents
of the branch -- all without disrupting the `main` branch.
:::

### Promote to UAT and then Production

Unlike our `test` `Stage`, which subscribes directly to an image repository,
our `uat` and `prod` `Stage`s both subscribe to other, _upstream_ `Stage`s,
thereby forming a _pipeline_:

1. `uat` subscribes to `test`
1. `prod` subscribes to `uat`.

We leave it as an exercise to the reader to use the `kargo promote` command
to progress the `Freight` from `test` to `uat` and again from `uat` to `prod`.

:::info
The `uat` and `prod` instances of our site should be accessible at:

* `uat`: [localhost:30082](http://localhost:30082)
* `prod`: [localhost:30083](http://localhost:30083)
:::

:::info
It is possible to automate promotion of new, qualified `Freight` for designated
`Stage`s and also possible to used RBAC to limit who can trigger manual
promotions for each `Stage`, however, both these topics are beyond the scope of
this introduction.
:::

## Cleaning up

Congratulations! You've just gotten hands on with Kargo for the first time!

Now let's clean up!

<Tabs groupId="cluster-start">
<TabItem value="docker-desktop" label="Docker Desktop">

Docker Desktop supports only a _single_ Kubernetes cluster. If you are
comfortable deleting not just just Kargo-related resources, but _all_ your
workloads and data, the cluster can be reset from the Docker Desktop
Dashboard.

If, instead, you wish to preserve non-Kargo-related workloads and data, you
will need to manually uninstall Kargo and its prerequisites:

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/uninstall.sh | sh
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
will need to manually uninstall Kargo and its prerequisites:

```shell
curl -L https://raw.githubusercontent.com/akuity/kargo/main/hack/quickstart/uninstall.sh | sh
```

</TabItem>
<TabItem value="kind" label="kind">

Simply destroy the cluster:

```shell
kind delete cluster --name kargo-quickstart
```

</TabItem>
<TabItem value="k3d" label="k3d">

Simply destroy the cluster:

```shell
k3d cluster delete kargo-quickstart
```

</TabItem>
</Tabs>
