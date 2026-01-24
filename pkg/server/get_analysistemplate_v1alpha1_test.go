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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestGetAnalysisTemplate(t *testing.T) {
	testCases := map[string]struct {
		req              *svcv1alpha1.GetAnalysisTemplateRequest
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "",
				Name:    "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing AnalysisTemplate": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetAnalysisTemplate())
				require.Equal(t, "kargo-demo", c.Msg.GetAnalysisTemplate().Namespace)
				require.Equal(t, "test", c.Msg.GetAnalysisTemplate().Name)
			},
		},
		"non-existing AnalysisTemplate": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "non-existing-project",
				Name:    "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting AnalysisTemplate": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			interceptor: interceptor.Funcs{
				// This interceptor will be called when the client.Get method is called.
				// It will return an error to simulate a failure in the client.Get method.
				Get: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectKey,
					_ client.Object,
					_ ...client.GetOption,
				) error {
					return apierrors.NewServiceUnavailable("test")
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"Argo Rollouts integration is not enabled": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetAnalysisTemplate())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.AnalysisTemplate)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetAnalysisTemplateRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetAnalysisTemplate())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, rollouts.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*rollouts.AnalysisTemplate)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			cfg := config.ServerConfigFromEnv()
			if testCase.rolloutsDisabled {
				cfg.RolloutsIntegrationEnabled = false
			}

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
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[rollouts.AnalysisTemplate]("testdata/analysistemplate.yaml"),
							).
							WithInterceptorFuncs(testCase.interceptor).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				cfg:                       cfg,
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := (svr).GetAnalysisTemplate(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_getAnalysisTemplate(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testTemplate := &rollouts.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-template",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/analysis-templates/"+testTemplate.Name,
		[]restTestCase{
			{
				name:          "Rollouts integration disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:         "Project does not exist",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: true},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "AnalysisTemplate does not exist",
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: true},
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:         "gets AnalysisTemplate",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: true},
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTemplate,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the AnalysisTemplate in the response
					template := &rollouts.AnalysisTemplate{}
					err := json.Unmarshal(w.Body.Bytes(), template)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, template.Namespace)
					require.Equal(t, testTemplate.Name, template.Name)
				},
			},
		},
	)
}
