package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestQueryFreight(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.QueryFreightRequest
		server     *server
		assertions func(*testing.T, *connect.Response[svcv1alpha1.QueryFreightResponse], error)
	}{
		{
			name: "empty project",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "",
			},
			server: &server{},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(t, "project should not be empty", connErr.Message())
			},
		},
		{
			name: "error validating project",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error validating group by and order by",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: "bogus-group-by",
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				var connErr *connect.Error
				require.True(t, errors.As(err, &connErr))
				require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
				require.Equal(t, "Invalid group by: bogus-group-by", connErr.Message())
			},
		},

		{
			name: "error getting Stage",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
				Stage:   "fake-stage",
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
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
			},
		},

		{
			name: "Stage not found",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
				Stage:   "fake-stage",
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
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
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
			name: "error getting available Freight for Stage",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
				Stage:   "fake-stage",
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
					return &kargoapi.Stage{}, nil
				},
				getAvailableFreightForStageFn: func(
					context.Context,
					string,
					string,
					[]kargoapi.FreightRequest,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
			},
		},

		{
			name: "error listing all Freight",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},

		{
			name: "group by image repo",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "kargo-demo",
				GroupBy: GroupByImageRepository,
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{Images: []kargoapi.Image{{RepoURL: "fake-repo-url"}}},
						{Images: []kargoapi.Image{{RepoURL: "fake-repo-url"}}},
						{Images: []kargoapi.Image{{RepoURL: "another-fake-repo-url"}}},
						{Images: []kargoapi.Image{{RepoURL: "another-fake-repo-url"}}},
					}
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				res *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, res.Msg.GetGroups(), 2)
				require.Len(t, res.Msg.GetGroups()["fake-repo-url"].Freight, 2)
				require.Len(t, res.Msg.GetGroups()["another-fake-repo-url"].Freight, 2)
			},
		},

		{
			name: "group by git repo",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "kargo-demo",
				GroupBy: GroupByGitRepository,
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{
						{Commits: []kargoapi.GitCommit{{RepoURL: "fake-repo-url"}}},
						{Commits: []kargoapi.GitCommit{{RepoURL: "fake-repo-url"}}},
						{Commits: []kargoapi.GitCommit{{RepoURL: "another-fake-repo-url"}}},
						{Commits: []kargoapi.GitCommit{{RepoURL: "another-fake-repo-url"}}},
					}
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				res *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, res.Msg.GetGroups(), 2)
				require.Len(t, res.Msg.GetGroups()["fake-repo-url"].Freight, 2)
				require.Len(t, res.Msg.GetGroups()["another-fake-repo-url"].Freight, 2)
			},
		},

		{
			name: "group by chart repo",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "kargo-demo",
				GroupBy: GroupByChartRepository,
			},
			server: &server{
				validateProjectExistsFn: func(context.Context, string) error {
					return nil
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
							Charts: []kargoapi.Chart{{
								RepoURL: "fake-repo-url",
								Name:    "fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RepoURL: "fake-repo-url",
								Name:    "fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RepoURL: "fake-repo-url",
								Name:    "another-fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RepoURL: "fake-repo-url",
								Name:    "another-fake-chart-name",
							}},
						},
					}
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				res *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, res.Msg.GetGroups(), 2)
				require.Len(
					t,
					res.Msg.GetGroups()["fake-repo-url/fake-chart-name"].Freight,
					2,
				)
				require.Len(
					t,
					res.Msg.GetGroups()["fake-repo-url/another-fake-chart-name"].Freight,
					2,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.server.QueryFreight(
				context.Background(),
				connect.NewRequest(testCase.req),
			)
			testCase.assertions(t, res, err)
		})
	}
}

func TestGetAvailableFreightForStage(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name       string
		reqs       []kargoapi.FreightRequest
		server     *server
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error getting Freight from Warehouse",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
			server: &server{
				getFreightFromWarehousesFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "get freight from warehouses")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting Freight verified in upstream Stages",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Stages: []string{"fake-stage"},
					},
				},
			},
			server: &server{
				getFreightFromWarehousesFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return nil, nil
				},
				getVerifiedFreightFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "get verified freight")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting Freight approved for Stage",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Stages: []string{"fake-stage"},
					},
				},
			},
			server: &server{
				getFreightFromWarehousesFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return nil, nil
				},
				getVerifiedFreightFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return nil, nil
				},
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "error listing Freight approved for Stage")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success getting available Freight",
			reqs: []kargoapi.FreightRequest{
				{
					Origin: testOrigin,
					Sources: kargoapi.FreightSources{
						Stages: []string{"fake-stage"},
					},
				},
			},
			server: &server{
				getFreightFromWarehousesFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight-from-warehouse",
							},
						},
					}, nil
				},
				getVerifiedFreightFn: func(context.Context, string, []string) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-verified-freight",
							},
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
								Name: "fake-approved-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 3)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.server.getAvailableFreightForStage(
				context.Background(),
				"fake-project",
				"fake-stage",
				testCase.reqs,
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			server: &server{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight for Warehouse")
			},
		},
		{
			name: "success",
			server: &server{
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
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.server.getFreightFromWarehouses(
				context.Background(),
				"fake-project",
				[]string{"fake-warehouse"},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGetVerifiedFreight(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "error listing Freight",
			server: &server{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error listing Freight verified in Stage")
			},
		},
		{
			name: "success",
			server: &server{
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
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another-fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				// Ensured the list is de-duped. If it weren't there would be 4 here.
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.server.getVerifiedFreight(
				context.Background(),
				"fake-project",
				[]string{
					"fake-stage",
					"another-fake-stage",
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}

func TestGroupByImageRepo(t *testing.T) {
	testFreight := []kargoapi.Freight{
		{Images: []kargoapi.Image{{RepoURL: "fake-repo-url"}}},
		{Images: []kargoapi.Image{{RepoURL: "fake-repo-url"}}},
		{Images: []kargoapi.Image{{RepoURL: "another-fake-repo-url"}}},
		{Images: []kargoapi.Image{{RepoURL: "another-fake-repo-url"}}},
	}
	testCases := []struct {
		name       string
		group      string
		assertions func(*testing.T, map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
				require.Len(t, groups["another-fake-repo-url"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-repo-url",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, groupByImageRepo(testFreight, testCase.group))
		})
	}
}

func TestGroupByGitRepo(t *testing.T) {
	testFreight := []kargoapi.Freight{
		{Commits: []kargoapi.GitCommit{{RepoURL: "fake-repo-url"}}},
		{Commits: []kargoapi.GitCommit{{RepoURL: "fake-repo-url"}}},
		{Commits: []kargoapi.GitCommit{{RepoURL: "another-fake-repo-url"}}},
		{Commits: []kargoapi.GitCommit{{RepoURL: "another-fake-repo-url"}}},
	}
	testCases := []struct {
		name       string
		group      string
		assertions func(*testing.T, map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
				require.Len(t, groups["another-fake-repo-url"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-repo-url",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, groupByGitRepo(testFreight, testCase.group))
		})
	}
}

func TestGroupByChart(t *testing.T) {
	testFreight := []kargoapi.Freight{
		{
			Charts: []kargoapi.Chart{{
				RepoURL: "fake-repo-url",
				Name:    "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RepoURL: "fake-repo-url",
				Name:    "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RepoURL: "another-fake-repo-url",
				Name:    "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RepoURL: "another-fake-repo-url",
				Name:    "fake-chart",
			}},
		},
	}
	testCases := []struct {
		name       string
		group      string
		assertions func(*testing.T, map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-repo-url/fake-chart"].Freight, 2)
				require.Len(t, groups["another-fake-repo-url/fake-chart"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-repo-url/fake-chart",
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-repo-url/fake-chart"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, groupByChart(testFreight, testCase.group))
		})
	}
}

func TestSortFreightGroups(t *testing.T) {
	now := time.Now().UTC()
	testCases := []struct {
		name       string
		groups     map[string]*svcv1alpha1.FreightList
		orderBy    string
		reverse    bool
		assertions func(*testing.T, map[string]*svcv1alpha1.FreightList)
	}{
		{
			name: "order by tag",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*kargoapi.Freight{
						{Images: []kargoapi.Image{{Tag: "b"}}},
						{Images: []kargoapi.Image{{Tag: "c"}}},
						{Images: []kargoapi.Image{{Tag: "a"}}},
					},
				},
			},
			orderBy: OrderByTag,
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, "a", groups[""].Freight[0].Images[0].Tag)
				require.Equal(t, "b", groups[""].Freight[1].Images[0].Tag)
				require.Equal(t, "c", groups[""].Freight[2].Images[0].Tag)
			},
		},
		{
			name: "reverse order by tag",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*kargoapi.Freight{
						{Images: []kargoapi.Image{{Tag: "b"}}},
						{Images: []kargoapi.Image{{Tag: "c"}}},
						{Images: []kargoapi.Image{{Tag: "a"}}},
					},
				},
			},
			orderBy: OrderByTag,
			reverse: true,
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, "c", groups[""].Freight[0].Images[0].Tag)
				require.Equal(t, "b", groups[""].Freight[1].Images[0].Tag)
				require.Equal(t, "a", groups[""].Freight[2].Images[0].Tag)
			},
		},
		{
			name: "order by first seen",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(time.Hour)),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
							},
						},
					},
				},
			},
			orderBy: OrderByFirstSeen,
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, now.Add(-time.Hour), groups[""].Freight[0].CreationTimestamp.Time)
				require.Equal(t, now, groups[""].Freight[1].CreationTimestamp.Time)
				require.Equal(t, now.Add(time.Hour), groups[""].Freight[2].CreationTimestamp.Time)
			},
		},
		{
			name: "reverse order by first seen",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(time.Hour)),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
							},
						},
					},
				},
			},
			orderBy: OrderByFirstSeen,
			reverse: true,
			assertions: func(t *testing.T, groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, now.Add(time.Hour), groups[""].Freight[0].CreationTimestamp.Time)
				require.Equal(t, now, groups[""].Freight[1].CreationTimestamp.Time)
				require.Equal(t, now.Add(-time.Hour), groups[""].Freight[2].CreationTimestamp.Time)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sortFreightGroups(testCase.orderBy, testCase.reverse, testCase.groups)
			testCase.assertions(t, testCase.groups)
		})
	}
}
