package function

import (
	"context"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_warehouse(t *testing.T) {
	tests := []struct {
		name       string
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "valid warehouse name",
			args: []any{"test-warehouse"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				origin, ok := result.(kargoapi.FreightOrigin)
				assert.True(t, ok)
				assert.Equal(t, kargoapi.FreightOriginKindWarehouse, origin.Kind)
				assert.Equal(t, "test-warehouse", origin.Name)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Empty(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"test-warehouse", "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Empty(t, result)
			},
		},
		{
			name: "invalid argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Empty(t, result)
			},
		},
		{
			name: "empty string name",
			args: []any{""},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "name must not be empty")
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := warehouse(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_getCommitFromFreight(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		objects     []client.Object
		freightReqs []kargoapi.FreightRequest
		freightRefs []kargoapi.FreightReference
		args        []any
		assertions  func(t *testing.T, result any, err error)
	}{
		{
			name: "repo URL only",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: testProject,
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{
							{
								Git: &kargoapi.GitSubscription{
									RepoURL: "https://github.com/example/repo",
								},
							},
						},
					},
				},
			},
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "https://github.com/example/repo",
							ID:      "abc123",
						},
					},
				},
			},
			args: []any{"https://github.com/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				commit, ok := result.(*kargoapi.GitCommit)
				assert.True(t, ok)
				assert.Equal(t, "https://github.com/example/repo", commit.RepoURL)
				assert.Equal(t, "abc123", commit.ID)
			},
		},
		{
			name: "repo URL and origin",
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "https://github.com/example/repo",
							ID:      "def456",
						},
					},
				},
			},
			args: []any{"https://github.com/example/repo", kargoapi.FreightOrigin{
				Kind: "Warehouse",
				Name: "fake-warehouse",
			}},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				commit, ok := result.(*kargoapi.GitCommit)
				assert.True(t, ok)
				assert.Equal(t, "https://github.com/example/repo", commit.RepoURL)
				assert.Equal(t, "def456", commit.ID)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"https://github.com/example/repo", kargoapi.FreightOrigin{}, "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "first argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid second argument type",
			args: []any{"https://github.com/example/repo", "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "second argument must be FreightOrigin")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getCommitFromFreight(
				ctx,
				c,
				testProject,
				tt.freightReqs,
				tt.freightRefs,
			)

			result, err := fn(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_getCommitFromDiscoveredArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "wrong number of args",
			args: []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name: "invalid arg type",
			args: []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "https://example.com/repo.git",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag: "def456",
								CreatorDate: &metav1.Time{
									Time: time.Date(2023, 9, 17, 2, 0, 0, 0, time.UTC),
								},
							},
							{
								Tag: "abc123",
								CreatorDate: &metav1.Time{
									Time: time.Date(2023, 9, 17, 1, 0, 0, 0, time.UTC),
								},
							},
						},
					},
				},
			},
			args: []any{"https://example.com/repo.git"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				commit, ok := result.(kargoapi.DiscoveredCommit)
				require.True(t, ok)
				require.Equal(t, "def456", commit.Tag)
			},
		},
		{
			name: "nil artifacts",
			args: []any{"https://example.com/repo.git"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
		{
			name:      "no commits found",
			artifacts: &kargoapi.DiscoveredArtifacts{},
			args:      []any{"https://example.com/repo.git"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := getCommitFromDiscoveredArtifacts(tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}

func Test_getImageFromFreight(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		objects     []client.Object
		freightReqs []kargoapi.FreightRequest
		freightRefs []kargoapi.FreightReference
		args        []any
		assertions  func(t *testing.T, result any, err error)
	}{
		{
			name: "repo URL only",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: testProject,
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "registry.example.com/app",
								},
							},
						},
					},
				},
			},
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Images: []kargoapi.Image{
						{
							RepoURL: "registry.example.com/app",
						},
					},
				},
			},
			args: []any{"registry.example.com/app"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				image, ok := result.(*kargoapi.Image)
				assert.True(t, ok)
				assert.Equal(t, "registry.example.com/app", image.RepoURL)
			},
		},
		{
			name: "repo URL and origin",
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Images: []kargoapi.Image{
						{
							RepoURL: "registry.example.com/app",
						},
					},
				},
			},
			args: []any{"registry.example.com/app", kargoapi.FreightOrigin{
				Kind: "Warehouse",
				Name: "fake-warehouse",
			}},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				image, ok := result.(*kargoapi.Image)
				assert.True(t, ok)
				assert.Equal(t, "registry.example.com/app", image.RepoURL)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"registry.example.com/app", kargoapi.FreightOrigin{}, "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "first argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid second argument type",
			args: []any{"registry.example.com/app", "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "second argument must be FreightOrigin")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getImageFromFreight(
				ctx,
				c,
				testProject,
				tt.freightReqs,
				tt.freightRefs,
			)

			result, err := fn(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_getImageFromDiscoveredArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "wrong number of args",
			args: []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name: "invalid arg type",
			args: []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Images: []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "docker.io/example/repo",
						References: []kargoapi.DiscoveredImageReference{
							{Tag: "v1.0.0"},
							{Tag: "v1.1.0"},
						},
					},
				},
			},
			args: []any{"docker.io/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				img, ok := result.(kargoapi.DiscoveredImageReference)
				require.True(t, ok)
				require.Equal(t, "v1.0.0", img.Tag)
			},
		},
		{
			name: "nil artifacts",
			args: []any{"docker.io/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
		{
			name:      "no images found",
			artifacts: &kargoapi.DiscoveredArtifacts{},
			args:      []any{"docker.io/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := getImageFromDiscoveredArtifacts(tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}

func Test_getChartFromFreight(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		objects     []client.Object
		freightReqs []kargoapi.FreightRequest
		freightRefs []kargoapi.FreightReference
		args        []any
		assertions  func(t *testing.T, result any, err error)
	}{
		{
			name: "repo URL only",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: testProject,
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{
							{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "oci://registry.example.com/chart",
								},
							},
						},
					},
				},
			},
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Charts: []kargoapi.Chart{
						{
							RepoURL: "oci://registry.example.com/chart",
						},
					},
				},
			},
			args: []any{"oci://registry.example.com/chart"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				chart, ok := result.(*kargoapi.Chart)
				assert.True(t, ok)
				assert.Equal(t, "oci://registry.example.com/chart", chart.RepoURL)
			},
		},
		{
			name: "repo URL and chart name",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: testProject,
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{
							{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "https://charts.example.com",
									Name:    "fake-chart",
								},
							},
						},
					},
				},
			},
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Charts: []kargoapi.Chart{
						{
							RepoURL: "https://charts.example.com",
							Name:    "fake-chart",
						},
					},
				},
			},
			args: []any{"https://charts.example.com", "fake-chart"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				chart, ok := result.(*kargoapi.Chart)
				assert.True(t, ok)
				assert.Equal(t, "https://charts.example.com", chart.RepoURL)
				assert.Equal(t, "fake-chart", chart.Name)
			},
		},
		{
			name: "repo URL and origin",
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Charts: []kargoapi.Chart{
						{
							RepoURL: "oci://registry.example.com/chart",
						},
					},
				},
			},
			args: []any{"oci://registry.example.com/chart", kargoapi.FreightOrigin{
				Kind: "Warehouse",
				Name: "fake-warehouse",
			}},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				chart, ok := result.(*kargoapi.Chart)
				assert.True(t, ok)
				assert.Equal(t, "oci://registry.example.com/chart", chart.RepoURL)
			},
		},
		{
			name: "repo URL, chart name, and origin",
			freightReqs: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
				},
			},
			freightRefs: []kargoapi.FreightReference{
				{
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
						Kind: "Warehouse",
					},
					Charts: []kargoapi.Chart{
						{
							RepoURL: "https://charts.example.com",
							Name:    "fake-chart",
						},
					},
				},
			},
			args: []any{"https://charts.example.com", "fake-chart", kargoapi.FreightOrigin{
				Kind: "Warehouse",
				Name: "fake-warehouse",
			}},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				chart, ok := result.(*kargoapi.Chart)
				assert.True(t, ok)
				assert.Equal(t, "https://charts.example.com", chart.RepoURL)
				assert.Equal(t, "fake-chart", chart.Name)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-3 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"url", "name", kargoapi.FreightOrigin{}, "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-3 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "first argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid second argument type",
			args: []any{"https://charts.example.com", 123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "second argument must be string or FreightOrigin")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid third argument with string second argument",
			args: []any{"https://charts.example.com", "fake-chart", "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "third argument must be FreightOrigin")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid third argument with origin second argument",
			args: []any{"https://charts.example.com", kargoapi.FreightOrigin{}, "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "when using three arguments, second argument must be string")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getChartFromFreight(
				ctx,
				c,
				testProject,
				tt.freightReqs,
				tt.freightRefs,
			)

			result, err := fn(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_getChartFromDiscoveredArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "wrong number of args",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1-2 arguments, got 0")
			},
		},
		{
			name: "1st arg type invalid",
			args: []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "2nd arg type invalid",
			args: []any{"oci://ghcr.io/akuity/kargo-charts/kargo", 2},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "second argument must be string, got int")
			},
		},
		{
			name: "success with repo URL only",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						RepoURL:  "oci://ghcr.io/other/chart-repo",
						Versions: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
					},
					{
						RepoURL:  "oci://ghcr.io/akuity/kargo-charts/kargo",
						Versions: []string{"v2.3.0", "v2.2.0", "v2.1.0"},
					},
				},
			},
			args: []any{"oci://ghcr.io/akuity/kargo-charts/kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				commit, ok := result.(kargoapi.Chart)
				require.True(t, ok)
				require.Equal(t, "v2.3.0", commit.Version)
			},
		},
		{
			name: "success with repo URL and name",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						Name:     "other-chart",
						RepoURL:  "oci://ghcr.io/other/chart-repo",
						Versions: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
					},
					{
						Name:     "kargo",
						RepoURL:  "https://charts.example.com",
						Versions: []string{"v2.3.0", "v2.2.0", "v2.1.0"},
					},
				},
			},
			args: []any{"https://charts.example.com", "kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				commit, ok := result.(kargoapi.Chart)
				require.True(t, ok)
				require.Equal(t, "v2.3.0", commit.Version)
			},
		},
		{
			name: "nil artifacts",
			args: []any{"oci://ghcr.io/akuity/kargo-charts/kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
		{
			name:      "no charts found",
			artifacts: &kargoapi.DiscoveredArtifacts{},
			args:      []any{"oci://ghcr.io/akuity/kargo-charts/kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := getChartFromDiscoveredArtifacts(tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}

func Test_getArtifactFromFreight(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	testCases := []struct {
		name        string
		objects     []client.Object
		freightReqs []kargoapi.FreightRequest
		freightRefs []kargoapi.FreightReference
		args        []any
		assertions  func(t *testing.T, result any, err error)
	}{
		{
			name: "subscription name only",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: testProject,
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Subscription: &kargoapi.Subscription{
								SubscriptionType: "fake-type",
								Name:             "fake-sub",
							},
						}},
					},
				},
			},
			freightReqs: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Name: "fake-warehouse",
					Kind: "Warehouse",
				},
			}},
			freightRefs: []kargoapi.FreightReference{{
				Origin: kargoapi.FreightOrigin{
					Name: "fake-warehouse",
					Kind: "Warehouse",
				},
				Artifacts: []kargoapi.ArtifactReference{{
					SubscriptionName: "fake-sub",
					Version:          "fake-version",
				}},
			}},
			args: []any{"fake-sub"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				artifact, ok := result.(expressionFriendlyArtifactReference)
				assert.True(t, ok)
				assert.Equal(t, "fake-sub", artifact.SubscriptionName)
				assert.Equal(t, "fake-version", artifact.Version)
			},
		},
		{
			name: "subscription name and origin",
			freightReqs: []kargoapi.FreightRequest{{
				Origin: kargoapi.FreightOrigin{
					Name: "fake-warehouse",
					Kind: "Warehouse",
				},
			}},
			freightRefs: []kargoapi.FreightReference{{
				Origin: kargoapi.FreightOrigin{
					Name: "fake-warehouse",
					Kind: "Warehouse",
				},
				Artifacts: []kargoapi.ArtifactReference{{
					SubscriptionName: "fake-sub",
					Version:          "fake-version",
				}},
			}},
			args: []any{"fake-sub", kargoapi.FreightOrigin{
				Kind: "Warehouse",
				Name: "fake-warehouse",
			}},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				artifact, ok := result.(expressionFriendlyArtifactReference)
				assert.True(t, ok)
				assert.Equal(t, "fake-sub", artifact.SubscriptionName)
				assert.Equal(t, "fake-version", artifact.Version)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"fake-sub", kargoapi.FreightOrigin{}, "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1-2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "first argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid second argument type",
			args: []any{"fake-sub", "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "second argument must be FreightOrigin")
				assert.Nil(t, result)
			},
		},
		{
			name: "success",
			args: []any{"fake-sub"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				Build()
			result, err := getArtifactFromFreight(
				ctx,
				c,
				testProject,
				testCase.freightReqs,
				testCase.freightRefs,
			)(testCase.args...)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_getArtifactFromDiscoveredArtifacts(t *testing.T) {
	testCases := []struct {
		name       string
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "wrong number of args",
			args: []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name: "invalid arg type",
			args: []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Results: []kargoapi.DiscoveryResult{{
					SubscriptionName: "fake-sub",
					ArtifactReferences: []kargoapi.ArtifactReference{
						{Version: "v1.0.0"},
						{Version: "v1.1.0"},
					},
				}},
			},
			args: []any{"fake-sub"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				artifact, ok := result.(expressionFriendlyArtifactReference)
				require.True(t, ok)
				require.Equal(t, "v1.0.0", artifact.Version)
			},
		},
		{
			name: "artifacts nil",
			args: []any{"fake-sub"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
		{
			name: "requested artifact not found",
			artifacts: &kargoapi.DiscoveredArtifacts{
				Results: []kargoapi.DiscoveryResult{{
					SubscriptionName: "fake-sub",
					ArtifactReferences: []kargoapi.ArtifactReference{
						{Version: "v1.0.0"},
						{Version: "v1.1.0"},
					},
				}},
			},
			args: []any{"wrong-sub"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := getArtifactFromDiscoveredArtifacts(
				testCase.artifacts,
			)(testCase.args...)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_getConfigMap(t *testing.T) {
	const testProject = "fake-project"
	const testConfigMap = "fake-configmap"

	testData := map[string]string{
		"foo": "bar",
	}

	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name       string
		objects    []client.Object
		args       []any
		cache      *cache.Cache
		assertions func(t *testing.T, cache *cache.Cache, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{testConfigMap, "extra"},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid argument type",
			args: []any{123},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "ConfigMap not found",
			args: []any{testConfigMap},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.IsType(t, map[string]string{}, result)
				assert.Empty(t, result)
				assert.Nil(t, cache)
			},
		},
		{
			name: "success",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testConfigMap,
					},
					Data: testData,
				},
			},
			args: []any{testConfigMap},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, testData, result)
				assert.Nil(t, cache)
			},
		},
		{
			name: "success with cache",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testConfigMap,
					},
					Data: testData,
				},
			},
			cache: cache.New(cache.NoExpiration, cache.NoExpiration),
			args:  []any{testConfigMap},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, testData, result)

				// Check if the item is in the cache
				data, ok := cache.Get(getCacheKey(cacheKeyPrefixConfigMap, testProject, testConfigMap))
				assert.True(t, ok)
				assert.Equal(t, testData, data)
			},
		},
		{
			name: "success from cache",
			cache: cache.NewFrom(cache.NoExpiration, cache.NoExpiration, map[string]cache.Item{
				getCacheKey(cacheKeyPrefixConfigMap, testProject, testConfigMap): {
					Object: testData,
				},
			}),
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testConfigMap,
					},
					Data: map[string]string{
						// This data should not be used
						"foo": "baz",
					},
				},
			},
			args: []any{testConfigMap},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, testData, result)

				// Check if the item data did not change
				data, ok := cache.Get(getCacheKey(cacheKeyPrefixConfigMap, testProject, testConfigMap))
				assert.True(t, ok)
				assert.Equal(t, testData, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getConfigMap(ctx, c, tt.cache, testProject)

			result, err := fn(tt.args...)
			tt.assertions(t, tt.cache, result, err)
		})
	}
}

func Test_getSecret(t *testing.T) {
	const testProject = "fake-project"
	const testSecret = "fake-secret"

	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name            string
		objects         []client.Object
		args            []any
		cache           *cache.Cache
		hasDirectAccess bool
		assertions      func(t *testing.T, cache *cache.Cache, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{testSecret, "extra"},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid argument type",
			args: []any{123},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "Secret not found",
			args: []any{testSecret},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.IsType(t, map[string]string{}, result)
				assert.Empty(t, result)
				assert.Nil(t, cache)
			},
		},
		{
			name:            "success",
			hasDirectAccess: true,
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testSecret,
					},
					Data: map[string][]byte{
						"foo": []byte("bar"),
					},
				},
			},
			args: []any{testSecret},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"foo": "bar"}, result)
				assert.Nil(t, cache)
			},
		},
		{
			name:            "success with cache",
			hasDirectAccess: true,
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testSecret,
					},
					Data: map[string][]byte{
						"foo": []byte("bar"),
					},
				},
			},
			cache: cache.New(cache.NoExpiration, cache.NoExpiration),
			args:  []any{testSecret},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"foo": "bar"}, result)

				// Check if the item is in the cache
				data, ok := cache.Get(getCacheKey(cacheKeyPrefixSecret, testProject, testSecret))
				assert.True(t, ok)
				assert.Equal(t, map[string]string{"foo": "bar"}, data)
			},
		},
		{
			name:            "success from cache",
			hasDirectAccess: true,
			cache: cache.NewFrom(cache.NoExpiration, cache.NoExpiration, map[string]cache.Item{
				getCacheKey(cacheKeyPrefixSecret, testProject, testSecret): {
					Object: map[string]string{"foo": "bar"},
				},
			}),
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testSecret,
					},
					Data: map[string][]byte{
						// This data should not be used
						"foo": []byte("baz"),
					},
				},
			},
			args: []any{testSecret},
			assertions: func(t *testing.T, cache *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"foo": "bar"}, result)

				// Check if the item data did not change
				data, ok := cache.Get(getCacheKey(cacheKeyPrefixSecret, testProject, testSecret))
				assert.True(t, ok)
				assert.Equal(t, map[string]string{"foo": "bar"}, data)
			},
		},
		{
			name:            "success with no direct access but is generic",
			hasDirectAccess: false,
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testSecret,
						Labels: map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
						},
					},
					Data: map[string][]byte{
						"foo": []byte("bar"),
					},
				},
			},
			args: []any{testSecret},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"foo": "bar"}, result)
			},
		},
		{
			name:            "no direct access and not generic returns empty data",
			hasDirectAccess: false,
			objects: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testSecret,
					},
					Data: map[string][]byte{
						"foo": []byte("bar"),
					},
				},
			},
			args: []any{testSecret},
			assertions: func(t *testing.T, _ *cache.Cache, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{}, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getSecret(ctx, c, tt.cache, testProject, tt.hasDirectAccess)

			result, err := fn(tt.args...)
			tt.assertions(t, tt.cache, result, err)
		})
	}
}

func Test_getConfigMap_getSecret_no_cache_key_collision(t *testing.T) {
	const testProject = "fake-project"
	const testIdenticalName = "fake-name"

	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name       string
		objects    []client.Object
		args       []any
		cache      *cache.Cache
		assertions func(t *testing.T, cache *cache.Cache)
	}{
		{
			name: "ConfigMap and Secret with identical names",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testIdenticalName,
					},
					Data: map[string]string{
						"type": "configmap",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testIdenticalName,
					},
					Data: map[string][]byte{
						"type": []byte("secret"),
					},
				},
			},
			cache: cache.New(cache.NoExpiration, cache.NoExpiration),
			args:  []any{testIdenticalName},
			assertions: func(t *testing.T, cache *cache.Cache) {
				// Check if both items are in the cache with different keys
				data, ok := cache.Get(getCacheKey(cacheKeyPrefixConfigMap, testProject, testIdenticalName))
				assert.True(t, ok)
				assert.Equal(t, map[string]string{"type": "configmap"}, data)

				data, ok = cache.Get(getCacheKey(cacheKeyPrefixSecret, testProject, testIdenticalName))
				assert.True(t, ok)
				assert.Equal(t, map[string]string{"type": "secret"}, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			cfgFn := getConfigMap(ctx, c, tt.cache, testProject)
			_, err := cfgFn(tt.args...)
			assert.NoError(t, err)

			secretFn := getSecret(ctx, c, tt.cache, testProject, true)
			_, err = secretFn(tt.args...)
			assert.NoError(t, err)

			tt.assertions(t, tt.cache)
		})
	}
}

func Test_getStatus(t *testing.T) {
	tests := []struct {
		name             string
		currentStepAlias string
		stepExecMetas    kargoapi.StepExecutionMetadataList
		args             []any
		assertions       func(t *testing.T, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Empty(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Empty(t, result)
			},
		},
		{
			name: "one empty argument",
			args: []any{""},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "must not be empty")
				assert.Empty(t, result)
			},
		},
		{
			name:             "basic usage; no hit",
			currentStepAlias: "step-2",
			stepExecMetas: kargoapi.StepExecutionMetadataList{{
				Alias:  "step-1",
				Status: kargoapi.PromotionStepStatusSucceeded,
			}},
			args: []any{"non-existent-step"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Empty(t, result)
			},
		},
		{
			name:             "basic usage; hit",
			currentStepAlias: "step-2",
			stepExecMetas: kargoapi.StepExecutionMetadataList{{
				Alias:  "step-1",
				Status: kargoapi.PromotionStepStatusSucceeded,
			}},
			args: []any{"step-1"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), result)
			},
		},
		{
			name:             "used in a task; no alias match",
			currentStepAlias: "task-2::step-2",
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Alias:  "step-1", // No task namespace; correct alias
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Alias:  "task-1::step-2", // Correct task namespace; wrong alias
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
			},
			args: []any{"step-1"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Empty(t, result)
			},
		},
		{
			name:             "used in a task; no namespace match",
			currentStepAlias: "task-2::step-2",
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Alias:  "step-1", // No task namespace; correct alias
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
				{
					Alias:  "task-1::step-2", // Wrong task namespace; correct alias
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
			},
			args: []any{"step-1"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Empty(t, result)
			},
		},
		{
			name:             "used in a task; hit",
			currentStepAlias: "task-2::step-2",
			stepExecMetas: kargoapi.StepExecutionMetadataList{
				{
					Alias:  "task-2::step-1",
					Status: kargoapi.PromotionStepStatusSucceeded,
				},
			},
			args: []any{"step-1"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, string(kargoapi.PromotionStepStatusSucceeded), result)
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fn := getStatus(testCase.currentStepAlias, testCase.stepExecMetas)
			result, err := fn(testCase.args...)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_freightMetadata(t *testing.T) {
	const testProject = "fake-project"
	const testFreightName = "fake-freight"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	// Sample metadata for testing
	testMetadata := map[string]any{
		"deployment-id": "abc123",
		"environment":   "staging",
	}

	expectedMetadata := map[string]any{
		"deployment-config": testMetadata,
		"build-number":      float64(42),
		"issue":             "#1234",
	}

	// Create a freight object with metadata
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testFreightName,
			Namespace: testProject,
		},
		Status: kargoapi.FreightStatus{
			Metadata: map[string]apiextensionsv1.JSON{
				"deployment-config": {Raw: []byte(`{"deployment-id":"abc123","environment":"staging"}`)},
				"build-number":      {Raw: []byte(`42`)},
				"issue":             {Raw: []byte(`"#1234"`)},
			},
		},
	}

	tests := []struct {
		name       string
		objects    []client.Object
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{testFreightName, "deployment-config", "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "empty freight ref name",
			args: []any{"", "deployment-config"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "freight ref name must not be empty")
				assert.Nil(t, result)
			},
		},
		{
			name:    "invalid second argument type",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, 123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name:    "empty metadata key",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, ""},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "metadata key must not be empty")
				assert.Nil(t, result)
			},
		},
		{
			name:    "freight not found",
			objects: []client.Object{}, // No freight objects
			args:    []any{testFreightName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:    "freight not found, two arg",
			objects: []client.Object{}, // No freight objects
			args:    []any{testFreightName, "deployment-config"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "freight exists but no metadata",
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testFreightName,
						Namespace: testProject,
					},
					Status: kargoapi.FreightStatus{}, // Empty status with no metadata
				},
			},
			args: []any{testFreightName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:    "metadata key not found, two arg",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, "non-existent-key"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:    "successful metadata retrieval, two arg - string map",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, "deployment-config"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, testMetadata, result)
			},
		},
		{
			name:    "successful metadata retrieval, two arg - number",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, "build-number"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				// JSON unmarshaling converts numbers to float64
				assert.Equal(t, float64(42), result)
			},
		},
		{
			name:    "successful metadata retrieval, two arg - string",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName, "issue"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "#1234", result)
			},
		},
		{
			name:    "successful metadata retrieval, single arg - string map",
			objects: []client.Object{testFreight},
			args:    []any{testFreightName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, expectedMetadata, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := freightMetadata(ctx, c, testProject)

			result, err := fn(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_stageMetadata(t *testing.T) {
	const testProject = "fake-project"
	const testStageName = "fake-stage"

	scheme := runtime.NewScheme()
	assert.NoError(t, kargoapi.AddToScheme(scheme))

	// Sample metadata for testing
	testMetadata := map[string]any{
		"deployment-id": "abc123",
		"environment":   "staging",
	}

	expectedMetadata := map[string]any{
		"deployment-config": testMetadata,
	}

	// Create a stage object with metadata
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testStageName,
			Namespace: testProject,
		},
		Status: kargoapi.StageStatus{
			Metadata: map[string]apiextensionsv1.JSON{
				"deployment-config": {
					Raw: []byte(`{"deployment-id":"abc123","environment":"staging"}`),
				},
			},
		},
	}

	tests := []struct {
		name       string
		objects    []client.Object
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{testStageName, "extra"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "empty stage name",
			args: []any{""},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "stage name must not be empty")
				assert.Nil(t, result)
			},
		},
		{
			name:    "stage not found",
			objects: []client.Object{}, // No stage objects
			args:    []any{testStageName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "stage exists but no metadata",
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testProject,
					},
					Status: kargoapi.StageStatus{}, // Empty status with no metadata
				},
			},
			args: []any{testStageName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:    "successful metadata retrieval",
			objects: []client.Object{testStage},
			args:    []any{testStageName},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, expectedMetadata, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := stageMetadata(ctx, c, testProject)

			result, err := fn(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_semverDiff(t *testing.T) {
	tests := []struct {
		name       string
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "major version difference",
			args: []any{"1.1.1", "2.2.2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Major", result)
			},
		},
		{
			name: "minor version difference",
			args: []any{"1.1.1", "1.2.2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Minor", result)
			},
		},
		{
			name: "patch version difference",
			args: []any{"1.1.1", "1.1.2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Patch", result)
			},
		},
		{
			name: "metadata difference",
			args: []any{"1.1.1+build1", "1.1.1+build2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Metadata", result)
			},
		},
		{
			name: "no difference",
			args: []any{"1.2.3", "1.2.3"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "None", result)
			},
		},
		{
			name: "invalid first version",
			args: []any{"invalid", "1.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Incomparable", result)
			},
		},
		{
			name: "invalid second version",
			args: []any{"1.0.0", "invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Incomparable", result)
			},
		},
		{
			name: "both versions invalid",
			args: []any{"invalid1", "invalid2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Incomparable", result)
			},
		},
		{
			name: "semver with prerelease",
			args: []any{"1.0.0-alpha", "1.0.0-beta"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "None", result) // Prerelease versions are considered equal for diff purposes
			},
		},
		{
			name: "complex semver with metadata and prerelease",
			args: []any{"1.0.0-alpha.1+build.1", "1.0.0-alpha.1+build.2"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Metadata", result)
			},
		},
		{
			name: "major difference with prerelease",
			args: []any{"1.0.0-alpha", "2.0.0-alpha"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Major", result)
			},
		},
		{
			name: "minor difference with prerelease",
			args: []any{"1.1.0-alpha", "1.2.0-alpha"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Minor", result)
			},
		},
		{
			name: "patch difference with prerelease",
			args: []any{"1.1.1-alpha", "1.1.2-alpha"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Patch", result)
			},
		},
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "one argument",
			args: []any{"1.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"1.0.0", "2.0.0", "3.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 2 arguments")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid first argument type",
			args: []any{123, "1.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "first argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid second argument type",
			args: []any{"1.0.0", 123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "second argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "empty version strings",
			args: []any{"", ""},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Incomparable", result)
			},
		},
		{
			name: "loose semver format",
			args: []any{"v1.0.0", "v2.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Major", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := semverDiff(tt.args...)
			tt.assertions(t, result, err)
		})
	}
}

func Test_semverParse(t *testing.T) {
	testCases := []struct {
		name       string
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name: "no arguments",
			args: []any{},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "too many arguments",
			args: []any{"1.0.0", "2.0.0"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "expected 1 argument")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid argument type",
			args: []any{123},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "argument must be string")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid semver",
			args: []any{"invalid"},
			assertions: func(t *testing.T, result any, err error) {
				assert.ErrorContains(t, err, "invalid semantic version")
				assert.Nil(t, result)
			},
		},
		{
			name: "success",
			args: []any{"1.2.3"},
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				parsed, ok := result.(*semver.Version)
				require.True(t, ok)
				assert.NotNil(t, parsed)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := semverParse(testCase.args...)
			testCase.assertions(t, result, err)
		})
	}
}
