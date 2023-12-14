package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestPromoteSubscribers(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.PromoteSubscribersRequest
		server     *server
		assertions func(
			*connect.Response[svcv1alpha1.PromoteSubscribersResponse],
			error,
		)
	}{
		{
			name:   "input validation error",
			server: &server{},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Stage",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
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
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
				require.Equal(t, "get stage: something went wrong", connErr.Message())
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
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
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "Stage")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
				require.Equal(t, "get freight: something went wrong", connErr.Message())
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "Freight not available",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Freight, string, []string) bool {
					return false
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "not available to Stage")
			},
		},
		{
			name: "error finding Stage subscribers",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Freight, string, []string) bool {
					return true
				},
				findStageSubscribersFn: func(
					context.Context,
					*kargoapi.Stage,
				) ([]kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
				require.Equal(t, "find stage subscribers: something went wrong", connErr.Message())
			},
		},
		{
			name: "no Stage subscribers found",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Freight, string, []string) bool {
					return true
				},
				findStageSubscribersFn: func(
					context.Context,
					*kargoapi.Stage,
				) ([]kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "stage")
				require.Contains(t, connErr.Message(), "has no subscribers")
			},
		},
		{
			name: "error creating Promotion",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Freight, string, []string) bool {
					return true
				},
				findStageSubscribersFn: func(
					context.Context,
					*kargoapi.Stage,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{{}}, nil
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
				_ *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Contains(t, connErr.Message(), "something went wrong")
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.PromoteSubscribersRequest{
				Project: "fake-project",
				Stage:   "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{
								UpstreamStages: []kargoapi.StageSubscription{
									{
										Name: "fake-upstream-stage",
									},
								},
							},
						},
					}, nil
				},
				getFreightFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				isFreightAvailableFn: func(*kargoapi.Freight, string, []string) bool {
					return true
				},
				findStageSubscribersFn: func(
					context.Context,
					*kargoapi.Stage,
				) ([]kargoapi.Stage, error) {
					return []kargoapi.Stage{{}}, nil
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
				res *connect.Response[svcv1alpha1.PromoteSubscribersResponse],
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.Msg.GetPromotions())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resp, err := testCase.server.PromoteSubscribers(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(resp, err)
		})
	}
}
