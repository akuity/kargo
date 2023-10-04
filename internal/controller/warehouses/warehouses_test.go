package warehouses

import (
	"context"
	"testing"

	"github.com/pkg/errors"
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
	require.NotNil(t, e.getLatestCommitsFn)
	require.NotNil(t, e.getLatestImagesFn)
	require.NotNil(t, e.getLatestTagFn)
	require.NotNil(t, e.getLatestChartsFn)
	require.NotNil(t, e.getLatestChartVersionFn)
	require.NotNil(t, e.getLatestCommitMetaFn)
	require.NotNil(t, e.createFreightFn)
}

func TestSyncWarehouse(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	testWarehouse := &kargoapi.Warehouse{
		Spec: &kargoapi.WarehouseSpec{
			Subscriptions: &kargoapi.RepoSubscriptions{},
		},
	}
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(error)
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
			assertions: func(err error) {
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
			assertions: func(err error) {
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
			assertions: func(err error) {
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
			assertions: func(err error) {
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
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err :=
				testCase.reconciler.syncWarehouse(context.Background(), testWarehouse)
			testCase.assertions(err)
		})
	}
}

func TestGetLatestFreightFromRepos(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*kargoapi.Freight, error)
	}{
		{
			name: "error getting latest git commit",
			reconciler: &reconciler{
				getLatestCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.GitSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error syncing git repo subscription")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest images",
			reconciler: &reconciler{
				getLatestCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.GitSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, nil
				},
				getLatestImagesFn: func(
					context.Context,
					string,
					[]kargoapi.ImageSubscription,
				) ([]kargoapi.Image, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
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
				getLatestCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.GitSubscription,
				) ([]kargoapi.GitCommit, error) {
					return nil, nil
				},
				getLatestImagesFn: func(
					context.Context,
					string,
					[]kargoapi.ImageSubscription,
				) ([]kargoapi.Image, error) {
					return nil, nil
				},
				getLatestChartsFn: func(
					context.Context,
					string,
					[]kargoapi.ChartSubscription,
				) ([]kargoapi.Chart, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
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
				getLatestCommitsFn: func(
					context.Context,
					string,
					[]kargoapi.GitSubscription,
				) ([]kargoapi.GitCommit, error) {
					return []kargoapi.GitCommit{
						{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
					}, nil
				},
				getLatestImagesFn: func(
					context.Context,
					string,
					[]kargoapi.ImageSubscription,
				) ([]kargoapi.Image, error) {
					return []kargoapi.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					}, nil
				},
				getLatestChartsFn: func(
					context.Context,
					string,
					[]kargoapi.ChartSubscription,
				) ([]kargoapi.Chart, error) {
					return []kargoapi.Chart{
						{
							RegistryURL: "fake-registry",
							Name:        "fake-chart",
							Version:     "fake-version",
						},
					}, nil
				},
			},
			assertions: func(freight *kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.NotNil(t, freight)
				require.NotEmpty(t, freight.Name)
				require.NotEmpty(t, freight.ID)
				require.NotEmpty(t, freight.OwnerReferences)
				// All other fields should have a predictable value
				freight.Name = ""
				freight.ID = ""
				freight.OwnerReferences = nil
				require.Equal(
					t,
					&kargoapi.Freight{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-namespace",
						},
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
								RegistryURL: "fake-registry",
								Name:        "fake-chart",
								Version:     "fake-version",
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
			testCase.assertions(
				testCase.reconciler.getLatestFreightFromRepos(
					context.Background(),
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "fake-namespace",
						},
						Spec: &kargoapi.WarehouseSpec{
							Subscriptions: &kargoapi.RepoSubscriptions{},
						},
					},
				),
			)
		})
	}
}
