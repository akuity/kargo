---
sidebar_label: GitHub
---

# GitHub

## Required Secrets for Github

Our webhook receiver will require a Kubernetes secret. This secret is required to contain a `secret` key in its `stringData`. We will need to provide the value we assigned to the `secret` key to GitHub in a later step so keep it handy.

```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: my-secret
      namespace: my-namespace
    stringData:
    # Replace 'your-secret-here' with any non-empty
    # arbitrary string data.
    # The key here literally needs to be named 'secret'.
      secret: your-secret-here
```

Github will sign payloads using this secret via HMAC signature. Our webhook receiver will use this secret to verify the signature.

## Github Webhook Receiver Configuration

Create a new project config; specifying a Github webhook receiver that
targets the secret we created in the last step.

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: my-project-config
  namespace: my-namespace
spec:
  webhookReceivers: 
    - name: my-receiver
      github:
        secretRef:
          name: my-secret
```

## Retrieving the Webhook URL

The secret (among other things) will be used as an input to generate
a secure URL for our newly created webhook receiver. We can obtain
this URL using the following command:

    kubectl \
        get projectconfigs \
        my-project-config \
        -n my-namespace \
        -o=jsonpath='{.status.webhookReceivers[0].url}'


## Configure on Github

When configuring on Github, you can configure either a webhook or an app. We will outline instructions for both, starting with webhooks.

    kubectl \
        get secrets \
        my-secret \
        --template={{.data.secret}} | base64 -d

#### Webhooks

### Step 1: Navigate to Settings

![Step 1](/static/img/github/webhooks/1.png "Settings")

### Step 2: Navigate to Webhooks

![Step 2](/static/img/github/webhooks/2.png "Webhooks")

### Step 3: Create A New Webhook

Click the `Add Webhook` button.

![Step 3](/static/img/github/webhooks/3.png "Add Webhook Button")

1. For the `Payload URL`, we will use the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

2. Select `application/json` for the `Content type` field.

3. In the `Secret` field, we will input the value we assigned to the `secret` key in [Required Secrets for GitHub](#required-secrets-for-github).

![Step 4](/static/img/github/webhooks/4.png "Add Webhook")

Leave `Just the push event` field checked unless you're
looking to subscribe to `ghcr` events.

If you're looking to subscribe to `ghcr` events you should select `Let me select individual events` and then select `Packages`.

![Step 5](/static/img/github/webhooks/5.png "Event Subscription")

Then finally make sure to toggle the webhook as `Active` and
press `Add webhook`.

![Step 6](/static/img/github/webhooks/6.png "Submit Form")

### Step 4: Verify Connectivity

Click on the webhook URL in the view below.

![Step 7](/static/img/github/webhooks/7.png "Created")

Navigate toe `Recent Deliveries`.

![Step 8](/static/img/github/webhooks/8.png "Recent Deliveries")

Click on the `ping` event and ensure a successful response was returned.

![Step 9](/static/img/github/webhooks/9.png "Response")


#### Apps

### Step 1: Navigate to Settings

This will be listed in a dropdown menu that is
toggled by clicking your Github avatar.

![Step 1](/static/img/github/apps/1.png "Settings")

### Step 2: Navigate to Developer Settings

This will be in the bottom left-hand corner of the settings UI.

![Step 2](/static/img/github/apps/2.png "Developer Settings")

### Step 3: Navigate to Github Apps

![Step 3](/static/img/github/apps/3.png "Github Apps")

### Step 4: Register a new Github App

1. Click the `New Github App` button.

2. Add a unique name and a homepage URL (this can be repo URL).

![Step 4](/static/img/github/apps/4.png "Register New App")

For the `Webhook URL` field, we will use the value we retrieved from the [Retrieving the Webhook URL](#retrieving-the-webhook-url) step.

In the `Secret` field, we will input the value we assigned to the `secret` key in [Required Secrets for GitHub](#required-secrets-for-github).

![Step 5](/static/img/github/apps/5.png "Configure Webhook")

### Step 5: Configure Permissions

![Step 6](/static/img/github/apps/6.png "Permissions")

For the option to subscribe to repo push events, we will need `read + write` access for the `Contents` permission.

![Step 7](/static/img/github/apps/7.png "Permissions - Contents")

For the option to subscribe to registry push events(ghcr), we will need `read + write` access for the `Packages` permission.

![Step 8](/static/img/github/apps/8.png "Permissions - Packages")

### Step 6: Configure Event Subscriptions

Here we can subscribe to `push` or `package` events depending
on the permissions you selected in the previous step.

![Step 9](/static/img/github/apps/9.png "Subscribe to Events")

### Step 7: Confirm Visibility + Create

![Step 10](/static/img/github/apps/10.png "Submit Form")

### Step 8: Verify

In the Github Apps UI, navigate to `Advanced` in the left-hand side menu and click `Recent Deliveries`.

![Step 11](/static/img/github/apps/11.png "Recent Deliveries")

Click on the `ping` event and then the `response` tab to
verify the connection was established successfully.

![Step 12](/static/img/github/apps/12.png "Response")

For more additional information on configuring Github Webhooks or Apps, refer to the [Github Docs](https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks)

