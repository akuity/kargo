---
sidebar_label: Static
---

# Static Webhook Receiver

The static webhook receiver responds to arbitrary events originating
from arbitrary repositories by executing an action against target(s) defined
in the static webhook receiver rule.

## Configuring the Receiver

A static webhook receiver must reference a Kubernetes `Secret` resource with a
`secret` key in its data map.

:::info
_This secret will not be shared directly with event host_

Kargo incorporates the secret into the generation of a hard-to-guess URL for the
receiver. This URL serves as a _de facto_
[shared secret](https://en.wikipedia.org/wiki/Shared_secret) and authentication
mechanism.
:::

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
  name: static-wh-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
data:
  secret-token: <base64-encoded secret>
---
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers: 
    - name: my-wh-receiver
      static:
        secretRef:
          name: static-wh-secret
        rule:
          action: Refresh
          targets:
            - name: my-warehouse
              namespace: kargo-demo
              type: Warehouse
            - name: my-other-warehouse
              namespace: kargo-demo
              type: Warehouse
```

:::note
In the example above, our rule definition states that when events are received, 
we will perform the `Refresh` action on the targets specified. In this case 
two `Warehouse`'s.
:::

## Type/Action combinations


| Type  |  Supported Actions |
|---|---|
| Warehouse  |  Refresh |

## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the receiver's configuration. This
URL can be obtained using a command such as the following:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```

## Registering your static webhook receiver

After retrieving the receiver's URL, you can configure it on any platform
that supports webhooks.

:::note
If you only want the action to be applied to a single target within your target
list, you can do so by appending the `?target=<target-name>` query param to your receiver URL when registering your webhook on your host provider.
:::