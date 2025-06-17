---
sidebar_label: GitLab
---

# GitLab

## Required Secrets for GitLab

Our webhook receiver will require a Kubernetes secret. This secret is required to contain a `secret-token` key in its `stringData`. We will need to provide the value we assigned to the `secret-token` key will to GitLab in a later step so keep it handy.

```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: my-secret
      namespace: my-namespace
    stringData:
    # Replace 'your-secret-token-here' with any non-empty
    # arbitrary string data.
    # The key here literally needs to be named 'secret-token'.
      secret-token: your-secret-token-here
```

Our webhook receiver will use the `secret-token` to verify the `X-Gitlab-Token` header that comes from GitLab.

## GitLab Webhook Receiver Configuration

Create a new project config; specifying a GitLab webhook receiver that
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
      gitlab:
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


## Configure on GitLab

### Step 1: Navigate to Settings -> Webhooks

![Step 1](/img/gitlab/1.png "Settings")

### Step 2: Click Add New Webhook

![Step 2](/img/gitlab/2.png "Add Webhook Button")

### Step 3: Complete Form

Our webhook receiver URL goes in the `URL` field.

In the `Secret token` field, we will input the value we assigned to the `secret-token` key in [Required Secrets for Gitlab](#required-secrets-for-gitlab).

![Step 3](/img/gitlab/3.png "Add Webhook Form")

### Step 4: Submit Form

![Step 4](/img/gitlab/4.png "Submit Form")

![Step 5](/img/gitlab/5.png "Created")

### Step 5: Test

Select a `Push events` from the `Test` dropdown menu.

![Step 6](/img/gitlab/6.png "Test Button")

### Step 6: Verify

Click the `Edit` button.

![Step 7](/img/gitlab/7.png "Edit Button")

Scroll down to `Recent Events` and click `View Details`.

![Step 8](/img/gitlab/8.png "Recent Events")

Confirm successful response.

![Step 9](/img/gitlab/9.png "Response")

For additional information on configuring GitLab webhooks, refer to the [GitLab Docs](https://docs.gitlab.com/user/project/integrations/webhooks/).