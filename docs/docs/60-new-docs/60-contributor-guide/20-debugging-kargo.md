---
description: Steps to help you debug Kargo when you run into issues.
sidebar_label: Debugging Kargo
---

# Debugging Kargo

From time to time, you may need to debug Kargo. This document outlines options
to help you achieve a better understanding of what Kargo (internally) is doing,
especially when you are running into issues which are not immediately obvious
by just reading the code.

As a Kargo user, you may not need to debug Kargo itself, but this document may
serve you when a Kargo maintainer asks you to provide additional information
about an issue you are facing.

## Enabling pprof endpoints

Kargo components can be configured to expose [`pprof` endpoints](https://golang.org/pkg/net/http/pprof/).
These endpoints can be used to profile the components when they are running,
and can be useful to understand what the components are doing and where they
are spending time.

To enable the `pprof` endpoint on a component, you can set the
`PPROF_BIND_ADDRESS` environment variable to the address where the component
should listen for `pprof` requests. For example, to enable the `pprof` endpoint
on port `6060` of the controller, you can set the `PPROF_BIND_ADDRESS`
environment variable to `:6060`.

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kargo-controller
spec:
  # ...omitted for brevity
  template:
    spec:
      containers:
      - name: kargo-controller
        env:
        - name: PPROF_BIND_ADDRESS
          value: ":6060"
```

After setting the `PPROF_BIND_ADDRESS` environment variable, the `pprof`
endpoints will be available at `http://<controller-ip>:6060/debug/pprof/`.

### Collecting a profile

To collect a profile, you can port-forward the `pprof` address to your local
machine and collect the data from an endpoint of choice. For example, to
collect a heap profile, you can run:

```console
$ kubectl port-forward -n <namespace> deployment/<component> 6060
$ curl -Sk -v http://localhost:6060/debug/pprof/heap > heap.out
```

This will collect a heap profile in the `heap.out` file, which you can then
[analyze using `go`](https://go.dev/blog/pprof), or share with a Kargo
maintainer.
