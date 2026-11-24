# Database queries

Put sqlc query files directly in this directory. `sqlc.yaml` reads the schema
from Goose migrations in `db/migrations` and generates pgx/v5 Go code under
`pkg/database`.

Tilt regenerates the code whenever this directory or `db/migrations` changes.
Outside Tilt, run `make codegen-db`; `make codegen` also includes this step.
Keep migrations, queries, and generated code in the same change.
