package api

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1meta1 "github.com/akuity/kargo/pkg/api/metav1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func TestQueryFreight(t *testing.T) {
	testCases := []struct {
		name       string
		req        *svcv1alpha1.QueryFreightRequest
		server     *server
		assertions func(*connect.Response[svcv1alpha1.QueryFreightResponse], error)
	}{
		{
			name: "empty project",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "",
			},
			server: &server{},
			assertions: func(
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
				validateProjectFn: func(context.Context, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error validating group by and order by",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: "bogus-group-by",
			},
			server: &server{
				validateProjectFn: func(context.Context, string) error {
					return nil
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
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
				validateProjectFn: func(context.Context, string) error {
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
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
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
				validateProjectFn: func(context.Context, string) error {
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
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
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
			name: "error getting available Freight for Stage",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
				Stage:   "fake-stage",
			},
			server: &server{
				validateProjectFn: func(context.Context, string) error {
					return nil
				},
				getStageFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*kargoapi.Stage, error) {
					return &kargoapi.Stage{
						Spec: &kargoapi.StageSpec{
							Subscriptions: &kargoapi.Subscriptions{},
						},
					}, nil
				},
				getAvailableFreightForStageFn: func(
					context.Context,
					string,
					string,
					kargoapi.Subscriptions,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
			},
		},

		{
			name: "error listing all Freight",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "fake-project",
				GroupBy: GroupByImageRepository,
			},
			server: &server{
				validateProjectFn: func(context.Context, string) error {
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
				_ *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.Error(t, err)
				connErr, ok := err.(*connect.Error)
				require.True(t, ok)
				require.Equal(t, connect.CodeUnknown, connErr.Code())
				require.Contains(t, connErr.Message(), "something went wrong")
			},
		},

		{
			name: "group by image repo",
			req: &svcv1alpha1.QueryFreightRequest{
				Project: "kargo-demo",
				GroupBy: GroupByImageRepository,
			},
			server: &server{
				validateProjectFn: func(context.Context, string) error {
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
				validateProjectFn: func(context.Context, string) error {
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
				validateProjectFn: func(context.Context, string) error {
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
								RegistryURL: "fake-registry-url",
								Name:        "fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RegistryURL: "fake-registry-url",
								Name:        "fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RegistryURL: "fake-registry-url",
								Name:        "another-fake-chart-name",
							}},
						},
						{
							Charts: []kargoapi.Chart{{
								RegistryURL: "fake-registry-url",
								Name:        "another-fake-chart-name",
							}},
						},
					}
					return nil
				},
			},
			assertions: func(
				res *connect.Response[svcv1alpha1.QueryFreightResponse],
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, res.Msg.GetGroups(), 2)
				require.Len(
					t,
					res.Msg.GetGroups()["fake-registry-url/fake-chart-name"].Freight,
					2,
				)
				require.Len(
					t,
					res.Msg.GetGroups()["fake-registry-url/another-fake-chart-name"].Freight,
					2,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.QueryFreight(
					context.Background(),
					connect.NewRequest(testCase.req),
				),
			)
		})
	}
}

func TestGetAvailableFreightForStage(t *testing.T) {
	testCases := []struct {
		name       string
		subs       kargoapi.Subscriptions
		server     *server
		assertions func([]kargoapi.Freight, error)
	}{
		{
			name: "error getting Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			server: &server{
				getFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
			},
		},
		{
			name: "success getting Freight from Warehouse",
			subs: kargoapi.Subscriptions{
				Warehouse: "fake-warehouse",
			},
			server: &server{
				getFreightFromWarehouseFn: func(
					context.Context,
					string,
					string,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "another fake-freight",
							},
						},
					}, nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
			},
		},
		{
			name: "error getting Freight verified in upstream Stages",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{
					{
						Name: "fake-stage",
					},
				},
			},
			server: &server{
				getVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(f []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error listing Freight verified in Stages upstream from Stage",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error getting Freight approved for Stage",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{
					{
						Name: "fake-stage",
					},
				},
			},
			server: &server{
				getVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
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
			assertions: func(f []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error listing Freight approved for Stage",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success getting available Freight",
			subs: kargoapi.Subscriptions{
				UpstreamStages: []kargoapi.StageSubscription{
					{
						Name: "fake-stage",
					},
				},
			},
			server: &server{
				getVerifiedFreightFn: func(
					context.Context,
					string,
					[]kargoapi.StageSubscription,
				) ([]kargoapi.Freight, error) {
					return []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fake-freight",
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
								Name: "another fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.getAvailableFreightForStage(
					context.Background(),
					"fake-project",
					"fake-stage",
					testCase.subs,
				),
			)
		})
	}
}

func TestGetFreightFromWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func([]kargoapi.Freight, error)
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
			assertions: func(_ []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error listing Freight for Warehouse")
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
								Name: "another fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.getFreightFromWarehouse(
					context.Background(),
					"fake-project",
					"fake-warehouse",
				),
			)
		})
	}
}

func TestGetVerifiedFreight(t *testing.T) {
	testCases := []struct {
		name       string
		server     *server
		assertions func([]kargoapi.Freight, error)
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
			assertions: func(_ []kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error listing Freight verified in Stage",
				)
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
								Name: "another fake-freight",
							},
						},
					}
					return nil
				},
			},
			assertions: func(freight []kargoapi.Freight, err error) {
				require.NoError(t, err)
				// Ensured the list is de-duped. If it weren't there would be 4 here.
				require.Len(t, freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.server.getVerifiedFreight(
					context.Background(),
					"fake-project",
					[]kargoapi.StageSubscription{
						{
							Name: "fake-stage",
						},
						{
							Name: "another-fake-stage",
						},
					},
				),
			)
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
		assertions func(map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
				require.Len(t, groups["another-fake-repo-url"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-repo-url",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(groupByImageRepo(testFreight, testCase.group))
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
		assertions func(map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
				require.Len(t, groups["another-fake-repo-url"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-repo-url",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-repo-url"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(groupByGitRepo(testFreight, testCase.group))
		})
	}
}

func TestGroupByChartRepo(t *testing.T) {
	testFreight := []kargoapi.Freight{
		{
			Charts: []kargoapi.Chart{{
				RegistryURL: "fake-registry-url",
				Name:        "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RegistryURL: "fake-registry-url",
				Name:        "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RegistryURL: "another-fake-registry-url",
				Name:        "fake-chart",
			}},
		},
		{
			Charts: []kargoapi.Chart{{
				RegistryURL: "another-fake-registry-url",
				Name:        "fake-chart",
			}},
		},
	}
	testCases := []struct {
		name       string
		group      string
		assertions func(map[string]*svcv1alpha1.FreightList)
	}{
		{
			name:  "without group filter",
			group: "",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 2)
				require.Len(t, groups["fake-registry-url/fake-chart"].Freight, 2)
				require.Len(t, groups["another-fake-registry-url/fake-chart"].Freight, 2)
			},
		},
		{
			name:  "with group filter",
			group: "fake-registry-url/fake-chart",
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups, 1)
				require.Len(t, groups["fake-registry-url/fake-chart"].Freight, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(groupByChartRepo(testFreight, testCase.group))
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
		assertions func(map[string]*svcv1alpha1.FreightList)
	}{
		{
			name: "order by tag",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*v1alpha1.Freight{
						{Images: []*v1alpha1.Image{{Tag: "b"}}},
						{Images: []*v1alpha1.Image{{Tag: "c"}}},
						{Images: []*v1alpha1.Image{{Tag: "a"}}},
					},
				},
			},
			orderBy: OrderByTag,
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
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
					Freight: []*v1alpha1.Freight{
						{Images: []*v1alpha1.Image{{Tag: "b"}}},
						{Images: []*v1alpha1.Image{{Tag: "c"}}},
						{Images: []*v1alpha1.Image{{Tag: "a"}}},
					},
				},
			},
			orderBy: OrderByTag,
			reverse: true,
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
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
					Freight: []*v1alpha1.Freight{
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now),
							},
						},
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now.Add(time.Hour)),
							},
						},
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now.Add(-time.Hour)),
							},
						},
					},
				},
			},
			orderBy: OrderByFirstSeen,
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, now.Add(-time.Hour), groups[""].Freight[0].Metadata.CreationTimestamp.AsTime())
				require.Equal(t, now, groups[""].Freight[1].Metadata.CreationTimestamp.AsTime())
				require.Equal(t, now.Add(time.Hour), groups[""].Freight[2].Metadata.CreationTimestamp.AsTime())
			},
		},
		{
			name: "reverse order by first seen",
			groups: map[string]*svcv1alpha1.FreightList{
				"": {
					Freight: []*v1alpha1.Freight{
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now),
							},
						},
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now.Add(time.Hour)),
							},
						},
						{
							Metadata: &svcv1meta1.ObjectMeta{
								CreationTimestamp: timestamppb.New(now.Add(-time.Hour)),
							},
						},
					},
				},
			},
			orderBy: OrderByFirstSeen,
			reverse: true,
			assertions: func(groups map[string]*svcv1alpha1.FreightList) {
				require.Len(t, groups[""].Freight, 3)
				require.Equal(t, now.Add(time.Hour), groups[""].Freight[0].Metadata.CreationTimestamp.AsTime())
				require.Equal(t, now, groups[""].Freight[1].Metadata.CreationTimestamp.AsTime())
				require.Equal(t, now.Add(-time.Hour), groups[""].Freight[2].Metadata.CreationTimestamp.AsTime())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sortFreightGroups(testCase.orderBy, testCase.reverse, testCase.groups)
			testCase.assertions(testCase.groups)
		})
	}
}
