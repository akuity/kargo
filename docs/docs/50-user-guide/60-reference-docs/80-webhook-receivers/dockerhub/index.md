---
sidebar_label: Docker Hub
description: How to configure Docker Hub webhooks with Kargo for automated artifact discovery.
---

# Docker Hub Webhook Receiver

The Docker Hub Webhook Receiver responds when container images or charts are
pushed to Docker Hub repositories.

When this happens, the receiver "refreshes" `Warehouse` resources subscribed to
the corresponding Docker Hub repository.

:::info
"Refreshing" a `Warehouse` means enqueuing it for immediate reconciliation by
the Kargo controller, which will attempt to discover new artifacts from all
subscribed repositories.
:::

## Configuring the Receiver

To enable webhook support for Docker Hub, you must configure a Kubernetes
`Secret` and reference it in your `ProjectConfig`.

The `Secret` must include a `secret` key in its data map. This value is used to
generate a unique, hard-to-guess URL for the webhook receiver, providing basic
protection against unauthorized requests. Because Docker Hub webhook payloads
are **not** signed, Kargo uses this static token for basic validation.

:::note
The following command is suggested for generating a complex shared secret and
encoding it for use in the `data` field:

```shell
openssl rand -base64 48 | tr -d '=+/' | head -c 32 | base64
```

:::

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dh-wh-secret
  namespace: kargo-demo
data:
  secret: <your-base64-encoded-token-here>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers:
    - name: dh-wh-receiver
      dockerhub:
        secretRef:
          name: dh-wh-secret
```

## Retrieving the Receiver's URL

After applying the `ProjectConfig`, you can retrieve the unique URL for the
Docker Hub webhook receiver using the following command:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```

## Registering with Docker Hub

To configure a Docker Hub repository to send events to the webhook receiver:

1. Navigate to your Docker Hub repository and select the <Hlt>Webhooks</Hlt> tab.

   ![Webhooks Tab](./img/webhooks-tab.png "Webhooks Tab")

1. In the <Hlt>Webhooks</Hlt> form:

   ![New Webhook](./img/new-webhook.png "New Webhook Form")

   1. Provide a name for the webhook.

   1. Set <Hlt>Webhook URL</Hlt> to the
      [receiver URL](#retrieving-the-receivers-url).

   1. Click <Hlt>+</Hlt> to create webhook.

      ![Create Webhook](./img/create-webhook.png "Create Webhook Button")

## Verifying connectivity when a new image is pushed

1. Return to the <Hlt>Webhooks</Hlt> tab of your repository.

1. In the <Hlt>Current Webhooks</Hlt> section, hover over your webhook,
   select the _menu options_ icon, and click <Hlt>View History</Hlt>.

   ![View Webhook History](./img/view-history.png "View History")

1. Check the delivery log to confirm a successful webhook request.

   ![Delivery Detail](./img/delivery-detail.png "Webhook Delivery Detail")

If everything is configured correctly, Kargo will automatically refresh the
corresponding `Warehouse` and initiate artifact discovery whenever new images
or charts are pushed to your Docker Hub repository.
