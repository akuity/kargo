package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// freightListNames decodes a FreightList response body and returns the names
// of the Freight it contains, in response order.
func freightListNames(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	list := &kargoapi.FreightList{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
	names := make([]string, 0, len(list.Items))
	for _, f := range list.Items {
		names = append(names, f.Name)
	}
	return names
}

func Test_server_listFreightFromOrigin(t *testing.T) {
	const (
		downstreamStage = "downstream"
		upstreamA       = "up-a"
		upstreamB       = "up-b"
	)
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: listFreightTestProject}}

	plain := freightFromWarehouse("plain", listFreightTestWarehouse)

	approved := freightFromWarehouse("approved", listFreightTestWarehouse)
	approved.Status.ApprovedFor = map[string]kargoapi.ApprovedStage{downstreamStage: {}}

	verifiedA := freightFromWarehouse("verified-a", listFreightTestWarehouse)
	verifiedA.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{upstreamA: {}}

	verifiedAB := freightFromWarehouse("verified-ab", listFreightTestWarehouse)
	verifiedAB.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{
		upstreamA: {},
		upstreamB: {},
	}

	// Verified in and currently in upstream A, but only just arrived, so it has
	// not soaked for any meaningful length of time.
	now := metav1.Now()
	currentA := freightFromWarehouse("current-a", listFreightTestWarehouse)
	currentA.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{upstreamA: {}}
	currentA.Status.CurrentlyIn = map[string]kargoapi.CurrentStage{upstreamA: {Since: &now}}

	soakedA := freightFromWarehouse("soaked-a", listFreightTestWarehouse)
	soakedA.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{
		upstreamA: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
	}

	// Matches every filter, but originates from another Warehouse.
	elsewhere := freightFromWarehouse("elsewhere", "other-warehouse")
	elsewhere.Status.ApprovedFor = map[string]kargoapi.ApprovedStage{downstreamStage: {}}
	elsewhere.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{upstreamA: {}}

	allFreight := func() *fake.ClientBuilder {
		return freightWithIndexes(t).WithObjects(
			project, plain, approved, verifiedA, verifiedAB, currentA, soakedA, elsewhere,
		)
	}

	baseURL := "/v1beta1/projects/" + listFreightTestProject +
		"/internal/origin/" + string(kargoapi.FreightOriginKindWarehouse) +
		"/" + listFreightTestWarehouse + "/freight"

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, baseURL,
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: freightWithIndexes(t),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "unknown origin kind",
				url: "/v1beta1/projects/" + listFreightTestProject +
					"/internal/origin/Bogus/" + listFreightTestWarehouse + "/freight",
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "invalid requiredSoakTime",
				url:           baseURL + "?verifiedIn=" + upstreamA + "&requiredSoakTime=nope",
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "invalid requiredSoakTime")
				},
			},
			{
				name:          "unsupported availabilityStrategy",
				url:           baseURL + "?verifiedIn=" + upstreamA + "&availabilityStrategy=Bogus",
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "unsupported AvailabilityStrategy")
				},
			},
			{
				// The endpoint does not resolve the Warehouse resource, so a
				// missing Warehouse yields an empty list rather than a 404.
				name: "missing Warehouse yields an empty list",
				url: "/v1beta1/projects/" + listFreightTestProject +
					"/internal/origin/" + string(kargoapi.FreightOriginKindWarehouse) +
					"/missing-warehouse/freight",
				clientBuilder: freightWithIndexes(t).WithObjects(project),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Empty(t, freightListNames(t, w))
				},
			},
			{
				name:          "no filters returns all Freight from the origin",
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t,
						[]string{"approved", "current-a", "plain", "soaked-a", "verified-a", "verified-ab"},
						freightListNames(t, w),
					)
				},
			},
			{
				name:          "approvedFor",
				url:           baseURL + "?approvedFor=" + downstreamStage,
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, []string{"approved"}, freightListNames(t, w))
				},
			},
			{
				name:          "approvedFor is OR'ed with verifiedIn",
				url:           baseURL + "?approvedFor=" + downstreamStage + "&verifiedIn=" + upstreamB,
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, []string{"approved", "verified-ab"}, freightListNames(t, w))
				},
			},
			{
				name:          "repeated verifiedIn defaults to OneOf",
				url:           baseURL + "?verifiedIn=" + upstreamA + "&verifiedIn=" + upstreamB,
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t,
						[]string{"current-a", "soaked-a", "verified-a", "verified-ab"},
						freightListNames(t, w),
					)
				},
			},
			{
				name: "repeated verifiedIn with the All availabilityStrategy",
				url: baseURL + "?verifiedIn=" + upstreamA + "&verifiedIn=" + upstreamB +
					"&availabilityStrategy=" + string(kargoapi.FreightAvailabilityStrategyAll),
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, []string{"verified-ab"}, freightListNames(t, w))
				},
			},
			{
				name:          "currentlyIn",
				url:           baseURL + "?verifiedIn=" + upstreamA + "&currentlyIn=" + upstreamA,
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, []string{"current-a"}, freightListNames(t, w))
				},
			},
			{
				name:          "requiredSoakTime",
				url:           baseURL + "?verifiedIn=" + upstreamA + "&requiredSoakTime=1h",
				clientBuilder: allFreight(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, []string{"soaked-a"}, freightListNames(t, w))
				},
			},
		},
	)
}
