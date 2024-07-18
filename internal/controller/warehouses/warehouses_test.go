package warehouses

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	e := newReconciler(
		kubeClient,
		&credentials.FakeDB{},
	)
	require.NotNil(t, e.client)
	require.NotNil(t, e.credentialsDB)
	require.NotEmpty(t, e.imageSourceURLFnsByBaseURL)

	// Assert that all overridable behaviors were initialized to a default:
	require.NotNil(t, e.discoverArtifactsFn)
	require.NotNil(t, e.discoverCommitsFn)
	require.NotNil(t, e.discoverImagesFn)
	require.NotNil(t, e.discoverChartsFn)
	require.NotNil(t, e.buildFreightFromLatestArtifactsFn)
	require.NotNil(t, e.listCommitsFn)
	require.NotNil(t, e.listTagsFn)
	require.NotNil(t, e.discoverBranchHistoryFn)
	require.NotNil(t, e.discoverTagsFn)
	require.NotNil(t, e.getDiffPathsForCommitIDFn)
	require.NotNil(t, e.listFreightFn)
	require.NotNil(t, e.createFreightFn)
}

func TestSyncWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, kargoapi.WarehouseStatus, error)
	}{
		{
			name: "error discovering latest artifacts",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return nil, errors.New("something went wrong")
				},
			},
			warehouse: &kargoapi.Warehouse{
				Status: kargoapi.WarehouseStatus{
					DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{},
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering artifacts")

				// Ensure previous discovered artifacts are preserved.
				require.NotNil(t, status.DiscoveredArtifacts)
			},
		},

		{
			name: "Freight build error",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			warehouse: &kargoapi.Warehouse{},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "failed to build Freight from latest artifacts")
				require.NotNil(t, status.DiscoveredArtifacts)
			},
		},

		{
			name: "Freight for latest artifacts already exists",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
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
					freight.Items = []kargoapi.Freight{{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
						},
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "fake-warehouse",
						},
					}}
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-warehouse",
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				// Ensure that even if the Freight already exists, the status
				// is still updated with the latest Freight.
				require.NotEmpty(t, status.LastFreightID)
			},
		},

		{
			name: "error creating Freight",
			reconciler: &reconciler{
				discoverArtifactsFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
					return nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating Freight")
				require.NotNil(t, status.DiscoveredArtifacts)
				require.Empty(t, status.LastFreightID)
			},
		},

		{
			name: "automatic Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
				buildFreightFromLatestArtifactsFn: func(
					string,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				listFreightFn: func(context.Context, client.ObjectList, ...client.ListOption) error {
					return nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				require.NotEmpty(t, status.LastFreightID)
			},
		},

		{
			name: "manual Freight creation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.DiscoveredArtifacts)
				require.Empty(t, status.LastFreightID)
			},
		},

		{
			name: "updates refresh request status value",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "new",
					},
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					LastHandledRefresh: "old",
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, "new", status.LastHandledRefresh)
			},
		},

		{
			name: "updates observed generation",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					ObservedGeneration: 1,
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(2), status.ObservedGeneration)
			},
		},

		{
			name: "clears previous error message",
			reconciler: &reconciler{
				discoverArtifactsFn: func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error) {
					return &kargoapi.DiscoveredArtifacts{}, nil
				},
			},
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					FreightCreationPolicy: kargoapi.FreightCreationPolicyManual,
				},
				Status: kargoapi.WarehouseStatus{
					Message: "previous error",
				},
			},
			assertions: func(t *testing.T, status kargoapi.WarehouseStatus, err error) {
				require.NoError(t, err)
				require.Empty(t, status.Message)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncWarehouse(context.TODO(), testCase.warehouse)
			testCase.assertions(t, status, err)
		})
	}
}

func TestDiscoverArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.DiscoveredArtifacts, error)
	}{
		{
			name: "error discovering commits",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering commits")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "error discovering images",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering images")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "error discovering charts",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return []kargoapi.ImageDiscoveryResult{}, nil
				},
				discoverChartsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ChartDiscoveryResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error discovering charts")
				require.Nil(t, discoveredArtifacts)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				discoverCommitsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitDiscoveryResult, error) {
					return []kargoapi.GitDiscoveryResult{
						{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{
							{ID: "fake-commit"},
						}},
					}, nil
				},
				discoverImagesFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ImageDiscoveryResult, error) {
					return []kargoapi.ImageDiscoveryResult{
						{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{
							{Tag: "fake-tag"},
						}},
					}, nil
				},
				discoverChartsFn: func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.ChartDiscoveryResult, error) {
					return []kargoapi.ChartDiscoveryResult{
						{RepoURL: "fake-repo", Versions: []string{
							"fake-version",
						}},
					}, nil
				},
			},
			assertions: func(t *testing.T, discoveredArtifacts *kargoapi.DiscoveredArtifacts, err error) {
				require.NoError(t, err)
				require.Len(t, discoveredArtifacts.Git, 1)
				require.Len(t, discoveredArtifacts.Images, 1)
				require.Len(t, discoveredArtifacts.Charts, 1)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			discoveredArtifacts, err := testCase.reconciler.discoverArtifacts(
				context.TODO(),
				&kargoapi.Warehouse{},
			)
			testCase.assertions(t, discoveredArtifacts, err)
		})
	}
}

func TestBuildFreightFromLatestArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name:      "no artifacts discovered",
			artifacts: nil,
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no artifacts discovered")
				require.Nil(t, freight)
			},
		},
		{
			name: "no commits discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no commits discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no images discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no images discovered for repository")
				require.Nil(t, freight)
			},
		},
		{
			name: "no charts discovered",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
				},
				Charts: []kargoapi.ChartDiscoveryResult{
					{RepoURL: "fake-repo", Versions: []string{}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "no versions discovered for chart")
				require.Nil(t, freight)
			},
		},
		{
			name: "success",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
				Images: []kargoapi.ImageDiscoveryResult{
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
					{RepoURL: "fake-repo", References: []kargoapi.DiscoveredImageReference{{Tag: "fake-tag"}}},
				},
				Charts: []kargoapi.ChartDiscoveryResult{
					{RepoURL: "fake-repo", Versions: []string{"fake-version"}},
					{RepoURL: "fake-repo", Versions: []string{"fake-version"}},
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.Len(t, freight.Commits, 2)
				require.Len(t, freight.Images, 2)
				require.Len(t, freight.Charts, 2)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := (&reconciler{}).buildFreightFromLatestArtifacts(
				"fake-namespace",
				testCase.artifacts,
			)
			testCase.assertions(t, freight, err)
		})
	}
}
