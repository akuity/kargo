# yaml_parse_update

Parses a value from a YAML file in the repo and writes it into another field with `yaml-update`, then verifies (via a second `yaml-parse`) that the field was updated. Nothing is committed back to the repo.

## Required environment context

This suite reads the following from the `context` section of the env file
passed with `-env-file` (see [`../../envs`](../../envs)):

| Variable | Description |
| --- | --- |
| `kargo_demo_gitops_repo` | HTTPS URL of a fork of the `kargo-demo-gitops` repository. Substituted into the fixtures at runtime (the Warehouse subscription, the promotion's `gitRepo` var, and/or the Argo CD `ApplicationSet` source). |

Example:

```yaml
context:
  kargo_demo_gitops_repo: https://github.com/<you>/kargo-demo-gitops.git
```
