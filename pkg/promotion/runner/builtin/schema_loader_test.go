package builtin

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"

	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
)

func TestGetConfigSchemaLoader(t *testing.T) {
	schemaFiles, err := fs.Glob(stepconfig.Schemas, "schemas/*-config.json")
	require.NoError(t, err)
	require.NotEmpty(t, schemaFiles)
	for _, schemaFile := range schemaFiles {
		name := strings.TrimSuffix(
			strings.TrimPrefix(schemaFile, "schemas/"),
			"-config.json",
		)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := gojsonschema.NewSchema(getConfigSchemaLoader(name))
			require.NoError(t, err)
		})
	}
}
