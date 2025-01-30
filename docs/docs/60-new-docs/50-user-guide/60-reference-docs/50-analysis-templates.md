---
sidebar_label: Analysis Templates
description: Learn about AnalysisTemplate for verification
---

# AnalysisTemplate Reference

An `AnalysisTemplate` is a resource that defines how to perform verification testing, including:

* Container images and commands to run
* Queries to external monitoring tools
* How to interpret results from metric providers
* Success or failure criteria
* Frequency and duration of measurements

`AnalysisTemplate` resources (and the `AnalysisRun` resources that are spawned from them) are CRDs re-used from the [Argo Rollouts](https://argoproj.github.io/argo-rollouts) project. They were intentionally built to be useful in contexts other than Argo Rollouts. Re-using this resource type to define verification processes means those processes benefit from this rich and battle-tested feature of Argo Rollouts.

:::info
This reference guide is intended to give a brief introduction to `AnalysisTemplate`s for some common use cases. Please consult the [relevant sections](https://argoproj.github.io/argo-rollouts/features/analysis/) of the Argo Rollouts documentation for comprehensive coverage of the full range of `AnalysisTemplate` capabilities.
:::

`AnalysisTemplate`s integrate natively with many popular open-source and commercial monitoring tools, including:

* [Prometheus](https://prometheus.io/)
* [DataDog](https://www.datadoghq.com/)
* [Amazon CloudWatch](https://aws.amazon.com/cloudwatch/)
* [NewRelic](https://newrelic.com/)
* [InfluxDB](https://www.influxdata.com/)
* [Apache SkyWalking](https://skywalking.apache.org/)
* [Graphite](https://graphiteapp.org/)

In addition to monitoring tools, analysis can integrate with internal systems by:

* Running containerized processes as Kubernetes `Job`s
* Making HTTP requests and interpreting JSON responses


## Arguments

`AnalysisTemplate`s may declare a set of arguments that can be "passed" in by the `Stage`. The arguments are resolved at the time the `AnalysisRun` is created and can then be referenced in metrics configuration. Arguments are dereferenced using the syntax: `{{ args.<name> }}`.

The following example shows an `AnalysisTemplate` with three arguments. Values for arguments can have a default value, supplied by the `Stage`, or obtained from a `Secret` if the value is sensitive (e.g. a bearer token for an HTTP request):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: args-example
spec:
  args:
  # An argument can specify a value to be used as its default.
  # This will be overridden by a value supplied by the Stage.
  - name: api-url
    value: http://example/measure
  # If an argument specifies no value, it is considered a required
  # argument and must be supplied by the Stage.
  - name: service-name
  # Arguments can be obtained from a Secret in the Project Namespace
  - name: api-token
    valueFrom:
      secretKeyRef:
        name: token-secret
        key: apiToken
  metrics:
  - name: webmetric
    successCondition: result == 'true'
    provider:
      web:
        # placeholders are resolved when an AnalysisRun is created
        url: "{{ args.api-url }}?service={{ args.service-name }}"
        headers:
        - key: Authorization
          value: "Bearer {{ args.api-token }}"
        jsonPath: "{$.results.ok}"
```

## Success Condition

When interpreting the result of a query, an [expr](https://expr-lang.org/) expression can be used to evaluate the response. The response payload is set in a variable `result`. The following will interpret the response of a Prometheus query, and require that the element of the returned vector is greater than or equal to `0.95`:


```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  metrics:
  - name: success-rate
    # Expr expression that can be evaluated to true or false
    # NOTE: prometheus queries return results in the form of a vector.
    # So it is common to access the index 0 of the returned array to obtain the value
    successCondition: result[0] >= 0.95
    provider:
      prometheus:
        address: "http://prometheus.example.com:9090"
        query: |
          sum(irate(
            istio_requests_total{reporter="source",response_code!~"5.*"}[5m]
          )) /
          sum(irate(
            istio_requests_total{reporter="source"}[5m]
          ))
```

## Failure Conditions and Limits

As an alternative to `successCondition`, a `failureCondition` can be used to describe when a measurement is considered failed. Additionally, `failureLimit` can also be used to specify the maximum number of failed measurements that are allowed before the entire `AnalysisRun` is considered `Failed`.

The following example continually polls a Prometheus server to get the total number of errors (i.e., HTTP response code >= 500) every 5 minutes, causing the measurement to fail if ten or more errors are encountered. The entire analysis run is considered as Failed after three failed measurements.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: failure-condition-example
spec:
  metrics:
  - name: total-errors
    interval: 5m
    failureCondition: result[0] >= 10
    failureLimit: 3
    provider:
      prometheus:
        address: http://prometheus.example.com:9090
        query: |
          sum(irate(
            istio_requests_total{reporter="source",response_code=~"5.*"}[5m]
          ))
```

## Delaying Measurements

In some scenarios, it may be necessary to delay the start of a metric measurement. For example, some time may need to pass after an update in order for new data to populate in the monitoring services. The `initialDelay` option can be used to delay the start of measurements. Each metric can be configured to have a different delay.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: initial-delay-example
spec:
  metrics:
  - name: success-rate
    # Duration before measurement collection. Default is no delay
    initialDelay: 5m
    successCondition: result[0] >= 0.90
    provider:
      prometheus:
        address: http://prometheus.example.com:9090
        query: ...
```

## Example Metric Types

### Web

An HTTP request can be performed against some external service to obtain the measurements.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: web-metric-example
spec:
  args:
  - name: api-token
    valueFrom:
      secretKeyRef:
        name: token-secret
        key: apiToken
  metrics:
  - name: webmetric
    successCondition: result == true
    provider:
      web:
        url: "http://example.com/api/v1/measurement"
        # HTTP Method. valid values are GET|POST|PUT. Defaults to GET
        method: POST
        # Timeout for the request. Defaults to 10 seconds
        timeoutSeconds: 20 
        headers:
        - key: Authorization
          value: "Bearer {{ args.api-token }}"
          # if body is a json, it is recommended to set the Content-Type
        - key: Content-Type 
          value: "application/json"
        # Requst body to send. 
        body: |
          {"foo": "bar"}
        # Optional JSON path to set the value of `result` in successCondition/failureCondition
        jsonPath: "{$.data.ok}"
```

### Job

A Kubernetes `Job` can be used to perform analysis. When a `Job` is used, the metric is considered successful if the `Job` completes with an exit code of zero and is otherwise considered to have failed.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: integration-test
  namespace: guestbook
spec:
  metrics:
  - name: integration-test
    provider:
      job:
        spec:
          template:
            spec:
              containers:
              - name: sleep
                image: alpine:latest
                command: [sleep, "10"]
              restartPolicy: Never
          backoffLimit: 1
```
