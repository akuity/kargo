---
sidebar_label: Quay.io
---

# Quay

## Required Secrets for Github

Our webhook receiver will require a Kubernetes secret. This secret is required to contain a `secret` key in its `stringData`.

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

## Quay Webhook Receiver Configuration

Create a new project config; specifying a Quay webhook receiver that
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
      quay:
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


## Configure on Quay

### Step 1: Navigate to Repository Settings

![Step 1](/img/quay/1.png "Settings")

### Step 2: Click Create Notification

![Step 2](/img/quay/2.png "Create Notification Button")

### Step 3: Fill out the form

Select `Push to Repository` from the first dropdown menu( This is the only supported event for Quay at this time).

Select `Webhook POST` from the second dropdown menu.

For the third input field (`Webhook URL`) we will input our webhook receiver URL.

![Step 3](/img/quay/3.png "Create Notification Form")

### Step 3: Submit

Click the `Create Notification` button.

![Step 4](/img/quay/4.png "Submit Form")

![Step 5](/img/quay/5.png "Created")

For additional information on configuring Quay notifications/webhooks, refer to the [Quay Docs](https://docs.quay.io/guides/notifications.html).
