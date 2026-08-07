---
sidebar_label: Using the CLI
description: Learn how to use the Kargo command line interface
---

# Using the Kargo CLI

The Kargo command line interface (CLI) lets you manage Kargo resources and
promote Freight from a terminal.

Before using the CLI, [install it](./index.md) and
[log in](../20-how-to-guides/10-logging-in/index.md).

## Getting help

Run `kargo --help` to list all available commands. To learn about a command,
its arguments, and its options, append `--help` to that command:

```shell
kargo promote --help
```

The CLI includes examples in the help output for many commands.

## Selecting a project

Many commands operate on a `Project`. Specify one for an individual command
with the `--project` option:

```shell
kargo get stages --project=my-project
```

To avoid repeating the option, set a default project:

```shell
kargo config set-project my-project
```

View the default project with `kargo config get-project`. Use
`kargo config set-project ""` to clear it.

## Common workflows

### Inspect Kargo resources

Use `kargo get` to list resources or retrieve a resource by name:

```shell
# List Projects
kargo get projects

# List Stages in a Project
kargo get stages --project=my-project

# Get a Stage by name
kargo get stages --project=my-project test

# List Freight produced by a Warehouse
kargo get freight --project=my-project --warehouse=my-warehouse

# List Promotions for a Stage
kargo get promotions --project=my-project --stage=test
```

Run `kargo get --help` to see every resource type that can be retrieved.

### Promote Freight

Use `kargo promote` to promote a specific piece of Freight, or to have Kargo
select the current auto-promotion candidate from a Warehouse:

```shell
# Promote Freight by name
kargo promote --project=my-project --freight=abc123 --stage=test

# Promote Freight selected from a Warehouse
kargo promote --project=my-project --warehouse=my-warehouse --stage=test
```

The `--warehouse` option accepts a Warehouse name, not an origin in
`Warehouse/name` form. See `kargo promote --help` for all promotion options,
including selecting Freight by alias and promoting to downstream Stages.

### Refresh or verify resources

Use `kargo refresh` to make Kargo reconcile a resource immediately:

```shell
kargo refresh warehouse --project=my-project my-warehouse
kargo refresh stage --project=my-project test
```

To (re)run verification for a Stage's current Freight, use:

```shell
kargo verify stage --project=my-project test
```

## Apply resource definitions

Use `kargo apply` to create or update resources from YAML:

```shell
kargo apply -f stage.yaml
```

Use `kargo create`, `kargo update`, and `kargo delete` when you prefer to
manage individual resources from the command line. Run each command with
`--help` to see supported resource types and examples.

## Shell completion

The CLI can generate completion scripts for Bash, Fish, PowerShell, and Zsh.
For example, to enable Zsh completion in the current shell:

```shell
source <(kargo completion zsh)
```

Run `kargo completion --help` and `kargo completion <shell> --help` for
installation instructions for your shell.
