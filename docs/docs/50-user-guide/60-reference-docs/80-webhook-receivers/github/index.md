---
sidebar_label: GitHub
---

# GitHub

The following instructions will work for Github,
Github Enterprise Cloud, and GitHub Enterprise Server.

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

Github will sign payloads using this `secret` via HMAC signature. Our webhook receiver will use this secret to verify the signature.

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

### Webhooks

1. Navigate to Settings

![Step 1](/img/github/webhooks/1.png "Settings")

2. Navigate to Webhooks

![Step 2](/img/github/webhooks/2.png "Webhooks")

3. Create A New Webhook

:::note
The `Payload URL` will use the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

The `Content type` field must be set to `application/json`.

The `Secret` field must be set to the `secret` key from the [Github Webhook Receiver Configuration](#github-webhook-receiver-configuration) step.
:::

![Step 3](/img/github/webhooks/4.png "Add Webhook")

Leave the `Just the push event` field checked unless you're
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


### Apps

It may be tedious to configure webhooks for each of your Github repositories. You can instead opt to configure a [Github App](https://docs.github.com/en/apps); allowing you to receive events from all or select repositories.

1. Navigate to Settings

This will be listed in a dropdown menu that is
toggled by clicking your Github avatar.

![Step 1](/img/github/apps/1.png "Settings")

2. Navigate to Developer Settings

This will be in the bottom left-hand corner of the settings dashboard.

![Step 2](/img/github/apps/2.png "Developer Settings")

3. Navigate to Github Apps

![Step 3](/img/github/apps/3.png "Github Apps")

4. Register a new Github App

Add a unique name and a homepage URL (this can be repo URL).

![Step 4](/img/github/apps/4.png "Register New App")

:::note
The `Webhook URL` requires the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

The `Secret` field must be set to the `secret` key from the [Github Webhook Receiver Configuration](#github-webhook-receiver-configuration) step.
:::

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

In the Github Apps dashboard, navigate to `Advanced` in the left-hand side menu and click `Recent Deliveries`.

![Step 11](/img/github/apps/11.png "Recent Deliveries")

Click on the `ping` event and then the `response` tab to
verify the connection was established successfully.

![Step 12](/img/github/apps/12.png "Response")

#### Additional Documentation

For more additional information on configuring Github Webhooks or Apps, refer to the [Github Docs](https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks)

