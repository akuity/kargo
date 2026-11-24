// Package db embeds Kargo's database schema migrations so that the binary
// that runs against a schema is the same one that can migrate it.
package db

import (
	"embed"
	"fmt"
	"io/fs"
)

// migrations holds every SQL migration under db/migrations. go:embed cannot
// reach outside this directory, which is why this package lives next to them.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrations returns the embedded SQL migrations as a filesystem rooted at
// the migrations directory, in the layout Goose expects.
func Migrations() (fs.FS, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("error opening embedded migrations: %w", err)
	}
	return fsys, nil
}
