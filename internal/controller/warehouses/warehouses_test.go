package warehouses

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	require.NotNil(t, e.getLatestFreightFromReposFn)
	require.NotNil(t, e.selectCommitsFn)
	require.NotNil(t, e.getLastCommitIDFn)
	require.NotNil(t, e.listTagsFn)
	require.NotNil(t, e.checkoutTagFn)
	require.NotNil(t, e.selectImagesFn)
	require.NotNil(t, e.getImageRefsFn)
	require.NotNil(t, e.selectChartsFn)
	require.NotNil(t, e.selectChartVersionFn)
	require.NotNil(t, e.selectCommitMetaFn)
	require.NotNil(t, e.createFreightFn)
}

func TestSyncWarehouse(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	testWarehouse := &kargoapi.Warehouse{
		Spec: &kargoapi.WarehouseSpec{},
	}
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error getting latest Freight from repos",
			reconciler: &reconciler{
				getLatestFreightFromReposFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.Freight, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error getting latest Freight from repos",
				)
			},
		},

		{
			name: "no latest Freight from repos",
			reconciler: &reconciler{
				getLatestFreightFromReposFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.Freight, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "latest Freight from repos isn't new",
			reconciler: &reconciler{
				getLatestFreightFromReposFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(
						schema.GroupResource{
							Group:    kargoapi.GroupVersion.Group,
							Resource: "Warehouse",
						},
						"",
					)
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "error creating Freight",
			reconciler: &reconciler{
				getLatestFreightFromReposFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error creating Freight")
			},
		},

		{
			name: "success creating Freight",
			reconciler: &reconciler{
				getLatestFreightFromReposFn: func(
					context.Context,
					*kargoapi.Warehouse,
				) (*kargoapi.Freight, error) {
					return &kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-freight",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createFreightFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
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
			_, err := testCase.reconciler.syncWarehouse(context.Background(), testWarehouse)
			testCase.assertions(t, err)
		})
	}
}

func TestGetLatestFreightFromRepos(t *testing.T) {
	const testWarehouseName = "fake-warehouse"

	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, *kargoapi.Freight, error)
	}{
		{
			name: "error getting latest git commits",
			reconciler: &reconciler{
				selectCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error syncing git repo subscription")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest images",
			reconciler: &reconciler{
				selectCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, nil
				},
				selectImagesFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error syncing image repo subscriptions",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest charts",
			reconciler: &reconciler{
				selectCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, nil
				},
				selectImagesFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.Image, error) {
					return nil, nil
				},
				selectChartsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.Chart, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error syncing chart repo subscriptions",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			reconciler: &reconciler{
				selectCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.GitCommit, error) {
					return []kargoapi.GitCommit{
						{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
					}, nil
				},
				selectImagesFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.Image, error) {
					return []kargoapi.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					}, nil
				},
				selectChartsFn: func(
					context.Context,
					string,
					[]kargoapi.RepoSubscription,
				) ([]kargoapi.Chart, error) {
					return []kargoapi.Chart{
						{
							RepoURL: "fake-repo",
							Name:    "fake-chart",
							Version: "fake-version",
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.NotEmpty(t, freight.Name)
				// All other fields should have a predictable value
				freight.Name = ""
				freight.OwnerReferences = nil
				require.Equal(
					t,
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-namespace",
						},
						Warehouse: testWarehouseName,
						Commits: []kargoapi.GitCommit{
							{
								RepoURL: "fake-url",
								ID:      "fake-commit",
							},
						},
						Images: []kargoapi.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
						Charts: []kargoapi.Chart{
							{
								RepoURL: "fake-repo",
								Name:    "fake-chart",
								Version: "fake-version",
							},
						},
					},
					freight,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, err := testCase.reconciler.getLatestFreightFromRepos(
				context.Background(),
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      testWarehouseName,
					},
					Spec: &kargoapi.WarehouseSpec{},
				},
			)
			testCase.assertions(t, freight, err)
		})
	}
}
