package apply

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
)

const testManifest = "apiVersion: kargo.akuity.io/v1alpha1\nkind: Stage\nmetadata:\n" +
	"  name: test\n  namespace: kargo-demo\n"

func TestApplyOptions_Run(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "stage.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(testManifest), 0o600))

	respBody, err := json.Marshal(map[string]any{
		"results": []map[string]any{
			{
				"createdResourceManifest": map[string]any{
					"apiVersion": "kargo.akuity.io/v1alpha1",
					"kind":       "Stage",
					"metadata": map[string]any{
						"name":      "test",
						"namespace": "kargo-demo",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	var gotMethod, gotUpsert, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotUpsert = r.URL.Query().Get("upsert")
			gotContentType = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(respBody)
		},
	))
	defer srv.Close()

	streams, _, out, _ := genericiooptions.NewTestIOStreams()
	o := &applyOptions{
		Config:        config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"},
		IOStreams:     streams,
		PrintFlags:    genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
		ClientOptions: client.Options{},
		Filenames:     []string{manifestPath},
	}

	require.NoError(t, o.run(t.Context()))

	// Confirm the request the v2 client actually sent on the wire matches
	// what the old client sent: a PUT of the raw manifest text with
	// upsert=true as a query parameter (openapi-generator sends the
	// manifest as a text/plain body, not JSON). option.ReadManifests
	// re-serializes the manifest through a YAML encoder, so compare
	// semantically (decoded content) rather than byte-for-byte.
	require.Equal(t, http.MethodPut, gotMethod)
	require.Equal(t, "true", gotUpsert)
	require.Equal(t, "text/plain", gotContentType)
	var gotManifest, wantManifest map[string]any
	require.NoError(t, yaml.Unmarshal(gotBody, &gotManifest))
	require.NoError(t, yaml.Unmarshal([]byte(testManifest), &wantManifest))
	require.Equal(t, wantManifest, gotManifest)

	require.Contains(t, out.String(), "test")
}

func TestApplyOptions_Run_ResourceError(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "stage.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(testManifest), 0o600))

	respBody, err := json.Marshal(map[string]any{
		"results": []map[string]any{
			{"error": "resource is invalid"},
		},
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(respBody)
		},
	))
	defer srv.Close()

	streams, _, _, _ := genericiooptions.NewTestIOStreams()
	o := &applyOptions{
		Config:        config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"},
		IOStreams:     streams,
		PrintFlags:    genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
		ClientOptions: client.Options{},
		Filenames:     []string{manifestPath},
	}

	err = o.run(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "resource is invalid")
}
