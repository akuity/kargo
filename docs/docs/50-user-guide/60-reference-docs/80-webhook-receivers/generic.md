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

### Base Configuration

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

### Defining Actions

Actions are defined by:

1. [`actionType`](#actiontype)
1. [`whenExpression`](#whenexpression)
1. [`targetSelectionCriteria`](#defining-targetselectioncriteria)

#### actionType

The `actionType` field specifies the action that should be performed.

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
```

:::note

The only currently supported `actionType` is `Refresh`.
:::


#### whenExpression

Use `whenExpression` to ensure that an action is only executed when specific
criteria are met, providing fine-grained control over webhook behavior.

The following example depicts an action that is executed only when the request
contains an `X-Event-Type` header with the value `push`:

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
          whenExpression: "request.header("X-Event-Type") == 'push'"
```

:::note

This can be left empty if the action should run unconditionally.

:::

#### targetSelectionCriteria

`targetSelectionCriteria` is used to select resources that an action needs
to be performed on. There are three ways to define `targetSelectionCriteria`:

1. [By Name](#by-name)
1. [By Labels](#by-labels)
1. [By Values in an Index](#by-values-in-an-index)

All methods support both static and [dynamic](#expression-reference) values.

:::note

Using more than one of the above selects resources at the logical intersection
of the criteria. (i.e. Criteria are "ANDed.")

:::

##### By Name

The simplest way to select a resource is by specifying its `name`.

The following example depicts `targetSelectionCriteria` that selects
`Warehouse` resources by a static `name`.

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
          whenExpression: "request.header('X-Event-Type') == 'push'"
          targetSelectionCriteria:
            - kind: Warehouse
              name: my-warehouse
```

The following example depicts `targetSelectionCriteria` that selects
`Warehouse` resources by a `name` that is dynamically sourced from a repository
name in the request body.

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
          whenExpression: "request.header('X-Event-Type') == 'push'"
          targetSelectionCriteria:
            - kind: Warehouse
              name: "${{ normalizeGit(request.body.repository.name) }}"
```

##### By Labels

The following example depicts `targetSelectionCriteria` that selects
`Warehouse` resources with an `environment` label whose value is `prod`:

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
          whenExpression: "request.header('X-Event-Type') == 'push'"
          targetSelectionCriteria:
            - kind: Warehouse
              labelSelector:
                matchLabels:
                  environment: prod
```

The following example depicts `targetSelectionCriteria` that selects
`Warehouse` resources with a `service` label whose value is either `ui` or
`api`:

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
          whenExpression: "request.header('X-Event-Type') == 'push'"
          targetSelectionCriteria:
            - kind: Warehouse
              labelSelector:
                matchExpressions:
                  - key: service
                    operator: In
                    values: ["ui", "api"]
```

##### By Values in an Index

Use `indexSelector` to retrieve resources by a cached index.

The following example depicts `targetSelectionCriteria` that dynamically selects
`Warehouse` resources that contain subscriptions to the normalized git
URL from the request body.

```yaml
actions:
  - actionType: Refresh
    whenExpression: "request.header('X-Event-Type') == 'push'"
    targetSelectionCriteria:
      - kind: Warehouse
        indexSelector:
          matchIndices:
            - key: subscribedURLs
              operator: Equals
              value: "${{ normalizeGit(request.body.repository.url) }}"
```

:::note

`subscribedURLs` is the only available index. It refers to `Warehouse`
resources that contain subscriptions for a provided repository URL.

:::

### Expression Reference

The Generic webhook receiver extends
[built-in expr-lang support](https://expr-lang.org/docs/language-definition)
with utilities that can be used to help resolve `targetSelectionCriteria`
information from incoming requests. The following reference contains the
variables and functions available for yielding dynamic values.

- [request.body](#requestbody)
- [request.header](#requestheaderheaderkey)
- [request.headers](#requestheadersheaderkey)
- [request.params](#requestparamsqueryparamkey)
- [normalizeGit](#normalizegiturl)
- [normalizeImage](#normalizeimageurl)
- [normalizeChart](#normalizecharturl)

#### request.body

`request.body` is structured data parsed from the original request body which is
assumed to contain valid JSON. Fields can be accessed using bracket or
dot-notation. For example, `data.address.city` would access the `city` property
nested within the `address` object, and `data.users[0]` would access the first
item in a `users` array.

#### request.header(headerKey)

Retrieves first value for `headerKey`.

It has one argument:

- `headerKey` (Required): Case-insensitive header key.

If `headerKey` is not present in the request headers, an empty `string` will
be returned.

#### request.headers(headerKey)

Retrieves all values for `headerKey`.

It has one argument:

- `headerKey` (Required): Case-insensitive header key.

If `headerKey` is not present in the request headers, an empty `string` array
will be returned.

#### request.params(queryParamKey)

Retrieves the query param value for the provided query param key.

It has one argument:

- `queryParamKey` (Required): URL query parameter key.

If `queryParamKey` is not present in the request headers, an empty `string`
will be returned.

#### normalizeGit(url)

Normalizes Git URLs of the following forms:

- `http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]`
- `ssh://[user@]host.xz[:port][/path/to/repo[.git][/]]`
- `[user@]host.xz[:path/to/repo[.git][/]]`

This is useful for the purposes of comparison and also in cases where a
canonical representation of a Git URL is needed. Any URL that cannot be
normalized will be returned as-is.

It has one argument:

- `url` (Required): The URL of a git repository.

The returned value is a `string`.

#### normalizeImage(url)

Normalizes image repository URLs. Notably, hostnames docker.io
and index.docker.io, if present, are dropped. The optional /library prefix
for official images from Docker Hub, if included, is also dropped. Valid,
non-Docker Hub repository URLs will be returned unchanged.

This is useful for the purposes of comparison and also in cases where a
canonical representation of a repository URL is needed. Any URL that cannot
be normalized will be returned as-is.

It has one argument:

- `url` (Required): The URL of an image repository.

The returned value is a `string`.

#### normalizeChart(url)

Normalizes a chart repository URL for purposes of comparison.
Crucially, this function removes the oci:// prefix from the URL if there is
one.

It has one argument:

- `url` (Required): The URL of a chart repository.

The returned value is a `string`.

## Retrieving the Receiver's URL

Kargo will generate a hard-to-guess URL from the receiver's configuration. This
URL can be obtained using a command such as the following:

```shell
kubectl get projectconfigs kargo-demo \
  -n kargo-demo \
  -o=jsonpath='{.status.webhookReceivers}'
```
