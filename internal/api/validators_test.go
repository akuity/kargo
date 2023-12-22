package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/validation"
)

func TestValidateProject(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(error)
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
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
			assertions: func(err error) {
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
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.validateProject(context.Background(), "fake-project"),
			)
		})
	}
}

func TestValidateProjectAndStageNonEmpty(t *testing.T) {
	testCases := []struct {
		name       string
		project    string
		stage      string
		assertions func(error)
	}{
		{
			name:    "project is empty",
			project: "",
			stage:   "fake-stage",
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(t, "project should not be empty", connErr.Message())
			},
		},
		{
			name:    "stage is empty",
			project: "fake-project",
			stage:   "",
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(t, "stage should not be empty", connErr.Message())
			},
		},
		{
			name:    "project and stage are both non-empty",
			project: "fake-project",
			stage:   "fake-stage",
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				validateProjectAndStageNonEmpty(testCase.project, testCase.stage),
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
		assertions func(error)
	}{
		{
			name:    "group is not empty but group by is empty",
			group:   "fake-group",
			groupBy: "",
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Invalid group by")
			},
		},
		{
			name:    "invalid ordering by tag",
			groupBy: GroupByGitRepository,
			orderBy: OrderByTag,
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
			assertions: func(err error) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Contains(t, connErr.Message(), "Invalid order by")
			},
		},
		{
			name:    "valid group by and order by",
			groupBy: GroupByGitRepository,
			orderBy: OrderByFirstSeen,
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				validateGroupByOrderBy(
					testCase.group,
					testCase.groupBy,
					testCase.orderBy,
				),
			)
		})
	}
}
