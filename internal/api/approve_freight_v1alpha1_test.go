package api

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestApproveFreight(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.ApproveFreightRequest
		server     *server
		assertions func(*connect.Response[svcv1alpha1.ApproveFreightResponse], error)
	}{
		{
			name:   "input validation error",
			req:    &svcv1alpha1.ApproveFreightRequest{},
			server: &server{},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
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
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get freight: something went wrong", err.Error())
			},
		},
		{
			name: "Freight not found",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string,
					string,
					string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
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
			name: "error getting Stage",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
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
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "get stage: something went wrong", err.Error())
			},
		},
		{
			name: "Stage not found",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
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
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return nil, nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
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
			name: "approving (promoting) not authorized",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
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
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
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
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "not authorized", err.Error())
			},
		},
		{
			name: "error patching Freight",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
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
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
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
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "patch status: something went wrong", err.Error())
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.ApproveFreightRequest{
				Project: "fake-project",
				Name:    "fake-freight",
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectExistsFn: func(ctx context.Context, project string) error {
					return nil
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
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{}, nil
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
				patchFreightStatusFn: func(
					context.Context,
					*kargoapi.Freight,
					kargoapi.FreightStatus,
				) error {
					return nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.ApproveFreightResponse],
				err error,
			) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resp, err := testCase.server.ApproveFreight(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(resp, err)
		})
	}
}
