---
sidebar_label: Monitoring
description: Expose and scrape Prometheus metrics from Kargo's controllers.
---

# Monitoring

Several of Kargo's long-running components expose
[Prometheus](https://prometheus.io/) metrics through the underlying
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
metrics server:

- The controller
- The management controller
- The (internal) webhooks server

By default, each component's metrics server is disabled. The Kargo Helm chart
can enable it, expose the metrics through a `Service`, and -- for users of the
[Prometheus Operator](https://prometheus-operator.dev/) -- create a matching
`ServiceMonitor`.

:::info

For complete parameter documentation, refer to the
[chart documentation](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md).

:::

## Enabling Metrics

Setting a component's `metrics.enabled` flag to `true` does three things:

1. Binds the component's metrics server (by setting `METRICS_BIND_ADDRESS`).
1. Declares a metrics container port on the component's pods.
1. Creates a metrics `Service` that targets that port.

For example, to expose metrics for all three supported components:

```yaml
controller:
  metrics:
    enabled: true
managementController:
  metrics:
    enabled: true
webhooksServer:
  metrics:
    enabled: true
```

Each component's metrics are served over plain HTTP on port `9090` by default,
named `http-metrics`; scrapers do not need to be configured for TLS. The port
name and number are configurable per component, for example:

```yaml
controller:
  metrics:
    enabled: true
    service:
      servicePort: 8080
      portName: telemetry
```

:::note

The default port name (`http-metrics`) carries an `http-` prefix so that
service meshes that infer a port's protocol from its name treat it as HTTP. If
you rename the port for a meshed cluster, keep the `http-` prefix.

:::

## Scraping With the Prometheus Operator

If your cluster runs the
[Prometheus Operator](https://prometheus-operator.dev/), Kargo can create a
`ServiceMonitor` for each component. Enable it alongside `metrics.enabled`:

```yaml
controller:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      # Often required so your Prometheus instance selects the ServiceMonitor.
      additionalLabels:
        release: prometheus
      interval: 30s
```

The `ServiceMonitor` is only rendered when **both** `metrics.enabled` and
`metrics.serviceMonitor.enabled` are `true` **and** the Prometheus Operator
CRDs (`monitoring.coreos.com/v1`) are present in the cluster. If the CRDs are
absent, the `ServiceMonitor` is silently skipped so that installation does not
fail.

:::info

A `ServiceMonitor` does not scrape the `Service`'s cluster IP. The Prometheus
Operator discovers the `Service`'s endpoints and scrapes each backing pod
individually, so per-replica metrics are preserved even when a component runs
more than one replica.

:::

Additional `serviceMonitor` fields are available for tuning the scrape, such as
`scheme`, `tlsConfig`, `relabelings`, `metricRelabelings`, and `namespace`.
Refer to the
[chart documentation](https://github.com/akuity/kargo/blob/main/charts/kargo/README.md)
for the full list.

## Scraping Without the Prometheus Operator

If you collect metrics with a tool that performs its own endpoint discovery
(for example, an annotation-based scrape config), enable `metrics.enabled` and
point your scraper at the metrics `Service`.

A headless `Service` is often convenient in this case, since it resolves
directly to individual pod IPs. Set `clusterIP` to `None`:

```yaml
controller:
  metrics:
    enabled: true
    service:
      clusterIP: "None"
```

You can also attach annotations and labels to the metrics `Service` to drive
your scraper's discovery:

```yaml
controller:
  metrics:
    enabled: true
    service:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
```
