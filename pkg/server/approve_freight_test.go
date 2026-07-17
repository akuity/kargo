package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_approveFreight(t *testing.T) {
	const testStageName = "fake-stage"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
		},
		Origin: testOrigin,
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin}},
		},
	}
	// Same name as testStage, but doesn't request Freight from testOrigin.
	testStageWithoutRequest := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/approve?stage="+testStageName,
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				).WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "not authorized to approve (not authorized to promote)",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStage).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("stages").GroupResource(),
							testStageName,
							errors.New("not authorized"),
						)
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "Stage does not request Freight from origin",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStageWithoutRequest).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "does not request Freight from origin")

					// Verify the Freight was NOT approved for the Stage
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testFreight),
						freight,
					)
					require.NoError(t, err)
					require.False(t, freight.IsApprovedFor(testStageName))
				},
			},
			{
				name: "approves Freight",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight, testStage).
					WithStatusSubresource(testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Freight was approved for the Stage
					freight := &kargoapi.Freight{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testFreight),
						freight,
					)
					require.NoError(t, err)
					require.True(t, freight.IsApprovedFor(testStage.Name))
					require.Contains(t, freight.Status.ApprovedFor, testStage.Name)
				},
			},
		},
	)
}
