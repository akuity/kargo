package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server"
)

// TestGetClientV2 exercises the v2 client end to end against a fake API
// server: transport wiring (bearer token and CLI version headers, URL
// construction) and wire fidelity of the display path -- a Stage fetched
// through the client and round-tripped into kargoapi.Stage must reproduce
// the server's response exactly, with no phantom empty objects for unset
// optional fields.
func TestGetClientV2(t *testing.T) {
	served := &kargoapi.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kargo-demo",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "my-warehouse",
				},
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	servedJSON, err := json.Marshal(served)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			require.Equal(
				t,
				"/v1beta1/projects/kargo-demo/stages/test",
				r.URL.Path,
			)
			require.Equal(
				t, "Bearer test-token", r.Header.Get("Authorization"),
			)
			require.NotEmpty(t, r.Header.Get(server.CLIVersionHeader))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(servedJSON)
		},
	))
	defer srv.Close()

	apiClient, err := GetClientV2(srv.URL, "test-token", false)
	require.NoError(t, err)

	res, httpRes, err := apiClient.CoreAPI.
		GetStage(t.Context(), "kargo-demo", "test").
		Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	require.NoError(t, err)

	resJSON, err := json.Marshal(res)
	require.NoError(t, err)
	var displayed *kargoapi.Stage
	require.NoError(t, json.Unmarshal(resJSON, &displayed))
	displayedJSON, err := json.Marshal(displayed)
	require.NoError(t, err)

	require.JSONEq(t, string(servedJSON), string(displayedJSON))
	require.NotContains(t, string(resJSON), `"promotionTemplate":{}`)
	require.NotContains(t, string(resJSON), `"health":{}`)
}
