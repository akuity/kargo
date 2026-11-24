package server

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	libhttp "github.com/akuity/kargo/pkg/http"
)

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
				var httpErr *libhttp.HTTPError
				require.True(t, errors.As(err, &httpErr))
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
				require.Equal(
					t,
					"cannot filter by group without group by",
					httpErr.Error(),
				)
			},
		},
		{
			name:    "invalid group by",
			groupBy: "bogus-group-by",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var httpErr *libhttp.HTTPError
				require.True(t, errors.As(err, &httpErr))
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
				require.Contains(t, httpErr.Error(), "invalid group by")
			},
		},
		{
			name:    "invalid ordering by tag",
			groupBy: GroupByGitRepository,
			orderBy: OrderByTag,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var httpErr *libhttp.HTTPError
				require.True(t, errors.As(err, &httpErr))
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
				require.Contains(
					t,
					httpErr.Error(),
					"tag ordering only valid when grouping by",
				)
			},
		},
		{
			name:    "invalid order by",
			orderBy: "bogus-order-by",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var httpErr *libhttp.HTTPError
				require.True(t, errors.As(err, &httpErr))
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
				require.Contains(t, httpErr.Error(), "invalid order by")
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
