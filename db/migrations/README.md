# Database migrations

This directory holds the Goose SQL migrations that define Kargo's database
schema. Tilt watches it and runs `make db-migrate` on startup and whenever a
file here changes, so every migration in this directory is applied to the
local PostgreSQL database as soon as it is saved.

Because of that, do not draft migrations in this directory. A half-written
file is applied the moment it is saved, and Goose records it as done. Editing
the file afterward does not run it again, so the database ends up with the
draft's schema and no clean way back to the intended one.

## Workflow

1. Create the migration in a scratch directory outside the repository:

   ```shell
   draft_dir=$(mktemp -d)
   go tool goose -dir "$draft_dir" create create_widgets sql
   ```

2. Edit the generated file until the `Up` and `Down` statements are the ones
   you want to keep.

3. Move it here. Tilt applies it within a few seconds:

   ```shell
   mv "$draft_dir"/*.sql db/migrations/
   ```

4. Verify the result with `go tool goose status` or `psql`.

If a migration that has already been applied locally turns out to be wrong,
roll it back with `go tool goose down` before editing it, or add a new
migration. Do not edit an applied migration in place.

Queries that read from or write to these tables live in `db/queries/`, and
`sqlc.yaml` reads this directory for the schema. Keep a table's migration, its
queries, and the generated code in the same change.

See [the contributor guide](../../docs/docs/60-contributor-guide/10-hacking-on-kargo.md#working-with-postgresql)
for connection details and the environment variables the Goose commands above
expect.
