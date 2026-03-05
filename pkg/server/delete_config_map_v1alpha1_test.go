package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestDeleteConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.DeleteConfigMapRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.DeleteConfigMapResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				Name: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				Name:    "test-cm",
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"delete from project namespace": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
			},
		},
		"delete from shared namespace": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				Name:    "shared-cm",
				Project: "",
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "kargo-shared-resources",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
			},
		},
		"delete system-level": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				SystemLevel: true,
				Name:        "system-cm",
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "system-cm",
						Namespace: "kargo-system-resources",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
			},
		},
		"delete non-existing ConfigMap": {
			req: &svcv1alpha1.DeleteConfigMapRequest{
				Name:    "non-existing-cm",
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.DeleteConfigMapResponse], err error) {
				require.Error(t, err)
				require.Nil(t, r)
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
				cfg: config.ServerConfig{
					SharedResourcesNamespace: "kargo-shared-resources",
					SystemResourcesNamespace: "kargo-system-resources",
				},
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := svr.DeleteConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_deleteProjectConfigMap(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "ConfigMap does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfigMap,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSystemConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSystemResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodDelete, "/v1beta1/system/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "ConfigMap does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSharedConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodDelete, "/v1beta1/shared/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "ConfigMap does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
