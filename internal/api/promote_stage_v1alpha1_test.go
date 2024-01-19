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

func TestPromoteStage(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.PromoteStageRequest
		server     *server
		assertions func(*connect.Response[svcv1alpha1.PromoteStageResponse], error)
	}{
		{
			name:   "input validation error",
			req:    &svcv1alpha1.PromoteStageRequest{},
			server: &server{},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
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
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
				Freight: "fake-freight",
			},
			server: &server{
				validateProjectFn: func(ctx context.Context, project string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Stage",
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get stage: something went wrong", err.Error())
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
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
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get freight: something went wrong", err.Error())
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
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
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Freight")
				require.Contains(t, connErr.Message(), "is not available to Stage")
			},
		},
		{
			name: "error creating Promotion",
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "create promotion: something went wrong", err.Error())
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.PromoteStageRequest{
				Project: "fake-project",
				Name:    "fake-stage",
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
				createPromotionFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(
				res *connect.Response[svcv1alpha1.PromoteStageResponse],
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotNil(t, res.Msg.GetPromotion())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.PromoteStage(
					context.Background(),
					connect.NewRequest(testCase.req),
				),
			)
		})
	}
}
