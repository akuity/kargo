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
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestGetStage(t *testing.T) {
	testSets := map[string]struct {
		req         *svcv1alpha1.GetStageRequest
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetStageResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "",
				Name:    "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing Stage": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetStage())
				require.Equal(t, "kargo-demo", c.Msg.GetStage().Namespace)
				require.Equal(t, "test", c.Msg.GetStage().Name)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing Stage": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "non-existing",
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting Stage": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			interceptor: interceptor.Funcs{
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
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetStage())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Stage)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetStageRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetStageResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetStage())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Stage)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
	}
	for name, ts := range testSets {
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
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[kargoapi.Stage]("testdata/stage.yaml"),
							).
							WithInterceptorFuncs(ts.interceptor).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := (svr).GetStage(ctx, connect.NewRequest(ts.req))
			ts.assertions(t, res, err)
		})
	}
}

func Test_server_getStage(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-stage",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      testStage.Name,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Stage in the response
					stage := &kargoapi.Stage{}
					err := json.Unmarshal(w.Body.Bytes(), stage)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, stage.Namespace)
					require.Equal(t, testStage.Name, stage.Name)
				},
			},
		},
	)
}

func Test_server_getStage_watch(t *testing.T) {
	const projectName = "fake-project"
	const stageName = "fake-stage"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/stages/"+stageName+"?watch=true",
		[]restWatchTestCase{
			{
				name: "stage not found",
				url:  "/v1beta1/projects/" + projectName + "/stages/non-existent?watch=true",
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
				name: "watches stage successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      stageName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Fetch the current stage first to get the resource version
					stage := &kargoapi.Stage{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: stageName}, stage)

					// Update the stage to trigger a watch event
					stage.Spec.RequestedFreight = []kargoapi.FreightRequest{{Origin: kargoapi.FreightOrigin{Kind: "Warehouse"}}}
					_ = c.Update(ctx, stage)
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
