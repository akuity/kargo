package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
				require.Equal(t, kargoapi.EventReasonPromotionCreated, event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.server.recorder = recorder
			res, err := testCase.server.PromoteToStage(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, recorder, res, err)
		})
	}
}
