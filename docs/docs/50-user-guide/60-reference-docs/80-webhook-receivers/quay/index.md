---
sidebar_label: Quay.io
---

# The Quay Webhook Receiver

The Quay Webhook Receiver will respond to `Push to Repository` events by 
refreshing any Warehouses subscribed to the repository from which the event
originated.

In response to a `Push to Repository` event, the receiver "refreshes" 
`Warehouse`s subscribed to the Image repository from which the event originated.

:::info
"Refreshing" a `Warehouse` resource means enqueuing it for immediate
reconciliation by the Kargo controller, which will execute the discovery of
new artifacts from all repositories to which that `Warehouse` subscribes.
:::

## Configuring the Receiver

The Quay webhook receiver will need to reference a Kubernetes `Secret` with a
`secret` key in its data map.

:::note
The following command is suggested for generating a complex shared secret:

```shell
openssl rand -base64 48 | tr -d '=+/' | head -c 32
```
:::

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: q-wh-secret
  namespace: kargo-demo
stringData:
  secret: <your-secret-here>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers: 
    - name: q-wh-receiver
      github:
        secretRef:
          name: q-wh-secret
```

## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the configuration. We can obtain
this URL using the following command:

```shell
  kubectl \
    get projectconfigs \
    kargo-demo \
    -n kargo-demo \
    -o=jsonpath='{.status.webhookReceivers}'
```


## Registering with Quay

1. Navigate to `https://quay.io/repository/<account>/<repository>?tab=settings`,
   where `<account>` has been replaced with your Quay username or an organization
   for which you are an administrator and `<repository>` has been replaced with
   the name of a repository belonging to that account.

    ![Repository Settings](./img/repository-settings.png "Repository Settings")

    1. Scroll down to <Hlt>Events and Notifications</Hlt>.

    1. Click <Hlt>Create Notification</Hlt>.

1. Complete the <Hlt>Create repository notification</Hlt> form.

    ![Create Repository Notification](./img/create-repository-notification.png "Create Notification Form")

    1. Select <Hlt>Push to Repository</Hlt> from the <Hlt>When this event 
    occurs</Hlt> dropdown menu.

    1. Select <Hlt>Webhook POST</Hlt> from the 
    <Hlt>Then issue a notification</Hlt> dropdown menu.

    1. Set the <Hlt>Webhook URL</Hlt> to the URL 
    [retrieved for the webhook receiver](#retrieving-the-receivers-url).

    1. Click <Hlt>Create Notification</Hlt>.

:::info
You will then be redirected to the <Hlt>Repository Settings</Hlt> dashboard
where you should now see the notification you just created.
:::

![Created](./img/created.png "Created")

For additional information on configuring Quay notifications, refer to the
[Quay Docs](https://docs.quay.io/guides/notifications.html).
