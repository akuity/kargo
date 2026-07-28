# Chart Unit Tests

Uses [helm-unittest](https://github.com/helm-unittest/helm-unittest) v1.x (pinned to v1.1.0). Run with
`make test-chart` from the repo root.

All suites load `test/values-unittest.yaml` as a baseline. It contains only the values the chart
*hard-requires* to render at all (the API admin-account `passwordHash` / `tokenSigningKey`, which the
API Secret demands via `fail`). It sets **no feature flags** — those, and any value a test description
explicitly names ("when OIDC is enabled", "when crds.keep is false"), are set via `set:` in the
individual test case.

## File layout

One test file per component directory, named `<component>_test.yaml`. Each file holds one or more
suites, one suite per template file.

```
tests/
  _helpers_test.yaml                    # helpers from _helpers.tpl
  crds_test.yaml                        # crds.yaml
  namespaces_test.yaml                  # {system,shared,cluster-secrets}-resources-namespace
  common_test.yaml                      # common/ (shared cluster roles, cert issuer)
  api_test.yaml                         # api/*
  controller_test.yaml                  # controller/*
  management-controller_test.yaml       # management-controller/*
  garbage-collector_test.yaml           # garbage-collector/*
  kubernetes-webhooks-server_test.yaml  # kubernetes-webhooks-server/*
  external-webhooks-server_test.yaml    # external-webhooks-server/*
  dex-server_test.yaml                  # dex-server/*
  users_test.yaml                       # users/*
  argocd_test.yaml                      # argocd/*
```

## Suite naming

Suite names mirror the template path without the `templates/` prefix:
- Component templates: `api/deployment.yaml`
- Helpers: `_helpers.tpl/kargo.baseURL`

## Template scope: suite-level vs leaf-level

**Suite-level `templates:`** — for templates that render standalone.

**Leaf-level `template:`** (per-test) — required for the workload templates (api, controller,
management-controller, kubernetes/external webhooks server Deployments and the garbage-collector
CronJob) which `include` their sibling ConfigMap/Secret via `print $.Template.BasePath ...` to compute
a checksum annotation. Those siblings must be listed at suite level so the include resolves; each test
then scopes assertions with `template:`.

## Common patterns

| Pattern | When to use |
|---|---|
| `isKind` + `equal path: metadata.name` | Verify a template renders the right object |
| `hasDocuments: count: N` | Count documents in a multi-document template |
| `documentSelector: {path: metadata.name, value: X}` | Scope assertions to one document in a multi-doc file |
| `notExists` / `exists` | Verify a field is absent/present per feature flag |
| `contains` / `notContains` | List membership (RBAC rules, env, webhook operations) |
| `matchRegex` | Image refs / derived URLs |
| Bracket notation `metadata.annotations["helm.sh/resource-policy"]` | Keys containing dots |
