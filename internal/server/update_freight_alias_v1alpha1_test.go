package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestUpdateFreightAlias(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.UpdateFreightAliasRequest
		server     *server
		assertions func(*testing.T, error)
	}{
		{
			name:   "project not specified",
			req:    &svcv1alpha1.UpdateFreightAliasRequest{},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "neither name nor existing alias specified",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project: "fake-project",
			},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "new alias not specified",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project: "fake-project",
				Name:    "fake-freight",
			},
			server: &server{},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "error getting Freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "freight not found",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
				require.Contains(t, connErr.Message(), "freight")
				require.Contains(t, connErr.Message(), "not found in namespace")
			},
		},
		{
			name: "error listing freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Equal(t, "something went wrong", connErr.Message())
			},
		},
		{
			name: "alias is not unique",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fake-freight",
						},
					}, nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "different-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeAlreadyExists, connErr.Code())
				require.Contains(
					t,
					connErr.Message(),
					"already used by another piece of Freight",
				)
			},
		},
		{
			name: "error patching Freight",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					_ context.Context,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return nil
				},
				patchFreightAliasFn: func(
					context.Context,
					*kargoapi.Freight,
					string,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInternal, connErr.Code())
				require.Equal(t, "something went wrong", connErr.Message())
			},
		},
		{
			name: "success",
			req: &svcv1alpha1.UpdateFreightAliasRequest{
				Project:  "fake-project",
				Name:     "fake-freight",
				NewAlias: "fake-alias",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				getFreightByNameOrAliasFn: func(
					context.Context,
					client.Client,
					string, string, string,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(
					_ context.Context,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return nil
				},
				patchFreightAliasFn: func(
					context.Context,
					*kargoapi.Freight,
					string,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.server.UpdateFreightAlias(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, err)
		})
	}
}
