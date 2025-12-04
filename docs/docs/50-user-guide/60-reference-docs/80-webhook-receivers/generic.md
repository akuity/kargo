---
sidebar_label: Generic
---

# Generic Webhook Receiver

The generic webhook receiver responds to any inbound POST request by determining
whether it meets user-defined criteria, then executing user-defined actions on a
user-defined set of resources when it does.

:::note
Currently, these actions are limited to "refreshing" `Warehouse` resources,
which triggers their artifact discovery processes, so a typical use of this
component is responding to "push" events from artifact repositories that
lack dedicated webhook receiver implementations. Since this component
effectively enables imperatively refreshing a `Warehouse` from any external
process, other uses are possible and practical.
:::

## Configuring the Receiver

A Generic webhook receiver must reference a Kubernetes `Secret` resource with
a `secret` key in its data map.

:::info
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
  name: wh-secret
  namespace: kargo-demo
  labels:
    kargo.akuity.io/cred-type: generic
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
    - name: my-receiver
      generic:
        secretRef:
          name: wh-secret
```

### matchExpression

Use `matchExpression` to ensure that an action is only executed when specific 
criteria are met, providing fine-grained control over webhook behavior.

#### Example

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers:
    - name: my-receiver
      generic:
        secretRef:
          name: wh-secret
      actions:
        - actionType: Refresh
          # Only perform this action if this expression is satisfied
          matchExpression: "request.header("X-Event-Type") == 'push'"
```

### Designating Targets

There are 3 different ways of designating `Target`s:

1. [By name](#by-name)
1. [By labels](#by-labels)
1. [By values in an index](#by-values-in-an-index)

All of which support both static and 
[expression derived values](#expression-functions).

:::note
Designating targets by using more than one of the above methods results in
criteria that is the logical AND of all defined criteria.
:::

#### By name

The simplest way of designating a `Target` resource is by designating it by
`name`.

##### example

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers:
    - name: my-receiver
      generic:
        secretRef:
          name: wh-secret
      actions:
        - actionType: Refresh
          matchExpression: "request.header("X-Event-Type") == 'push'"
          targets:
            # This target is designated via static name
            - kind: Warehouse
              name: my-warehouse
            # This target is designated via expression derived name
            - kind: Warehouse
              name: "${{ normalizeGit(request.body.repository.name) }}"
```

### By labels

`labelSelector` contains supports for both `matchLabels` and `matchExpressions`.

#### example

```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: ProjectConfig
metadata:
  name: kargo-demo
  namespace: kargo-demo
spec:
  webhookReceivers:
    - name: my-receiver
      generic:
        secretRef:
          name: wh-secret
      actions:
        - actionType: Refresh
          matchExpression: "request.header("X-Event-Type") == 'push'"
          targets:
            - kind: Warehouse
              labelSelector:
                matchLabels:
                  # targets with 'environment' label with value 'prod'.
                  environment: prod
                matchExpressions:
                  # targets with 'service' label with value 'ui' OR 'api'.
                  - key: service
                    operator: In
                    values: ["ui", "api"]
```

### By values in an index

`indexSelector` can be used to retrieve resources by a cached index.

```yaml
actions:
  - actionType: Refresh
    matchExpression: "request.header("X-Event-Type") == 'push'"
    targets:
      - kind: Warehouse
        indexSelector:
          MatchIndices:
            - key: subscribedURLs
              operator: Equals
              value: "${{ normalizeGit(request.body.repository.url) }}"
```

:::note
`subscribedURLs` is the only available index and refers to `Warehouse`'s that 
contain subscriptions that subscribe to a provided repository URL.
:::

### Expression functions

The Generic webhook receiver extends
[built-in expr-lang support](https://expr-lang.org/docs/language-definition) 
with utilities that can be used to help derive `Target` information from 
incoming requests.

The following expression functions are available:

`request.body`

Derived from incoming requests whose fields can be accessed using
standard bracket or dot-notation. For example, `data.address.city` would access 
the `city` property nested within the `address` object, and `data.users[0]` 
would access the first item in a `users` array. 

`request.header(headerKey)`

Function that retrieves first value for `headerKey`.

It has one argument:
- `headerKey` (Required): Case-insensitive header key.

If `headerKey` is not present in the request headers, an empty `string` will
be returned.

`request.headers(headerKey)`

Function that retrieves all values for `headerKey`.

It has one argument:
- `headerKey` (Required): Case-insensitive header key.

If `headerKey` is not present in the request headers, an empty `string` array
will be returned.

`normalizeGit(url)`

Function that normalizes a git `url`.

It has one argument:
- `url` (Required): The URL of a git repository.

The returned value is a `string`.

`normalizeImage(url)`

Function that normalizes an image `url`.

It has one argument:
- `url` (Required): The URL of an image repository.

The returned value is a `string`.

`normalizeChart(url)`

Function that normalizes a chart `url`.

It has one argument:
- `url` (Required): The URL of a chart repository.

The returned value is a `string`.