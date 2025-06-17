---
sidebar_label: GitHub
---

# GitHub

## GitHub Webhook Receiver Configuration

The Kargo GitHub webhook receiver will require a Kubernetes Secret. This Secret is required to contain a `secret` key in its data map. You will be required to provide the value assigned to the `secret` key to GitHub in a later step so keep it handy.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gh-wh-secret
  namespace: kargo-demo
stringData:
  secret: your-secret-here
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: my-project-config
  namespace: kargo-demo
spec:
  webhookReceivers: 
    - name: gh-wh-receiver
      github:
        secretRef:
          name: gh-wh-secret
```

Github will sign payloads using this secret via HMAC signature. Our webhook receiver will use this secret to verify the signature.

:::note
The following command can be used to generate a sufficiently secure secret:

```shell
echo "$(openssl rand -base64 48 | tr -d '=+/' | head -c 32)"
```

:::

## Retrieving the Webhook URL

The secret (among other things) will be used as an input to generate
a secure URL for our newly created webhook receiver. We can obtain
this URL using the following command:

```shell
  kubectl \
    get projectconfigs \
    my-project-config \
    -n kargo-demo \
    -o=jsonpath='{.status.webhookReceivers}'
```


## Configure on Github

When configuring on Github, you can configure either a webhook or an app. We will outline instructions for both, starting with webhooks.

```shell
  kubectl \
    get secrets \
    my-secret \
    -n kargo-demo \
    --template={{.data.secret}} | base64 -d
```

#### Webhooks

1. Navigate to Settings

![Step 1](/img/github/webhooks/1.png "Settings")

2. Navigate to Webhooks

![Step 2](/img/github/webhooks/2.png "Webhooks")

3. Create A New Webhook

3.a Click the `Add Webhook` button.

3.b For the `Payload URL`, we will use the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

3.c Select `application/json` for the `Content type` field.

3.d In the `Secret` field, we will input the value we assigned to the `secret` key in [Required Secrets for GitHub](#required-secrets-for-github).

![Step 3](/img/github/webhooks/4.png "Add Webhook")

Leave `Just the push event` field checked unless you're
looking to subscribe to `ghcr` events.

:::note
	If you're looking to subscribe to `ghcr` events you should select `Let me select individual events` and then select `Packages`.
  This requires that you have connected the repository and package. For more information on connecting repositories and packages refer to the [Github Docs here](https://docs.github.com/en/packages/learn-github-packages/connecting-a-repository-to-a-package).
:::

![Step 5](/img/github/webhooks/5.png "Event Subscription")

Then finally make sure to toggle the webhook as `Active` and
press `Add webhook`.

![Step 6](/img/github/webhooks/6.png "Submit Form")

4. Verify Connectivity

Click on the webhook URL in the view below.

![Step 7](/img/github/webhooks/7.png "Created")

Navigate to `Recent Deliveries`.

![Step 8](/img/github/webhooks/8.png "Recent Deliveries")

Click on the `ping` event and ensure a successful response was returned.

![Step 9](/img/github/webhooks/9.png "Response")


#### Apps

1. Navigate to Settings

This will be listed in a dropdown menu that is
toggled by clicking your Github avatar.

![Step 1](/img/github/apps/1.png "Settings")

2. Navigate to Developer Settings

This will be in the bottom left-hand corner of the settings UI.

![Step 2](/img/github/apps/2.png "Developer Settings")

3. Navigate to Github Apps

![Step 3](/img/github/apps/3.png "Github Apps")

4. Register a new Github App

4.a Click the `New Github App` button.

4.b Add a unique name and a homepage URL (this can be repo URL).

![Step 4](/img/github/apps/4.png "Register New App")

For the `Webhook URL` field, we will use the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

In the `Secret` field, we will input the value we assigned to the `secret` key in [Required Secrets for GitHub](#required-secrets-for-github).

![Step 5](/img/github/apps/5.png "Configure Webhook")

5. Configure Permissions

![Step 6](/img/github/apps/6.png "Permissions")

For the option to subscribe to repo push events, we will need `read + write` access for the `Contents` permission.

![Step 7](/img/github/apps/7.png "Permissions - Contents")

For the option to subscribe to registry push events(ghcr), we will need `read + write` access for the `Packages` permission.

![Step 8](/img/github/apps/8.png "Permissions - Packages")

6. Configure Event Subscriptions

Here we can subscribe to `push` or `package` events depending
on the permissions you selected in the previous step.

![Step 9](/img/github/apps/9.png "Subscribe to Events")

7. Confirm Visibility + Create

![Step 10](/img/github/apps/10.png "Submit Form")

8. Verify

In the Github Apps UI, navigate to `Advanced` in the left-hand side menu and click `Recent Deliveries`.

![Step 11](/img/github/apps/11.png "Recent Deliveries")

Click on the `ping` event and then the `response` tab to
verify the connection was established successfully.

![Step 12](/img/github/apps/12.png "Response")

For more additional information on configuring Github Webhooks or Apps, refer to the [Github Docs](https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks)

