# Database migrations

This directory holds the Goose SQL migrations that define Kargo's database
schema. Tilt applies them to the local PostgreSQL database once at startup.
After that, changes here are applied only when you trigger the `db-migrate`
resource, from the Tilt UI or with:

```shell
hack/bin/tilt trigger db-migrate
```

Tilt still watches the directory and marks `db-migrate` as having pending
changes, so a saved but unapplied migration is visible without being run.

## Workflow

1. Create the migration here:

   ```shell
   go tool goose -dir db/migrations create create_widgets sql
   ```

2. Edit the generated file until the `Up` and `Down` statements are final.

3. Trigger `db-migrate`, then inspect the result:

   ```shell
   hack/bin/tilt trigger db-migrate
   make db-shell
   ```

Goose records each applied migration by version and never re-runs it, so an
applied migration that turns out to be wrong must be rolled back with
`go tool goose down` before it is edited. Once a migration has merged, add a
new one instead.

Queries that read from or write to these tables live in `db/queries/`, and
`sqlc.yaml` reads this directory for the schema. Keep a table's migration, its
queries, and the generated code in the same change.

See [the contributor guide](../../docs/docs/60-contributor-guide/10-hacking-on-kargo.md#working-with-postgresql)
for connection details and the environment variables the Goose commands above
expect.
