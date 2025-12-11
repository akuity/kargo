---
sidebar_label: Artifactory
---

# Artifactory Webhook Receiver

The Artifactory Webhook Receiver responds to `pushed` events originating from 
Artifactory repositories by _refreshing_ all `Warehouse` resources subscribed to 
those repositories.

:::warning

This webhook receiver does not respond to events where `domain` is `artifact`
and `event_type` is `deployed`.

:::

:::info

"Refreshing" a `Warehouse` resource means enqueuing it for immediate
reconciliation by the Kargo controller, which will execute the discovery of
new artifacts from all repositories to which that `Warehouse` subscribes.

:::

## Self-Hosted Artifactory

:::info

If you are not using a self-hosted Artifactory instance, skip to
[the configuring the receiver](#configuring-the-receiver) section.

:::

In order for a webhook initiated `Warehouse` refresh to successfully occur,
it is required that you set a <Hlt>Custom Base URL</Hlt> for your instance. 
When this setting hasn't been configured, critical information will be missing 
from the webhook payloads.

1. Navigate to 
`https://<base-url>/ui/admin/configuration/general`, where `<base-url>` has been replaced with the base URL of your self-hosted Artifactory instance.

1.  Set the <Hlt>Custom Base URL</Hlt> field to the base URL of your self-hosted
    Artifactory instance.

    ![Custom Base URL](./img/custom_base_url.png "Custom Base URL")

1. At the bottom of the form, click <Hlt>Save</Hlt>.

:::info

For additional information on configuring your <Hlt>Custom Base URL</Hlt>
refer directly to the [Artifactory Docs](https://jfrog.com/help/r/jfrog-platform-administration-documentation/general-settings).

:::

## Configuring the Receiver

An Artifactory webhook receiver must reference a Kubernetes `Secret` resource 
with a `secret-token` key in its data map. This
[shared secret](https://en.wikipedia.org/wiki/Shared_secret) will be used by
Artifactory to sign requests and by the receiver to verify those signatures.

:::note

The following commands are suggested for generating and base64-encoding a
complex secret:

```shell
secret_token=$(openssl rand -base64 48 | tr -d '=+/' | head -c 32)
echo "Secret token: $secret_token"
echo "Encoded secret token: $(echo -n $secret_token | base64)"
```

:::

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: artifactory-wh-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret-token: <base64-encoded secret token>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers: 
  - name: artifactory-wh-receiver
    artifactory:
      secretRef:
        name: artifactory-wh-secret
```

### Virtual Repositories

When Warehouses intended to be refreshed by an Artifactory webhook receiver
subscribe to Artifactory
[virtual repositories](https://jfrog.com/help/r/jfrog-artifactory-documentation/virtual-repositories) there will be discrepancies between the URLs the receiver will
infer for the
[local repositories](https://jfrog.com/help/r/jfrog-artifactory-documentation/local-repositories) from which push events have originated and the URLs actually used
by those Warehouses' subscriptions.

To compensate for this, a value can be provided for the Artifactory webhook
receiver configuration's `virtualRepoName` field. When specified, its value
supersedes the local repository name found in the webhook's payload, which
allows the receiver to infer the correct virtual repository URL for which all
subscribed Warehouses should be refreshed.

In practice, when using virtual repositories, a separate Artifactory webhook
receiver should be configured _for each_, but one such receiver can handle
events originating from _any number_ of local repositories that are aggregated by
that virtual repository. For example, if a virtual repository `proj-virtual`
aggregates container images from all of the `proj` Artifactory project's local
image repositories, with a single webhook configured to post to the following
receiver, an image pushed to
`example.frog.io/proj-<local-repo-name>/<path>/image`, will correctly cause that
receiver to refresh all Warehouses subscribed to
`example.frog.io/proj-virtual/<path>/image`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: artifactory-wh-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret-token: <base64-encoded secret token>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers: 
  - name: proj-virtual-wh-receiver
    artifactory:
      secretRef:
        name: artifactory-wh-secret
      virtualRepoName: proj-virtual
```

## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the receiver's configuration. This
URL can be obtained using a command such as the following:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```

## Registering with Artifactory

1. Navigate to 
   `https://<jfrog-instance>.jfrog.io/ui/admin/configuration/webhooks`, where
   `<jfrog-instance>` has been replaced with an Artifactory instance for which 
   you are an administrator.

1. Click <Hlt>New Webhook</Hlt>.

    ![Webhooks Dashboard](./img/webhooks.png "Webhooks Dashboard")

1. Complete the <Hlt>Create new webhook</Hlt> form:

    ![Add Webhook](./img/add-webhook.png "Add Webhook")

    1. Enter a descriptive name in the <Hlt>Name</Hlt> field.

    1. Complete the <Hlt>URL</Hlt> field using the URL
       [for the webhook receiver](#retrieving-the-receivers-url).

    1. Under <Hlt>Execution Results</Hlt> check
       <Hlt>Show status of successful executions in the Troubleshooting tab</Hlt>.

        :::info

        Although Artifactory supports sending test events to the URL, such
        events are _not_ displayed in the troubleshooting tab; only actual
        events are.
        :::

    1. In the <Hlt>Events</Hlt> drop-down, select
       <Hlt>Docker and OCI</Hlt> ⃗ <Hlt>Tag was pushed</Hlt>.

        ![Select Trigger](./img/select-trigger.png "Select Trigger")

        :::info

        Artifactory supports many different types of registries and repositories.
        This webhook responds only to events originating from repositories in OCI
        registries. No other type of repository, including legacy (HTTP/S) Helm
        chart repositories, is supported.
        :::

    1. Complete the dialog that appears:

       ![Select Repos](./img/select-repos.png "Select Repos")

        1. Select repositories from which you would like to receive events from
           those listed on the left.

        1. Click <Hlt>&gt;</Hlt> to move your selections to the right.

            Upon doing so, the <Hlt>Save</Hlt> button will be enabled.

            ![Repos Selected](./img/repos-selected.png "Repos Selected")

        1. Click <Hlt>Save</Hlt>.

    1. Under <Hlt>Authentication</Hlt>, complete the <Hlt>Secret token</Hlt>
       field using the (unencoded) value of the `secret-token` key in the
       `Secret` resource referenced by the
       [webhook receiver's configuration](#configuring-the-receiver).

        ![Setup Auth](./img/setup-auth.png "Setup Auth")

    1. Select <Hlt>Use secret for payload signing</Hlt>.

        :::caution

        The webhook receiver won't accept unsigned requests.
        :::

    1. Click <Hlt>Save</Hlt>.

        You will be redirected to the <Hlt>Webhooks</Hlt> page where the newly
        created webhook will appear.

        ![Created](./img/created.png "Created")
