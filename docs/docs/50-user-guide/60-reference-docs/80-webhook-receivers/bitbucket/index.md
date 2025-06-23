---
sidebar_label: Bitbucket
---

# The Bitbucket Webhook Receiver

The Bitbucket webhook receiver will respond to `repo:push` events originating
from Bitbucket repositories.

In response to a `repo:push` event, the receiver "refreshes" `Warehouse's`
subscribed to the Bitbucket repository from which the event originated.

:::info
"Refreshing" a `Warehouse` resource means enqueuing it for immediate
reconciliation by the Kargo controller, which will execute the discovery of
new artifacts from all repositories to which that `Warehouse` subscribes.
:::

## Configuring the Receiver

The Bitbucket webhook receiver will need to reference a Kubernetes `Secret` with
a `secret` key in its data map. This
[shared secret](https://en.wikipedia.org/wiki/Shared_secret) will be used by
Bitbucket to sign requests. The receiver will use it to authenticate those
requests by verifying their signatures.

:::note
The following commands are suggested for generating and base64-encoding a
complex secret:

```shell
secret=$(openssl rand -base64 48 | tr -d '=+/' | head -c 32)
echo "Secret: $secret"
echo "Encoded secret: $(echo -n $secret | base64)"
```

:::

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bb-wh-secret
  namespace: kargo-demo
data:
  secret: <base64-encoded secret>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers: 
  - name: bb-wh-receiver
    bitbucket:
      secretRef:
        name: bb-wh-secret
```

## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the configuration. We can obtain
this URL using the following command:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```

## Registering with Bitbucket

To configure a single repository to notify the receiver of relevant events:

1. Navigate to
   `https://bitbucket.org/<workspace>/<repository>/admin/webhooks` where
   `<workspace>` has been replaced with a Bitbucket workspace for which you are
   an administrator and `<repository>` has been replaced with the name of a
   repository belonging to that workspace.

1. Click <Hlt>Add webhook</Hlt>.

1. Complete the <Hlt>Add new webhook</Hlt> form:

    ![Add New Webhook Form](./img/add-new-webhook-form.png "Add New Webhook Form")

    1. Enter a <Hlt>Title</Hlt> with a short description.

    1. Set <Hlt>URL</Hlt> to the URL
       [for the webhook receiver](#retrieving-the-receivers-url).

    1. Set <Hlt>Secret</Hlt> to the value assigned to the `secret` key
       of the `Secret` referenced by the
       [webhook receiver's configuration](#configuring-the-receiver).

        :::danger
        Do not use the <Hlt>Generate secret</Hlt> button in the Bitbucket UI.

        Kargo incorporates the secret's value into the URL it generates for the
        webhook receiver. Using a secret in this field other than the one
        already referenced by the receiver's configuration will require
        revisiting that configuration _and doing so will change the receiver's
        URL._
        :::

    1. Under <Hlt>Status</Hlt>, ensure <Hlt>Active</Hlt> is checked.

    1. Under <Hlt>Triggers</Hlt> → <Hlt>Repository</Hlt>, ensure <Hlt>Push</Hlt>
       is checked.

    1. Click <Hlt>Save</Hlt>.

1. Verify that the webhook appears under <Hlt>Repository hooks</Hlt>.

1. If you'd like to record outbound webhook requests for troubleshooting
   purposes:

    1. Click the <Hlt>View requests</Hlt> link next to your webhook.

    1. Click on <Hlt>Enable History</Hlt>.

    ![Enable History](./img/enable-history.png "Enabled History")

:::info
For additional information on configuring webhooks, refer directly to the
[Bitbucket Docs](https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/).
:::
