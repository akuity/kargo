package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/server/config"
)

const (
	listFreightTestProject   = "fake-project"
	listFreightTestWarehouse = "fake-warehouse"
	listFreightTestStage     = "fake-stage"
)

// freightWithIndexes returns a client builder seeded with the Freight field
// indexes the ListFreight* helpers query through, so field selectors resolve
// against the fake client the way they do against the real cache. The scheme is
// set first because WithIndex must resolve the Freight kind.
func freightWithIndexes(t *testing.T) *fake.ClientBuilder {
	return fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByWarehouseField, indexer.FreightByWarehouse).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByCurrentStagesField, indexer.FreightByCurrentStages).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByVerifiedStagesField, indexer.FreightByVerifiedStages).
		WithIndex(&kargoapi.Freight{}, indexer.FreightApprovedForStagesField, indexer.FreightApprovedForStages)
}

func freightFromWarehouse(name, warehouse string) *kargoapi.Freight {
	return &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: name},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: warehouse,
		},
	}
}

// freightListNames decodes a FreightList response body and returns the set of
// Freight names it contains.
func freightListNames(t *testing.T, w *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()
	list := &kargoapi.FreightList{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
	names := make(map[string]struct{}, len(list.Items))
	for _, f := range list.Items {
		names[f.Name] = struct{}{}
	}
	return names
}

func Test_server_listFreightByCurrentStage(t *testing.T) {
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: listFreightTestProject}}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: listFreightTestStage},
	}
	freightIn := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: "in-stage"},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{listFreightTestStage: {}},
		},
	}
	freightElsewhere := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: "in-other-stage"},
		Status: kargoapi.FreightStatus{
			CurrentlyIn: map[string]kargoapi.CurrentStage{"other-stage": {}},
		},
	}
	url := "/v1beta1/projects/" + listFreightTestProject + "/stages/" + listFreightTestStage + "/freight"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, url,
		[]restTestCase{
			{
				// The endpoint no longer resolves the Stage resource, so a
				// missing Stage yields an empty list rather than a 404.
				name:          "missing Stage yields an empty list",
				clientBuilder: freightWithIndexes(t).WithObjects(project),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, freightListNames(t, w))
				},
			},
			{
				name: "returns only Freight currently in the Stage",
				clientBuilder: freightWithIndexes(t).
					WithObjects(project, stage, freightIn, freightElsewhere),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					names := freightListNames(t, w)
					require.Contains(t, names, "in-stage")
					require.NotContains(t, names, "in-other-stage")
				},
			},
		},
	)
}

func Test_server_listFreightReferencingStage(t *testing.T) {
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: listFreightTestProject}}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: listFreightTestStage},
	}
	verified := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: "verified"},
		Status: kargoapi.FreightStatus{
			VerifiedIn: map[string]kargoapi.VerifiedStage{listFreightTestStage: {}},
		},
	}
	approved := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: "approved"},
		Status: kargoapi.FreightStatus{
			ApprovedFor: map[string]kargoapi.ApprovedStage{listFreightTestStage: {}},
		},
	}
	unrelated := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: "unrelated"},
		Status: kargoapi.FreightStatus{
			VerifiedIn: map[string]kargoapi.VerifiedStage{"other-stage": {}},
		},
	}
	url := "/v1beta1/projects/" + listFreightTestProject +
		"/stages/" + listFreightTestStage + "/referencing-freight"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, url,
		[]restTestCase{
			{
				// The endpoint no longer resolves the Stage resource, so a
				// missing Stage yields an empty list rather than a 404.
				name:          "missing Stage yields an empty list",
				clientBuilder: freightWithIndexes(t).WithObjects(project),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, freightListNames(t, w))
				},
			},
			{
				name: "returns Freight verified in or approved for the Stage",
				clientBuilder: freightWithIndexes(t).
					WithObjects(project, stage, verified, approved, unrelated),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					names := freightListNames(t, w)
					require.Contains(t, names, "verified")
					require.Contains(t, names, "approved")
					require.NotContains(t, names, "unrelated")
				},
			},
		},
	)
}

func Test_server_listFreightFromWarehouse(t *testing.T) {
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: listFreightTestProject}}
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{Namespace: listFreightTestProject, Name: listFreightTestWarehouse},
	}
	f1 := freightFromWarehouse("f1", listFreightTestWarehouse)
	f2 := freightFromWarehouse("f2", listFreightTestWarehouse)
	other := freightFromWarehouse("other", "other-warehouse")
	url := "/v1beta1/projects/" + listFreightTestProject +
		"/warehouses/" + listFreightTestWarehouse + "/freight"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, url,
		[]restTestCase{
			{
				// The endpoint no longer resolves the Warehouse resource, so a
				// missing Warehouse yields an empty list rather than a 404.
				name:          "missing Warehouse yields an empty list",
				clientBuilder: freightWithIndexes(t).WithObjects(project),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, freightListNames(t, w))
				},
			},
			{
				name: "returns only Freight from the Warehouse",
				clientBuilder: freightWithIndexes(t).
					WithObjects(project, warehouse, f1, f2, other),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					names := freightListNames(t, w)
					require.Contains(t, names, "f1")
					require.Contains(t, names, "f2")
					require.NotContains(t, names, "other")
				},
			},
		},
	)
}
