# Database migrations

Put completed Goose SQL migrations in this directory. Tilt runs `make db-migrate`
on startup and whenever this directory changes, applying pending migrations to
the local PostgreSQL database.

Create and edit drafts outside this directory, then move the completed SQL file
here. This prevents Tilt from applying an unfinished migration. Editing an
already-applied migration does not run it again; add a new migration or explicitly
roll back the previous one during local development.

See [the contributor guide](../../docs/docs/60-contributor-guide/10-hacking-on-kargo.md#working-with-postgresql)
for connection details and an example workflow.
