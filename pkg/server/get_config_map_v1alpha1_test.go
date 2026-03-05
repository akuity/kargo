package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestGetConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req         *svcv1alpha1.GetConfigMapRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetConfigMapResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Project: "",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing ConfigMap": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "kargo-x",
				Project: "kargo-y",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ConfigMap": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cm-1",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"foo": "bar",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetRaw())

				require.NotNil(t, r.Msg.GetConfigMap())

				require.Equal(t, "kargo-demo", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "cm-1", r.Msg.GetConfigMap().Name)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetConfigMap())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()

				require.NoError(t, corev1.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)

				require.NoError(t, err)

				tObj, ok := obj.(*corev1.ConfigMap)
				require.True(t, ok)
				require.Equal(t, "cm-1", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "bar", tObj.Data["foo"])
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Project: "kargo-demo",
				Name:    "cm-1",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetConfigMap())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*corev1.ConfigMap)
				require.True(t, ok)
				require.Equal(t, "cm-1", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "bar", tObj.Data["foo"])
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						s *runtime.Scheme,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(testCase.interceptor)
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
				externalValidateProjectFn: validation.ValidateProject,
			}

			res, err := (svr).GetConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_getProjectConfigMap(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/configmaps/"+testConfig.Name,
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
				name: "gets ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfig,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMap in the response
					config := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), config)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, config.Namespace)
					require.Equal(t, testConfig.Name, config.Name)
				},
			},
		},
	)
}

func Test_server_getSystemConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSystemResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodGet, "/v1beta1/system/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name:         "ConfigMap does not exist",
				serverConfig: &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMap in the response
					cm := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), cm)
					require.NoError(t, err)
					require.Equal(t, testSystemResourcesNamespace, cm.Namespace)
					require.Equal(t, testConfigMap.Name, cm.Name)
				},
			},
		},
	)
}

func Test_server_getSharedConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "ConfigMap does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMap in the response
					cm := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), cm)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, cm.Namespace)
					require.Equal(t, testConfigMap.Name, cm.Name)
				},
			},
		},
	)
}
