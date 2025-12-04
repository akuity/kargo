---
sidebar_label: Generic
---

# Generic Webhook Receiver

The Generic webhook receiver responds to events originating from arbitrary
repositories by performing `Action`s on `Target`s. Consider the example of a 
`Warehouse` refresh. In this example, the `Action` is the refresh and the 
`Target` is the `Warehouse`.

:::note
A generic webhook receiver is not limited to a single `Action` and `Action`'s 
are not limited to a single `Target`.
:::

:::info
"Refreshing" a `Warehouse` resource means enqueuing it for immediate
reconciliation by the Kargo controller, which will execute the discovery of new
artifacts from all repositories to which that `Warehouse` subscribes.
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
      actions:
        - actionType: Refresh
          # Parameters can optionally be defined for use in expressions
          parameters:
            - targetEvent: "push"
            - headerKey: "X-Event-Type"
          # Only perform this action if this expression is satisfied
          matchExpression: "request.header(params.headerKey) == params.targetEvent"
          targets:
            # This target is designated via static name
            - kind: Warehouse
              name: my-warehouse
            # This target is designated via expression derived name
            - kind: Warehouse
              name: "${{ request.body.repository.name }}"
            # This target omits the name all together and designates any
            # resources that fulfill the label and index selector criteria
            - kind: Warehouse
              labelSelector:
                matchLabels:
                  environment: production
                matchExpressions:
                  - key: tier
                    operator: In
                    values: ["critical", "high"]
              indexSelector:
                matchExpressions:
                  - key: subscribedURL
                    operator: In
                    value: "${{ normalizeGit(request.body.repository.url) }}"
```

:::note
`name`, `labelSelector`, and `indexSelector` are all
optional. However, at least one of them must be specified.
:::

## Designating Targets

There are 3 different ways of designating `Target`s:

1. [Static](#static)
1. [Expression-Derived](#expression-derived)
1. [Selector-Based](#selector-based)

### Static

The simplest way of designating a `Target` is by setting a static `name` 
to identify the resource.

### Expression-Derived

The Generic webhook receiver extends
[built-in expr-lang support](https://expr-lang.org/docs/language-definition) 
with various expression functions and objects. These functions and objects can 
be used to help users derive `Target` information from incoming requests.

The following expression functions are available:

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

`request.body`

Generic object derived from incoming requests whose fields can be accessed using
standard bracket or dot-notation. For example, `data.address.city` would access 
the `city` property nested within the `address` object, and `data.users[0]` 
would access the first item in a `users` array. 

### Selector-Based

Generic webhook receivers use label and index selectors. They can be used in
combination with one another(as well as a static or dynamic `Name`). Listed 
targets would be ones that match the logical AND of all defined constraints.

#### Label Selectors

Label selectors contain support for both `matchLabels` and `matchExpressions`.
A static list of key/value pairs should be used for `matchLabels`. For 
`matchExpressions`, you must specify a `key`, `operator` and `values` where
`values` supports static and/or expression derived values.

```
labelSelector:
  matchLabels:
    environment: production
  matchExpressions:
  - key: env
    operator: In
    values: ["critical", "high"]
```

#### Index Selectors

Index Selectors have `matchExpression` support. They use a `key`, `operator`, 
and `value` combination. Supported operators include `Equal` and `NotEqual`. The
only supported `key` at this time is `subscribedURLs`. Expressions are supported
for `value`.

:::note
`subscribedURLs` refers to `Warehouse`'s that contain subscriptions that 
subscribe to the provided repository URL.
:::

```
indexSelector:
  matchExpressions:
  - key: subscribedURLs
    operator: Equal
    value: "${{ normalize("git", request.body.repository.url) }}"
```


## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the receiver's configuration. This
URL can be obtained using a command such as the following:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```

This URL can then be used anywhere webhooks can be configured.
