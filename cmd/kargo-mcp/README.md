# kargo-mcp

An [MCP](https://modelcontextprotocol.io/) server that exposes
[Kargo](https://kargo.akuity.io/) as a set of tools for AI agents.
Use it with Claude Code (or any MCP client) to query pipeline state,
inspect freight and promotions, and trigger or approve promotions.

## Build

```bash
make build-mcp
```

Produces `bin/kargo-mcp`.

## Authentication

`kargo-mcp` reuses the credentials stored by the Kargo CLI. Log in once
with `kargo login` and the server picks up the session automatically:

```bash
kargo login https://kargo.example.com
```

For non-interactive or CI use, set environment variables instead:

```bash
export KARGO_API_ADDRESS=https://kargo.example.com
export KARGO_API_TOKEN=<bearer-token>
```

Environment variables take priority over the credential file. If your
token expires, run `kargo login` again — the server surfaces a clear
error message asking you to do so.

## Configure Claude Code

The project already ships a `.mcp.json` that points at `bin/kargo-mcp`:

```json
{
  "mcpServers": {
    "kargo": {
      "type": "stdio",
      "command": "bin/kargo-mcp",
      "args": []
    }
  }
}
```

Run `/mcp` in Claude Code to connect (or reconnect after a rebuild).

For a different Kargo instance, pass environment variables:

```json
{
  "mcpServers": {
    "kargo-prod": {
      "type": "stdio",
      "command": "/path/to/kargo-mcp",
      "args": [],
      "env": {
        "KARGO_API_ADDRESS": "https://kargo.prod.example.com",
        "KARGO_API_TOKEN": "${KARGO_PROD_TOKEN}"
      }
    }
  }
}
```

Multiple named servers compose transparently — register one per Kargo
instance.

## Tools

### Server

| Tool | Description |
|------|-------------|
| `get_version_info` | Kargo server version |

### Projects

| Tool | Description |
|------|-------------|
| `list_projects` | All projects (compact summary) |
| `get_project` | Full project details |

### Stages

| Tool | Description |
|------|-------------|
| `list_stages` | Stages in a project — filter by `warehouses` or `health` |
| `get_stage` | Full stage details (spec + current status, without freight history) |
| `get_stage_freight_history` | Freight history for a stage with verification results |
| `refresh_stage` | Trigger an out-of-band stage refresh |

### Warehouses

| Tool | Description |
|------|-------------|
| `list_warehouses` | Warehouses in a project (compact summary) |
| `get_warehouse` | Full warehouse details including discovered artifacts |
| `refresh_warehouse` | Trigger an out-of-band warehouse refresh |

### Freight

| Tool | Description |
|------|-------------|
| `list_freight` | Freight in a project, newest first — filter by `stage` (eligible) or `origins` |
| `get_freight` | Full freight details by name or alias |
| `approve_freight` | Manually approve freight for a stage, bypassing verification |

### Promotions

| Tool | Description |
|------|-------------|
| `list_promotions` | Promotions, newest first — filter by `stage` and/or `phase` |
| `get_promotion` | Full promotion details with step execution trace |
| `promote_to_stage` | Promote freight to a specific stage |
| `promote_downstream` | Promote freight to all stages downstream of a given stage |
| `abort_promotion` | Abort a non-terminal promotion |

### Promotion Tasks

| Tool | Description |
|------|-------------|
| `list_promotion_tasks` | Reusable PromotionTask templates in a project |
| `list_cluster_promotion_tasks` | Cluster-scoped PromotionTask templates |

## Design notes

- `list_*` tools return compact summaries. Use `get_*` for full details.
- `get_*` responses strip Kubernetes bookkeeping (`managedFields`,
  `resourceVersion`, `last-applied-configuration` annotation) and null
  fields to keep context small.
- `get_stage_freight_history` is intentionally separate from `get_stage`
  — freight history can be large and is only needed for auditing.
- Auth is handled entirely client-side: the server reads credentials
  from `~/.config/kargo/config` (written by `kargo login`) and handles
  OIDC token refresh transparently.
