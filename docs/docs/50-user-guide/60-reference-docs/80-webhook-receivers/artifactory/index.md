---
sidebar_label: Artifactory
---

# Artifactory Webhook Receiver

The Artifactory Webhook Receiver responds to `pushed` events originating from 
Artifactory repositories by _refreshing_ all `Warehouse` resources subscribed to 
those repositories.

:::info
"Refreshing" a `Warehouse` resource means enqueuing it for immediate
reconciliation by the Kargo controller, which will execute the discovery of
new artifacts from all repositories to which that `Warehouse` subscribes.
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
apiVersion: kargo.akuity.io/v1alpha1
kind: Project
metadata:
  name: kargo-demo
---
apiVersion: v1
kind: Secret
metadata:
  name: artifactory-wh-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: image
data:
  repoURL: <jfrog-instance>.jfrog.io/<repo-key>/<image-name>
  username: <jfrog username>
  password: <jfrog docker login password>
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

:::note
Artifactory repositories are private by default, if you have configured your
repository to be publicly available, you can omit the `repoURL`, `username`, and
`password` fields.
:::

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
  `<jfrog-instance>` has been replaced with na Artifactory instance for which 
  you are an administrator.

1. Click <Hlt>New Webhook</Hlt>.

![Webhooks Dashboard](./img/webhooks.png "Webhooks Dashboard")

1. Complete the <Hlt>Create new webhook</Hlt> form:

![Add Webhook](./img/add-webhook.png "Add Webhook")

  1. Enter a descriptive name in the <Hlt>Name</Hlt> field.

  1. Complete the <Hlt>URL</Hlt> field using the URL
      [for the webhook receiver](#retrieving-the-receivers-url).
  
  1. Under <Hlt>Execution Results</Hlt> check <Hlt>Show status of successful 
  executions in Troubleshooting tab.</Hlt>

  :::info
    Although Artifactory supports sending test/dummy events to the URL,
    only organically triggered events will show up in the troubleshooting tab.
    Not test/dummy events. Even if they're successful.
  :::

  1. Scroll down to <Hlt>Events</Hlt> and select <Hlt>Docker and OCI</Hlt> > 
  <Hlt>Tag was pushed</Hlt>.

![Select Trigger](./img/select-trigger.png "Select Trigger")

  1. Upon clicking out of the input menu, an <Hlt>Add Repositories</Hlt> modal 
  will appear.

![Select Repos](./img/select-repos.png "Select Repos")

  1. Check any boxes corresponding to repositores this applies to.

  1. Click <Hlt>></Hlt> to move selected repositores into the selected window.

  :::info
    Upon moving repositores to the selected section, the <Hlt>Save</Hlt> will
    become enabled.
  :::

![Repos Selected](./img/repos-selected.png "Repos Selected")

  1. Click <Hlt>Save</Hlt>.

  1. Scroll down to <Hlt>Authentication</Hlt>.

![Setup Auth](./img/setup-auth.png "Setup Auth")

  1. Complete the <Hlt>Secret token</Hlt> field using to the (unencoded) value
      assigned to the `secret-token` key of the `Secret` resource referenced by
      the
      [webhook receiver's configuration](#configuring-the-receiver).

  1. Select <Hlt>Use secret for payload signing</Hlt>.

  1. Click <Hlt>Save</Hlt>.

  1. You will then be redirected to the <Hlt>Webhooks Dashboard</Hlt> where the 
  newly created webhook will now be rendered.

![Created](./img/created.png "Created")