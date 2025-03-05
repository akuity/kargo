package server

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/server/validation"
)

func TestValidateFieldNotEmpty(t *testing.T) {
	testCases := []struct {
		name       string
		fieldName  string
		fieldValue string
		assertions func(*testing.T, error)
	}{
		{
			name:       "field is empty",
			fieldName:  "project",
			fieldValue: "",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(t, "project should not be empty", connErr.Message())
			},
		},
		{
			name:       "field is not empty",
			fieldName:  "project",
			fieldValue: "fake-project",
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				validateFieldNotEmpty(testCase.fieldName, testCase.fieldValue),
			)
		})
	}
}

func TestValidateProjectExists(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, error)
	}{
		{
			name: "project not found",
			server: &server{
				externalValidateProjectFn: func(
					context.Context,
					client.Client,
					string,
				) error {
					return validation.ErrProjectNotFound
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeNotFound, connErr.Code())
			},
		},
		{
			name: "field error",
			server: &server{
				externalValidateProjectFn: func(
					context.Context,
					client.Client,
					string,
				) error {
					return &field.Error{}
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
			},
		},
		{
			name: "other error",
			server: &server{
				externalValidateProjectFn: func(
					context.Context,
					client.Client,
					string,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "project is valid",
			server: &server{
				externalValidateProjectFn: func(
					context.Context,
					client.Client,
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
			testCase.assertions(
				t,
				testCase.server.validateProjectExists(context.Background(), "fake-project"),
			)
		})
	}
}

func TestValidateGroupByOrderBy(t *testing.T) {
	testCases := []struct {
		name       string
		group      string
		groupBy    string
		orderBy    string
		assertions func(*testing.T, error)
	}{
		{
			name:    "group is not empty but group by is empty",
			group:   "fake-group",
			groupBy: "",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(
					t,
					"Cannot filter by group without group by",
					connErr.Message(),
				)
			},
		},
		{
			name:    "invalid group by",
			groupBy: "bogus-group-by",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Invalid group by")
			},
		},
		{
			name:    "invalid ordering by tag",
			groupBy: GroupByGitRepository,
			orderBy: OrderByTag,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(
					t,
					connErr.Message(),
					"Tag ordering only valid when grouping by",
				)
			},
		},
		{
			name:    "invalid order by",
			orderBy: "bogus-order-by",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Invalid order by")
			},
		},
		{
			name:    "valid group by and order by",
			groupBy: GroupByGitRepository,
			orderBy: OrderByFirstSeen,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				validateGroupByOrderBy(
					testCase.group,
					testCase.groupBy,
					testCase.orderBy,
				),
			)
		})
	}
}
