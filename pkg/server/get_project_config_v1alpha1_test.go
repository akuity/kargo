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

func TestGetProjectConfig(t *testing.T) {
	testCases := map[string]struct {
		req         *svcv1alpha1.GetProjectConfigRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetProjectConfigResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Project: "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Project: "kargo-x",
			},
			objects: []client.Object{},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ProjectConfig": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetRaw())

				require.NotNil(t, r.Msg.GetProjectConfig())
				require.Equal(t, "kargo-demo", r.Msg.GetProjectConfig().Name)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Project: "kargo-demo",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.ProjectConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProjectConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								AutoPromotionEnabled: true,
								StageSelector: &kargoapi.PromotionPolicySelector{
									Name: "foo",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetProjectConfig())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)

				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ProjectConfig)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, 1, len(tObj.Spec.PromotionPolicies))
				require.Equal(t, true, tObj.Spec.PromotionPolicies[0].AutoPromotionEnabled)
				require.Equal(t, "foo", tObj.Spec.PromotionPolicies[0].StageSelector.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Project: "kargo-demo",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.ProjectConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProjectConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								AutoPromotionEnabled: true,
								StageSelector: &kargoapi.PromotionPolicySelector{
									Name: "foo",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetProjectConfig())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ProjectConfig)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, 1, len(tObj.Spec.PromotionPolicies))
				require.Equal(t, true, tObj.Spec.PromotionPolicies[0].AutoPromotionEnabled)
				require.Equal(t, "foo", tObj.Spec.PromotionPolicies[0].StageSelector.Name)
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
						scheme *runtime.Scheme,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(testCase.interceptor)
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

			res, err := (svr).GetProjectConfig(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_getProjectConfig(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/config",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "ProjectConfig does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets ProjectConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfig,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ProjectConfig in the response
					config := &kargoapi.ProjectConfig{}
					err := json.Unmarshal(w.Body.Bytes(), config)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, config.Namespace)
					require.Equal(t, testProject.Name, config.Name)
				},
			},
		},
	)
}

func Test_server_getProjectConfig_watch(t *testing.T) {
	const projectName = "fake-project"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/config?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/config?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "project config not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches project config successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      projectName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Update the project config to trigger a watch event
					// Fetch the current config first to get the resource version
					config := &kargoapi.ProjectConfig{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: projectName}, config)

					config.Spec.PromotionPolicies = []kargoapi.PromotionPolicy{
						{AutoPromotionEnabled: true},
					}
					_ = c.Update(ctx, config)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the update operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
				},
			},
		},
	)
}
