package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/user"
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

func TestGetFreightByNameOrAlias(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	newFreight := func(name, alias string) *kargoapi.Freight {
		return &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "fake-project",
				Labels:    map[string]string{kargoapi.LabelKeyAlias: alias},
			},
		}
	}
	requireHTTPStatus := func(t *testing.T, err error, code int) {
		var httpErr *libhttp.HTTPError
		require.True(t, errors.As(err, &httpErr))
		require.Equal(t, code, httpErr.Code())
	}

	testCases := []struct {
		name        string
		objects     []client.Object
		interceptor interceptor.Funcs
		nameOrAlias string
		assertions  func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:        "error getting by name",
			nameOrAlias: "fake-freight",
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, freight)
			},
		},
		{
			name:        "found by name",
			objects:     []client.Object{newFreight("fake-freight", "fake-alias")},
			nameOrAlias: "fake-freight",
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-freight", freight.Name)
			},
		},
		{
			name:        "error listing by alias",
			nameOrAlias: "fake-alias",
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, freight)
			},
		},
		{
			name:        "not found by name or alias",
			nameOrAlias: "nonexistent",
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				requireHTTPStatus(t, err, http.StatusNotFound)
				require.Nil(t, freight)
			},
		},
		{
			name:        "found by alias",
			objects:     []client.Object{newFreight("fake-freight", "fake-alias")},
			nameOrAlias: "fake-alias",
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-freight", freight.Name)
			},
		},
		{
			name: "alias shared by multiple Freight",
			objects: []client.Object{
				newFreight("fake-freight-1", "fake-alias"),
				newFreight("fake-freight-2", "fake-alias"),
			},
			nameOrAlias: "fake-alias",
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				requireHTTPStatus(t, err, http.StatusConflict)
				require.ErrorContains(t, err, "fake-freight-1")
				require.ErrorContains(t, err, "fake-freight-2")
				require.Nil(t, freight)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			internalClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			c, err := kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
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
			s := &server{client: c}
			// An admin user bypasses the wrapper's access review, which is not
			// what is under test here.
			ctx := user.ContextWithIdentity(t.Context(), user.Admin{})
			freight, err := s.getFreightByNameOrAlias(
				ctx,
				"fake-project",
				testCase.nameOrAlias,
			)
			testCase.assertions(t, freight, err)
		})
	}
}
