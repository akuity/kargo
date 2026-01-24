package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestPromoteToStage(t *testing.T) {
	testStageSpec := kargoapi.StageSpec{
		RequestedFreight: []kargoapi.FreightRequest{{
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "fake-warehouse",
			},
			Sources: kargoapi.FreightSources{
				Stages: []string{"fake-upstream-stage"},
			},
		}},
		PromotionTemplate: &kargoapi.PromotionTemplate{
			Spec: kargoapi.PromotionTemplateSpec{
				Steps: []kargoapi.PromotionStep{{}},
			},
		},
	}
	testCases := []struct {
		name       string
		req        *svcv1alpha1.PromoteToStageRequest
		server     *server
		assertions func(
			*testing.T,
			*fakeevent.EventRecorder,
			*connect.Response[svcv1alpha1.PromoteToStageResponse],
			error,
		)
	}{
		{
			name:   "input validation error",
			req:    &svcv1alpha1.PromoteToStageRequest{},
			server: &server{},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting Stage",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.ErrorContains(t, err, "get stage: something went wrong")
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "Stage")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.ErrorContains(t, err, "get freight: something went wrong")
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "freight")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "Freight not available",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return false
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "is not available to Stage")
			},
		},
		{
			name: "promoting not authorized",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return errors.New("not authorized")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err, "not authorized")
			},
		},
		{
			name: "error building Promotion",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.ErrorContains(t, err, "build promotion")
			},
		},
		{
			name: "error creating Promotion",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-freight",
						},
					}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.Error(t, err, "create promotion: something went wrong")
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.PromoteToStageRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-stage",
						},
						Spec: testStageSpec,
					}, nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-project",
							Name:      "fake-freight",
						},
					}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
				},
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				res *connect.Response[svcv1alpha1.PromoteToStageResponse],
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotNil(t, res.Msg.GetPromotion())
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, corev1.EventTypeNormal, event.EventType)
				require.Equal(t, string(kargoapi.EventTypePromotionCreated), event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.server.sender = k8sevent.NewEventSender(recorder)
			res, err := testCase.server.PromoteToStage(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, recorder, res, err)
		})
	}
}

func Test_server_promoteToStage(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse.Name,
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{
						{Uses: "fake-step"},
					},
				},
			},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/promotions",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Freight: "nonexistent-freight",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					FreightAlias: "nonexistent-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Neither freight nor freightAlias provided",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body:          mustJSONBody(promoteToStageRequest{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Both freight and freightAlias provided",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Freight:      testFreight.Name,
					FreightAlias: "fake-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Successfully promote by freight name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Verify a Promotion was created
					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
				},
			},
			{
				name:          "Successfully promote by freight alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				body: mustJSONBody(promoteToStageRequest{
					FreightAlias: "fake-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Verify a Promotion was created
					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
				},
			},
		},
	)
}
