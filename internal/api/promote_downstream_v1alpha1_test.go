package api

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
	kargoEvent "github.com/akuity/kargo/internal/event"
	fakeevent "github.com/akuity/kargo/internal/event/kubernetes/fake"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestPromoteDownstream(t *testing.T) {
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
	}
	testCases := []struct {
		name       string
		req        *svcv1alpha1.PromoteDownstreamRequest
		server     *server
		assertions func(
			*testing.T,
			*fakeevent.EventRecorder,
			*connect.Response[svcv1alpha1.PromoteDownstreamResponse],
			error,
		)
	}{
		{
			name:   "input validation error",
			server: &server{},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
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
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Stage",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get stage: something went wrong", err.Error())
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
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
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get freight: something went wrong", err.Error())
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
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
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "not available to Stage")
			},
		},
		{
			name: "error finding downstream Stages",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "find downstream stages: something went wrong", err.Error())
			},
		},
		{
			name: "no downstream Stages found",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "stage")
				require.Contains(t, connErr.Message(), "has no downstream stages")
			},
		},
		{
			name: "promoting not authorized",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{{}}, nil
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "not authorized", err.Error())
			},
		},
		{
			name: "error creating Promotion",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{
						{
							Spec: kargoapi.StageSpec{
								PromotionTemplate: &kargoapi.PromotionTemplate{
									Spec: kargoapi.PromotionTemplateSpec{
										Steps: []kargoapi.PromotionStep{{}},
									},
								},
							},
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
				_ *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Contains(t, connErr.Message(), "something went wrong")
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.PromoteDownstreamRequest{
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
						Status: kargoapi.FreightStatus{
							VerifiedIn: map[string]kargoapi.VerifiedStage{
								"fake-stage": {},
							},
						},
					}, nil
				},
				findDownstreamStagesFn: func(
					context.Context,
					*kargoapi.Stage,
					kargoapi.FreightOrigin,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{
						{
							Spec: kargoapi.StageSpec{
								PromotionTemplate: &kargoapi.PromotionTemplate{
									Spec: kargoapi.PromotionTemplateSpec{
										Steps: []kargoapi.PromotionStep{{}},
									},
								},
							},
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
				res *connect.Response[svcv1alpha1.PromoteDownstreamResponse],
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.Msg.GetPromotions())
				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, corev1.EventTypeNormal, event.EventType)
				require.Equal(t, kargoEvent.EventReasonPromotionCreated, event.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := fakeevent.NewEventRecorder(1)
			testCase.server.recorder = recorder
			resp, err := testCase.server.PromoteDownstream(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, recorder, resp, err)
		})
	}
}
