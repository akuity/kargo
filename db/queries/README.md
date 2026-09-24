# Database queries

Put sqlc query files directly in this directory. `sqlc.yaml` reads the schema
from Goose migrations in `db/migrations` and generates pgx/v5 Go code under
`pkg/database`.

Run `make codegen-db` after changing SQL. `make codegen` also includes this step.
Generation does nothing until this directory contains a `.sql` query file.
Keep resource migrations, queries, and generated code in the same change.
