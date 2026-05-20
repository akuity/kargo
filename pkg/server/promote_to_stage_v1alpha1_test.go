package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
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
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
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
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
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
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
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
				authorizeFn: func(
					context.Context,
					string,
					schema.GroupVersionResource,
					string,
					client.ObjectKey,
				) error {
					return nil
				},
				isFreightAvailableFn: func(*kargoapi.Stage, *kargoapi.Freight) bool {
					return true
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
				t.Context(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, recorder, res, err)
		})
	}
}

func TestPromoteToStageCreatesAutoPromotionHold(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-project",
			Name:      "fake-stage",
		},
		Spec: kargoapi.StageSpec{
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
				},
			},
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin:  origin,
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
		Status: kargoapi.StageStatus{
			AutoPromotionEnabled: true,
		},
	}
	olderFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-project",
			Name:      "older-freight",
		},
		Origin: origin,
	}
	newerFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-project",
			Name:      "newer-freight",
		},
		Origin: origin,
	}

	internalClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(stage, olderFreight, newerFreight).
		WithStatusSubresource(stage).
		Build()
	kubeClient, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
				string,
			) (client.WithWatch, error) {
				return internalClient, nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client: kubeClient,
		validateProjectExistsFn: func(context.Context, string) error {
			return nil
		},
		getStageFn: func(
			ctx context.Context,
			c client.Client,
			key types.NamespacedName,
		) (*kargoapi.Stage, error) {
			stage := &kargoapi.Stage{}
			if getErr := c.Get(ctx, key, stage); getErr != nil {
				return nil, getErr
			}
			return stage, nil
		},
		getFreightByNameOrAliasFn: func(
			context.Context,
			client.Client,
			string,
			string,
			string,
		) (*kargoapi.Freight, error) {
			return olderFreight, nil
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
		createPromotionFn: kubeClient.Create,
		isAutoPromotionEnabledFn: func(context.Context, client.Client, metav1.ObjectMeta) (bool, error) {
			return true, nil
		},
		getAvailableFreightForStageFn: func(
			context.Context,
			*kargoapi.Stage,
		) ([]kargoapi.Freight, error) {
			return []kargoapi.Freight{*newerFreight}, nil
		},
	}

	res, err := s.PromoteToStage(
		t.Context(),
		connect.NewRequest(&svcv1alpha1.PromoteToStageRequest{
			Project: "fake-project",
			Stage:   "fake-stage",
			Freight: "older-freight",
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetPromotion())

	updatedStage := &kargoapi.Stage{}
	require.NoError(t, internalClient.Get(
		t.Context(),
		client.ObjectKey{Namespace: "fake-project", Name: "fake-stage"},
		updatedStage,
	))
	require.Len(t, updatedStage.Status.AutoPromotionHolds, 1)
	hold := updatedStage.Status.AutoPromotionHolds[origin.String()]
	require.Equal(t, kargoapi.AutoPromotionHoldStatePending, hold.State)
	require.Equal(t, olderFreight.Name, hold.Freight.Name)
	require.Equal(t, res.Msg.GetPromotion().Name, hold.PromotionName)
}

func Test_server_promoteToStage(t *testing.T) {
	now := time.Now()
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testProjectConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProject.Name,
			Namespace: testProject.Name,
		},
		Spec: kargoapi.ProjectConfigSpec{
			PromotionPolicies: []kargoapi.PromotionPolicy{{
				StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
				AutoPromotionEnabled: true,
			}},
		},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "fake-freight",
			Namespace:         testProject.Name,
			CreationTimestamp: metav1.Time{Time: now.Add(-time.Hour)},
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
	}
	testNewerFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "newer-freight",
			Namespace:         testProject.Name,
			CreationTimestamp: metav1.Time{Time: now},
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
		Status: kargoapi.StageStatus{
			AutoPromotionEnabled: true,
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
				name:          "promoting not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
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
							testStage.Name,
							errors.New("not authorized"),
						)
					}
				},
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "Freight not available to Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *kargoapi.Stage {
						s := testStage.DeepCopy()
						s.Spec.RequestedFreight[0].Sources = kargoapi.FreightSources{
							Stages: []string{"some-other-stage"},
						}
						return s
					}(),
					testFreight,
				),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Successfully promote by freight name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
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
					require.Equal(t, kargoapi.PromotionSourceNonAuto, promos.Items[0].Spec.Source)
				},
			},
			{
				name:          "Successfully promote by freight alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
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
					require.Equal(t, kargoapi.PromotionSourceNonAuto, promos.Items[0].Spec.Source)
				},
			},
			{
				name: "older freight creates pending auto-promotion hold",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testProjectConfig, testStage, testFreight, testNewerFreight).
					WithStatusSubresource(testStage),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
					Reason:                "rollback to last good version",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)

					stage := &kargoapi.Stage{}
					err = c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testStage.Name},
						stage,
					)
					require.NoError(t, err)
					require.Len(t, stage.Status.AutoPromotionHolds, 1)
					hold, ok := stage.Status.AutoPromotionHolds[testFreight.Origin.String()]
					require.True(t, ok)
					require.Equal(t, testFreight.Name, hold.Freight.Name)
					require.Equal(t, kargoapi.AutoPromotionHoldStatePending, hold.State)
					require.Equal(t, promos.Items[0].Name, hold.PromotionName)
					require.Equal(t, "rollback to last good version", hold.Reason)
				},
			},
			{
				name: "older freight is rejected while active hold exists for origin",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testProjectConfig,
					func() *kargoapi.Stage {
						stage := testStage.DeepCopy()
						stage.Status.AutoPromotionHolds = map[string]kargoapi.AutoPromotionHold{
							testFreight.Origin.String(): {
								Freight: kargoapi.FreightReference{
									Name:   testFreight.Name,
									Origin: testFreight.Origin,
								},
								State:         kargoapi.AutoPromotionHoldStateActive,
								PromotionName: "previous-rollback",
								PromotionUID:  "previous-uid",
								CreatedAt:     &metav1.Time{Time: now.Add(-30 * time.Minute)},
							},
						}
						return stage
					}(),
					testFreight,
					testNewerFreight,
				).WithStatusSubresource(testStage),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
					require.Contains(t, w.Body.String(), "auto-promotion is already active")

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Empty(t, promos.Items)

					stage := &kargoapi.Stage{}
					err = c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testStage.Name},
						stage,
					)
					require.NoError(t, err)
					require.Len(t, stage.Status.AutoPromotionHolds, 1)
					hold := stage.Status.AutoPromotionHolds[testFreight.Origin.String()]
					require.Equal(t, kargoapi.AutoPromotionHoldStateActive, hold.State)
					require.Equal(t, "previous-rollback", hold.PromotionName)
				},
			},
			{
				name: "older freight is rejected while pending hold exists for origin",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testProjectConfig,
					func() *kargoapi.Stage {
						stage := testStage.DeepCopy()
						stage.Status.AutoPromotionHolds = map[string]kargoapi.AutoPromotionHold{
							testFreight.Origin.String(): {
								Freight: kargoapi.FreightReference{
									Name:   testFreight.Name,
									Origin: testFreight.Origin,
								},
								State:         kargoapi.AutoPromotionHoldStatePending,
								PromotionName: "rollback-in-progress",
								CreatedAt:     &metav1.Time{Time: now.Add(-time.Minute)},
							},
						}
						return stage
					}(),
					testFreight,
					testNewerFreight,
				).WithStatusSubresource(testStage),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
					require.Contains(t, w.Body.String(), "auto-promotion is already pending")

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Empty(t, promos.Items)

					stage := &kargoapi.Stage{}
					err = c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testStage.Name},
						stage,
					)
					require.NoError(t, err)
					require.Len(t, stage.Status.AutoPromotionHolds, 1)
					hold := stage.Status.AutoPromotionHolds[testFreight.Origin.String()]
					require.Equal(t, kargoapi.AutoPromotionHoldStatePending, hold.State)
					require.Equal(t, "rollback-in-progress", hold.PromotionName)
				},
			},
			{
				name: "stage refresh failure after rollback promotion creation still returns created",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testProjectConfig, testStage, testFreight, testNewerFreight).
					WithStatusSubresource(testStage).
					WithInterceptorFuncs(interceptor.Funcs{
						Patch: func(
							ctx context.Context,
							c client.WithWatch,
							obj client.Object,
							patch client.Patch,
							opts ...client.PatchOption,
						) error {
							if _, ok := obj.(*kargoapi.Stage); ok {
								return errors.New("refresh failed")
							}
							return c.Patch(ctx, obj, patch, opts...)
						},
					}),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)

					stage := &kargoapi.Stage{}
					err = c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: testStage.Name},
						stage,
					)
					require.NoError(t, err)
					require.Len(t, stage.Status.AutoPromotionHolds, 1)
					require.Equal(
						t,
						promos.Items[0].Name,
						stage.Status.AutoPromotionHolds[testFreight.Origin.String()].PromotionName,
					)
				},
			},
			{
				name: "promotion creation failure reports pending hold cleanup failure",
				clientBuilder: func() *fake.ClientBuilder {
					statusPatchCount := 0
					return fake.NewClientBuilder().
						WithObjects(testProject, testProjectConfig, testStage, testFreight, testNewerFreight).
						WithStatusSubresource(testStage).
						WithInterceptorFuncs(interceptor.Funcs{
							Create: func(
								ctx context.Context,
								c client.WithWatch,
								obj client.Object,
								opts ...client.CreateOption,
							) error {
								if _, ok := obj.(*kargoapi.Promotion); ok {
									return errors.New("promotion create failed")
								}
								return c.Create(ctx, obj, opts...)
							},
							SubResourcePatch: func(
								ctx context.Context,
								c client.Client,
								subResourceName string,
								obj client.Object,
								patch client.Patch,
								opts ...client.SubResourcePatchOption,
							) error {
								if _, ok := obj.(*kargoapi.Stage); ok && subResourceName == "status" {
									statusPatchCount++
									if statusPatchCount > 1 {
										return errors.New("cleanup failed")
									}
								}
								return c.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
							},
						})
				}(),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusInternalServerError, w.Code)
					require.Contains(t, w.Body.String(), "clean up pending auto-promotion hold")
					require.Contains(t, w.Body.String(), "promotion create failed")
					require.Contains(t, w.Body.String(), "cleanup failed")
				},
			},
			{
				name: "stale expected auto-promotion candidate is rejected",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testProjectConfig, testStage, testFreight, testNewerFreight),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testFreight.Name,
					ExpectedAutoCandidate: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Empty(t, promos.Items)
				},
			},
			{
				name: "promotion without auto candidate does not mark active hold for clearing",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *kargoapi.Stage {
						stage := testStage.DeepCopy()
						stage.Status.AutoPromotionEnabled = false
						stage.Status.AutoPromotionHolds = map[string]kargoapi.AutoPromotionHold{
							testFreight.Origin.String(): {
								Freight: kargoapi.FreightReference{
									Name:   testFreight.Name,
									Origin: testFreight.Origin,
								},
								State:         kargoapi.AutoPromotionHoldStateActive,
								PromotionName: "rollback-promotion",
								PromotionUID:  "rollback-uid",
								CreatedAt:     &metav1.Time{Time: now.Add(-30 * time.Minute)},
							},
						}
						return stage
					}(),
					testFreight,
				),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.NotContains(
						t,
						promos.Items[0].Annotations,
						kargoapi.AnnotationKeyClearAutoPromotionHold,
					)
				},
			},
			{
				name: "newest freight promotion marks active hold for clearing on success",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testProjectConfig,
					func() *kargoapi.Stage {
						stage := testStage.DeepCopy()
						stage.Status.AutoPromotionHolds = map[string]kargoapi.AutoPromotionHold{
							testFreight.Origin.String(): {
								Freight: kargoapi.FreightReference{
									Name:   testFreight.Name,
									Origin: testFreight.Origin,
								},
								State:         kargoapi.AutoPromotionHoldStateActive,
								PromotionName: "rollback-promotion",
								PromotionUID:  "rollback-uid",
								CreatedAt:     &metav1.Time{Time: now.Add(-30 * time.Minute)},
							},
						}
						return stage
					}(),
					testFreight,
					testNewerFreight,
				),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testNewerFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(
						t,
						testFreight.Origin.String(),
						promos.Items[0].Annotations[kargoapi.AnnotationKeyClearAutoPromotionHold],
					)
					require.Equal(
						t,
						"rollback-promotion",
						promos.Items[0].Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldPromotion],
					)
					require.Equal(
						t,
						"rollback-uid",
						promos.Items[0].Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldPromotionUID],
					)
					require.NotEmpty(
						t,
						promos.Items[0].Annotations[kargoapi.AnnotationKeyClearAutoPromotionHoldCreatedAt],
					)
				},
			},
			{
				name: "newest freight promotion rejects stale active hold",
				clientBuilder: func() *fake.ClientBuilder {
					stageGetCount := 0
					return fake.NewClientBuilder().WithObjects(
						testProject,
						testProjectConfig,
						func() *kargoapi.Stage {
							stage := testStage.DeepCopy()
							stage.Status.AutoPromotionHolds = map[string]kargoapi.AutoPromotionHold{
								testFreight.Origin.String(): {
									Freight: kargoapi.FreightReference{
										Name:   testFreight.Name,
										Origin: testFreight.Origin,
									},
									State:         kargoapi.AutoPromotionHoldStateActive,
									PromotionName: "rollback-promotion",
									PromotionUID:  "rollback-uid",
									CreatedAt:     &metav1.Time{Time: now.Add(-30 * time.Minute)},
								},
							}
							return stage
						}(),
						testFreight,
						testNewerFreight,
					).WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							ctx context.Context,
							c client.WithWatch,
							key client.ObjectKey,
							obj client.Object,
							opts ...client.GetOption,
						) error {
							if stage, ok := obj.(*kargoapi.Stage); ok && key.Name == testStage.Name {
								stageGetCount++
								if err := c.Get(ctx, key, obj, opts...); err != nil {
									return err
								}
								if stageGetCount > 1 {
									hold := stage.Status.AutoPromotionHolds[testFreight.Origin.String()]
									hold.PromotionUID = "newer-rollback-uid"
									stage.Status.AutoPromotionHolds[testFreight.Origin.String()] = hold
								}
								return nil
							}
							return c.Get(ctx, key, obj, opts...)
						},
					})
				}(),
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
				body: mustJSONBody(promoteToStageRequest{
					Freight:               testNewerFreight.Name,
					ExpectedAutoCandidate: testNewerFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
					require.Contains(t, w.Body.String(), "auto-promotion hold")

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Empty(t, promos.Items)
				},
			},
		},
	)
}
