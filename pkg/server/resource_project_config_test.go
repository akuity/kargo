package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/projectconfigs"
)

// Test_server_projectConfig_getOutputReappliesThroughResources reads a
// ProjectConfig from its own route and sends it back through the generic
// resources endpoint as `kargo get projectconfig -o yaml | kargo apply -f -`
// does. That only works if what the route serves carries apiVersion and
// kind, which the generic endpoint routes manifests by.
func Test_server_projectConfig_getOutputReappliesThroughResources(t *testing.T) {
	project := testResourceProject("my-project", "my-uid")
	store := storeHolding(project)
	var captured *fakeProjectConfigStore

	// Read the ProjectConfig as the CLI does: decode the response into the
	// ProjectConfig type, then print it as a resource, with apiVersion and
	// kind, as YAML.
	var manifest []byte
	testRESTEndpoint(
		t, nil,
		http.MethodGet, "/v1beta1/projects/"+project.Name+"/config",
		[]restTestCase{{
			name:          "get",
			clientBuilder: fake.NewClientBuilder().WithObjects(project),
			serverSetup:   withProjectConfigStore(store, &captured, nil),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Contains(t, w.Body.String(), `"apiVersion":"kargo.akuity.io/v1alpha1"`)
				require.Contains(t, w.Body.String(), `"kind":"ProjectConfig"`)
				cfg := &kargoapi.ProjectConfig{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), cfg))
				// Stand in for an edit the user makes before applying.
				cfg.Spec.FreightLinks = []kargoapi.DeepLink{{
					Title: "edited",
					URL:   "https://example.com/edited",
				}}
				var err error
				manifest, err = yaml.Marshal(projectconfigs.ResponseBody(cfg))
				require.NoError(t, err)
			},
		}},
	)
	require.Contains(t, string(manifest), "apiVersion: kargo.akuity.io/v1alpha1")
	require.Contains(t, string(manifest), "kind: ProjectConfig")

	testRESTEndpoint(
		t, nil,
		http.MethodPut, "/v1beta1/resources",
		[]restTestCase{{
			name:          "apply",
			clientBuilder: fake.NewClientBuilder().WithObjects(project),
			serverSetup:   withProjectConfigStore(store, &captured, nil),
			body:          strings.NewReader(string(manifest)),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				var res createOrUpdateResourceResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
				require.Len(t, res.Results, 1)
				require.Empty(t, res.Results[0].Error)
				updated := res.Results[0].UpdatedResourceManifest
				require.Equal(t, kargoapi.GroupVersion.String(), updated["apiVersion"])
				require.Equal(t, "ProjectConfig", updated["kind"])

				stored := captured.configs[project.Name]
				require.Equal(t, int64(2), stored.Generation)
				require.Contains(t, string(stored.Spec), "https://example.com/edited")
			},
		}},
	)
}
