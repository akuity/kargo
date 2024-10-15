package freight

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestFindCommit(t *testing.T) {

	const testNamespace = "test-namespace"

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	const testRepoURL = "fake-repo-url"

	testOrigin1 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}
	testOrigin2 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "some-other-warehouse",
	}

	testCommit1 := kargoapi.GitCommit{
		RepoURL: testRepoURL,
		ID:      "fake-commit-1",
	}
	testCommit2 := kargoapi.GitCommit{
		RepoURL: testRepoURL,
		ID:      "fake-commit-2",
	}

	testCases := []struct {
		name          string
		client        func() client.Client
		stage         *kargoapi.Stage
		desiredOrigin *kargoapi.FreightOrigin
		freight       []kargoapi.FreightReference
		assertions    func(*testing.T, *kargoapi.GitCommit, error)
	}{
		{
			name:          "desired origin specified, but commit not found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin:  testOrigin2, // Wrong origin
					Commits: []kargoapi.GitCommit{testCommit2},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.GitCommit, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name:          "desired origin specified and commit is found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin:  testOrigin1, // Correct origin
					Commits: []kargoapi.GitCommit{testCommit1},
				},
				{
					Origin:  testOrigin2,
					Commits: []kargoapi.GitCommit{testCommit2},
				},
			},
			assertions: func(t *testing.T, commit *kargoapi.GitCommit, err error) {
				require.NoError(t, err)
				require.Equal(t, &testCommit1, commit)
			},
		},
		{
			name: "desired origin not specified and warehouse not found",
			client: func() client.Client {
				// This client will not find a Warehouse with a name matching the
				// desired origin
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.GitCommit, err error) {
				require.ErrorContains(t, err, "Warehouse")
				require.ErrorContains(t, err, "not found in namespace")
			},
		},
		{
			name: "desired origin not specified and cannot be inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							// This Warehouse has no subscription to the desired repo
							Subscriptions: []kargoapi.RepoSubscription{{
								Git: &kargoapi.GitSubscription{
									RepoURL: "not-the-right-repo",
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.GitCommit, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "desired origin not specified and more than one possible origin found",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Git: &kargoapi.GitSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin2.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Git: &kargoapi.GitSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						// This Stage requests Freight from two Warehouses that both get
						// commits from the same repo
						{Origin: testOrigin1},
						{Origin: testOrigin2},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.GitCommit, err error) {
				require.ErrorContains(
					t,
					err,
					"multiple requested Freight could potentially provide",
				)
			},
		},
		{
			name: "desired origin not specified and successfully inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Git: &kargoapi.GitSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{Origin: testOrigin1},
					},
				},
			},
			freight: []kargoapi.FreightReference{
				{
					Origin:  testOrigin1, // Correct origin
					Commits: []kargoapi.GitCommit{testCommit1},
				},
				{
					Origin:  testOrigin2,
					Commits: []kargoapi.GitCommit{testCommit1},
				},
			},
			assertions: func(t *testing.T, commit *kargoapi.GitCommit, err error) {
				require.NoError(t, err)
				require.Equal(t, &testCommit1, commit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var cl client.Client
			if testCase.client != nil {
				cl = testCase.client()
			}
			commit, err := FindCommit(
				context.Background(),
				cl,
				testCase.stage.Namespace,
				testCase.stage.Spec.RequestedFreight,
				testCase.desiredOrigin,
				testCase.freight,
				testRepoURL,
			)
			testCase.assertions(t, commit, err)
		})
	}
}

func TestFindImage(t *testing.T) {
	const testNamespace = "test-namespace"
	const testRepoURL = "fake-repo-url"

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testOrigin1 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}
	testOrigin2 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "some-other-warehouse",
	}

	testImage1 := kargoapi.Image{
		RepoURL: testRepoURL,
		Tag:     "fake-tag-1",
	}
	testImage2 := kargoapi.Image{
		RepoURL: testRepoURL,
		Tag:     "fake-tag-2",
	}

	testCases := []struct {
		name          string
		client        func() client.Client
		stage         *kargoapi.Stage
		desiredOrigin *kargoapi.FreightOrigin
		freight       []kargoapi.FreightReference
		assertions    func(*testing.T, *kargoapi.Image, error)
	}{
		{
			name:          "desired origin specified, but image not found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin2, // Wrong origin
					Images: []kargoapi.Image{testImage2},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Image, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name:          "desired origin specified and image is found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin1, // Correct origin
					Images: []kargoapi.Image{testImage1},
				},
				{
					Origin: testOrigin2,
					Images: []kargoapi.Image{testImage2},
				},
			},
			assertions: func(t *testing.T, image *kargoapi.Image, err error) {
				require.NoError(t, err)
				require.Equal(t, &testImage1, image)
			},
		},
		{
			name: "desired origin not specified and warehouse not found",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Image, err error) {
				require.ErrorContains(t, err, "Warehouse")
				require.ErrorContains(t, err, "not found in namespace")
			},
		},
		{
			name: "desired origin not specified and cannot be inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							// This Warehouse has no subscription to the desired repo
							Subscriptions: []kargoapi.RepoSubscription{{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "not-the-right-repo",
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Image, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "desired origin not specified and more than one possible origin found",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Image: &kargoapi.ImageSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin2.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Image: &kargoapi.ImageSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						// This Stage requests Freight from two Warehouses that both get
						// images from the same repo
						{Origin: testOrigin1},
						{Origin: testOrigin2},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Image, err error) {
				require.ErrorContains(
					t,
					err,
					"multiple requested Freight could potentially provide",
				)
			},
		},
		{
			name: "desired origin not specified and successfully inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Image: &kargoapi.ImageSubscription{
									RepoURL: testRepoURL,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{Origin: testOrigin1},
					},
				},
			},
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin1, // Correct origin
					Images: []kargoapi.Image{testImage1},
				},
				{
					Origin: testOrigin2,
					Images: []kargoapi.Image{testImage1},
				},
			},
			assertions: func(t *testing.T, image *kargoapi.Image, err error) {
				require.NoError(t, err)
				require.Equal(t, &testImage1, image)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var cl client.Client
			if testCase.client != nil {
				cl = testCase.client()
			}
			image, err := FindImage(
				context.Background(),
				cl,
				testCase.stage.Namespace,
				testCase.stage.Spec.RequestedFreight,
				testCase.desiredOrigin,
				testCase.freight,
				testRepoURL)
			testCase.assertions(t, image, err)
		})
	}
}

func TestFindChart(t *testing.T) {
	const testNamespace = "test-namespace"
	const testRepoURL = "fake-repo-url"
	const testChartName = "fake-chart"

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testOrigin1 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}
	testOrigin2 := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "some-other-warehouse",
	}

	testChart1 := kargoapi.Chart{
		RepoURL: testRepoURL,
		Name:    testChartName,
		Version: "fake-version-1",
	}
	testChart2 := kargoapi.Chart{
		RepoURL: testRepoURL,
		Name:    testChartName,
		Version: "fake-version-2",
	}

	testCases := []struct {
		name          string
		client        func() client.Client
		stage         *kargoapi.Stage
		desiredOrigin *kargoapi.FreightOrigin
		freight       []kargoapi.FreightReference
		assertions    func(*testing.T, *kargoapi.Chart, error)
	}{
		{
			name:          "desired origin specified, but chart not found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin2, // Wrong origin
					Charts: []kargoapi.Chart{testChart2},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name:          "desired origin specified and chart is found",
			stage:         &kargoapi.Stage{},
			desiredOrigin: &testOrigin1,
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin1, // Correct origin
					Charts: []kargoapi.Chart{testChart1},
				},
				{
					Origin: testOrigin2,
					Charts: []kargoapi.Chart{testChart2},
				},
			},
			assertions: func(t *testing.T, chart *kargoapi.Chart, err error) {
				require.NoError(t, err)
				require.Equal(t, &testChart1, chart)
			},
		},
		{
			name: "desired origin not specified and warehouse not found",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "Warehouse")
				require.ErrorContains(t, err, "not found in namespace")
			},
		},
		{
			name: "desired origin not specified and cannot be inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							// This Warehouse has no subscription to the desired repo
							Subscriptions: []kargoapi.RepoSubscription{{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "not-the-right-repo",
									Name:    "not-the-right-chart",
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin1}},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Chart, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "desired origin not specified and more than one possible origin found",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: testRepoURL,
									Name:    testChartName,
								},
							}},
						},
					},
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin2.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: testRepoURL,
									Name:    testChartName,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						// This Stage requests Freight from two Warehouses that both get
						// the same chart from the same repo
						{Origin: testOrigin1},
						{Origin: testOrigin2},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Chart, err error) {
				require.ErrorContains(
					t,
					err,
					"multiple requested Freight could potentially provide",
				)
			},
		},
		{
			name: "desired origin not specified and successfully inferred",
			client: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testNamespace,
							Name:      testOrigin1.Name,
						},
						Spec: kargoapi.WarehouseSpec{
							Subscriptions: []kargoapi.RepoSubscription{{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: testRepoURL,
									Name:    testChartName,
								},
							}},
						},
					},
				).Build()
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{Origin: testOrigin1},
					},
				},
			},
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin1, // Correct origin
					Charts: []kargoapi.Chart{testChart1},
				},
				{
					Origin: testOrigin2,
					Charts: []kargoapi.Chart{testChart1},
				},
			},
			assertions: func(t *testing.T, chart *kargoapi.Chart, err error) {
				require.NoError(t, err)
				require.Equal(t, &testChart1, chart)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var cl client.Client
			if testCase.client != nil {
				cl = testCase.client()
			}
			chart, err := FindChart(
				context.Background(),
				cl,
				testCase.stage.Namespace,
				testCase.stage.Spec.RequestedFreight,
				testCase.desiredOrigin,
				testCase.freight,
				testRepoURL,
				testChartName,
			)
			testCase.assertions(t, chart, err)
		})
	}
}
