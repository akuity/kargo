package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

func TestListConfigMaps(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.ListConfigMapsRequest
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListConfigMapsResponse], error)
	}{
		"empty project (shared namespace)": {
			req: &svcv1alpha1.ListConfigMapsRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListConfigMapsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				// Should return config maps from shared namespace (none in this test)
				require.Empty(t, r.Msg.GetConfigMaps())
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListConfigMapsRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListConfigMapsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"config maps": {
			req: &svcv1alpha1.ListConfigMapsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-2.yaml"),
				// in different namespace
				mustNewObject[corev1.ConfigMap]("testdata/config-map-3.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListConfigMapsResponse], err error) {
				require.NoError(t, err)

				cms := r.Msg.GetConfigMaps()

				fmt.Println(cms)

				require.Equal(t, 2, len(cms))
				require.Equal(t, "cm-1", cms[0].Name)
				require.Equal(t, "bar", cms[0].Data["foo"])
				require.Equal(t, "cm-2", cms[1].Name)
				require.Equal(t, "baz", cms[1].Data["bar"])
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			cfg := config.ServerConfigFromEnv()

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
				client:                    client,
				cfg:                       cfg,
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := (svr).ListConfigMaps(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_listProjectConfigMaps(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/configmaps",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no ConfigMaps exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSystemConfigMaps(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodGet, "/v1beta1/system/configmaps",
		[]restTestCase{
			{
				name: "no ConfigMaps exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSharedConfigMaps(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/configmaps",
		[]restTestCase{
			{
				name: "no ConfigMaps exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}
