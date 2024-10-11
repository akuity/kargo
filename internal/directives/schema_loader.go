package directives

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/xeipuuv/gojsonschema"
)

var (
	//go:embed schemas/*
	embeddedSchemasFS embed.FS
	schemasFS         = http.FS(embeddedSchemasFS)
)

func getConfigSchemaLoader(name string) gojsonschema.JSONLoader {
	return gojsonschema.NewReferenceLoaderFileSystem(
		fmt.Sprintf("file:///schemas/%s-config.json", name),
		schemasFS,
	)
}
