package builtin

import (
	"fmt"
	"net/http"

	"github.com/xeipuuv/gojsonschema"

	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
)

var schemasFS = http.FS(stepconfig.Schemas)

func getConfigSchemaLoader(name string) gojsonschema.JSONLoader {
	return gojsonschema.NewReferenceLoaderFileSystem(
		fmt.Sprintf("file:///schemas/%s-config.json", name),
		schemasFS,
	)
}
