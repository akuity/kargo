-- name: GetDataImport :one
SELECT * FROM data_imports
WHERE name = sqlc.arg(name);

-- Recording an import that is already recorded keeps the first record.
-- name: RecordDataImport :exec
INSERT INTO data_imports (name)
VALUES (sqlc.arg(name)::text)
ON CONFLICT (name) DO NOTHING;
