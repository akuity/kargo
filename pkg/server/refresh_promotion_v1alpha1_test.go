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

func Test_server_refreshPromotion(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}
	testPromotion := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-promotion",
			Namespace: testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/promotions/"+testPromotion.Name+"/refresh",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Promotion not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testPromotion),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("promotions").GroupResource(),
							testPromotion.Name,
							errors.New("not authorized"),
						)
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name:          "refreshes Promotion",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testPromotion),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = func(
						_ context.Context,
						verb string,
						gvr schema.GroupVersionResource,
						subresource string,
						key client.ObjectKey,
					) error {
						require.Equal(t, "get", verb)
						require.Equal(t, kargoapi.GroupVersion.WithResource("promotions"), gvr)
						require.Empty(t, subresource)
						require.Equal(t, client.ObjectKeyFromObject(testPromotion), key)
						return nil
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Promotion was refreshed
					promotion := &kargoapi.Promotion{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testPromotion),
						promotion,
					)
					require.NoError(t, err)
					require.NotEmpty(t, promotion.Annotations[kargoapi.AnnotationKeyRefresh])
				},
			},
		},
	)
}
