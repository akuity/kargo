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

## Kargo Metrics

Alongside the Go runtime and controller-runtime metrics that every component
reports, the controller exports its own metrics, all prefixed with `kargo_` and
labeled by `project`:

| Metric | Type | Description |
|--------|------|-------------|
| `kargo_promotions_created_total` | Counter | Number of `Promotion` resources the controller has observed being created. |
| `kargo_promotions_errored_retryable_total` | Counter | Number of retryable errors encountered while executing Promotions. A single Promotion may contribute more than once, since each failed attempt is counted. |
| `kargo_promotions_errored_terminal_total` | Counter | Number of Promotions that reached the terminal `Errored` phase. |
| `kargo_promotions_non_terminal` | Gauge | Number of Promotions currently in a non-terminal phase (`Pending` or `Running`), sampled at scrape time. |
| `kargo_promotion_duration_seconds` | Histogram | Time each Promotion spent running, from the moment it started until it reached a terminal phase. |
| `kargo_promotion_step_duration_seconds` | Histogram | Time each individual promotion step spent running, from the moment it started until it reached a terminal status. |
| `kargo_stages_by_ready_reason` | Gauge | Number of Stages in each readiness state, by Project, sampled at scrape time. Control flow Stages are omitted. |

:::note

The `project` label is the name of the Kargo `Project` (namespace) the
`Promotion` belongs to. Each controller only reports on the Promotions it is
responsible for, so queries spanning a sharded topology should aggregate across
controllers. Your scrape config should include a desired shard name to attach as
a label to the metrics, so that you can filter by shard in queries.

The two gauges are sampled from the controller's cache at scrape time and
report only the Projects that currently have something to report. A Project
with no non-terminal Promotions produces no `kargo_promotions_non_terminal`
series at all, rather than a zero, so use `or vector(0)` in queries that need a
value when a Project is idle.

:::

### Promotion Duration

`kargo_promotion_duration_seconds` measures only the time a `Promotion` spent
actually running: the interval between its `startedAt` and `finishedAt`
timestamps. Time spent `Pending`, waiting for its `Stage` to acknowledge it, is
excluded, as are Promotions that reached a terminal phase without ever having
started.

In addition to `project`, it carries an `terminalPhase` label that has the phase
with which the promotion finished for more granular queries.

Because promotion durations span a very wide range, this histogram also emits a
[native histogram](https://prometheus.io/docs/specs/native_histograms/).
Scraping the native histogram rather than the static buckets gives a much more
accurate view of the distribution, if your Prometheus server supports it.

### Promotion Step Duration

`kargo_promotion_step_duration_seconds` breaks the same interval down by
individual step. A step's clock starts the first time it runs and stops when it
reaches a terminal status, so a step that waits across several reconciliations
-- one waiting on a pull request to merge, for instance -- reports its whole
lifetime, not just the attempt that finished it. Steps that were skipped, and
steps still waiting, are not recorded.

In addition to `project`, it carries a `stepKind` label with the step's kind
(`git-clone`, `argocd-update`, and so on) and a `terminalStatus` label with the
status the step ended in:

| `terminalStatus` | Meaning |
|------------------|---------|
| `Succeeded` | The step completed successfully. |
| `Failed` | The step failed for non-technical reasons. This is terminal -- the step is not retried. |
| `Errored` | The step failed for technical reasons. Kargo retries the step until its error threshold is met, so the observed duration covers every attempt. |

Skipped steps are not recorded, whether they were skipped by their `if`
condition or determined on their own that they had nothing to do.

Because step durations span a very wide range, this histogram also emits a
[native histogram](https://prometheus.io/docs/specs/native_histograms/).
Scraping the native histogram rather than the static buckets gives a much more
accurate view of the distribution, if your Prometheus server supports it.

### Stages by Ready Reason

Unlike a `Promotion`, a `Stage` has no single phase. Its state is expressed
through Kubernetes conditions, and the controller already collapses those into
a single `Ready` condition whose `reason` names the most important thing
currently true of the Stage. `kargo_stages_by_ready_reason` counts Stages by
that reason, so every Stage lands in exactly one series and the series sum to
the number of Stages in the Project (excluding control flow Stages, see below).

Alongside `project`, it carries a `reason` label:

```
kargo_stages_by_ready_reason{project="guestbook",reason="Verified"}         10
kargo_stages_by_ready_reason{project="guestbook",reason="ActivePromotion"}   3
kargo_stages_by_ready_reason{project="guestbook",reason="Unhealthy"}         1
```

This table summarizes the reasons that Stages report, and what they mean:

| Reason | Meaning |
|--------|---------|
| `ReconcileError` | The controller hit an error reconciling the Stage and will retry. |
| `ActivePromotion`, `ListPromotionsFailed` | A Promotion is in progress, or the controller could not determine whether one is. |
| `LastPromotionFailed`, `LastPromotionErrored`, `LastPromotionAborted` | No Promotion is running, but the last one did not succeed. |
| `Unhealthy`, `Progressing`, `NoFreight`, `WaitingForHealthCheck`, `LastPromotion*` | The Stage is not healthy, or its health cannot yet be assessed. |
| `PendingVerification`, `VerificationPending`, `VerificationRunning`, `VerificationFailed`, `VerificationError`, `VerificationAborted`, `VerificationInconclusive` | The Stage is healthy, but its Freight is not verified. |
| `Verified` | The Stage is ready. |
| `Unknown` | The controller has not reconciled the Stage yet, so it has no `Ready` condition. |

Control flow Stages are omitted. Their `Ready` condition is set directly and
never accounts for health or verification, so the reason they report conveys no
useful information. A Project containing only control flow Stages reports no
series at all.

:::note

A Stage is only `Verified` once it is healthy **and** its Freight has passed
verification. Stages with no `spec.verification` are not stuck: the controller
records a synthetic successful verification for them, so they reach `Verified`
like any other Stage.

Conversely, not-`Verified` is a broad bucket. It covers genuinely broken Stages
alongside entirely normal ones: a Stage that is mid-promotion, or one that
has never been promoted at all. Your alerts should trigger on specific reasons
rather than on "everything that is not `Verified`".

:::

:::info

This gauge is sampled from the controller's cache at scrape time, and a
per-Project breakdown has to walk every Stage to produce it, so its cost scales
with the number of Stages in the cluster.

Only the states actually observed are reported. A reason that hasn't been
reported by any Stage in the project produces no series at all rather than a
zero, so use `or vector(0)` in queries that need a value when a state is empty.

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
