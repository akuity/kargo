package function

import (
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getCommitFromWarehouse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name:      "wrong number of args",
			warehouse: nil,
			args:      []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name:      "invalid arg type",
			warehouse: nil,
			args:      []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://example.com/repo.git",
							},
						},
					},
				},
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Git: []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "https://example.com/repo.git",
						Commits: []kargoapi.DiscoveredCommit{
							{
								Tag: "abc123",
								CreatorDate: &metav1.Time{
									Time: time.Date(2023, 9, 17, 1, 0, 0, 0, time.UTC),
								},
							},
							{
								Tag: "def456",
								CreatorDate: &metav1.Time{
									Time: time.Date(2023, 9, 17, 2, 0, 0, 0, time.UTC),
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
				commit, ok := result.(*kargoapi.GitCommit)
				require.True(t, ok)
				require.Equal(t, "def456", commit.Tag)
			},
		},
		{
			name:      "no commits found",
			warehouse: new(kargoapi.Warehouse),
			args:      []any{"https://example.com/repo.git"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "no commits found for repoURL")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
			require.NoError(t, err)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			fn := getCommitFromWarehouse(ctx, tc.warehouse, tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}

func Test_getImageFromWarehouse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name:      "wrong number of args",
			warehouse: nil,
			args:      []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name:      "invalid arg type",
			warehouse: nil,
			args:      []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Image: &kargoapi.ImageSubscription{
								RepoURL: "docker.io/example/repo",
							},
						},
					},
				},
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Images: []kargoapi.ImageDiscoveryResult{
					{
						RepoURL: "docker.io/example/repo",
						References: []kargoapi.DiscoveredImageReference{
							{
								Tag: "abc123",
								CreatedAt: &metav1.Time{
									Time: time.Date(2023, 9, 17, 1, 0, 0, 0, time.UTC),
								},
							},
							{
								Tag: "def456",
								CreatedAt: &metav1.Time{
									Time: time.Date(2023, 9, 17, 2, 0, 0, 0, time.UTC),
								},
							},
						},
					},
				},
			},
			args: []any{"docker.io/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				img, ok := result.(*kargoapi.Image)
				require.True(t, ok)
				require.Equal(t, "def456", img.Tag)
			},
		},
		{
			name:      "no images found",
			warehouse: new(kargoapi.Warehouse),
			args:      []any{"docker.io/example/repo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "no images found for repoURL")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
			require.NoError(t, err)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			fn := getImageFromWarehouse(ctx, tc.warehouse, tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}

func Test_getChartromWarehouse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		artifacts  *kargoapi.DiscoveredArtifacts
		args       []any
		assertions func(t *testing.T, result any, err error)
	}{
		{
			name:      "wrong number of args",
			warehouse: nil,
			args:      []any{"one", "two"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "expected 1 argument, got 2")
			},
		},
		{
			name:      "invalid arg type",
			warehouse: nil,
			args:      []any{1},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "first argument must be string, got int")
			},
		},
		{
			name: "success",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "oci://ghcr.io/akuity/kargo-charts/kargo",
							},
						},
					},
				},
			},
			artifacts: &kargoapi.DiscoveredArtifacts{
				Charts: []kargoapi.ChartDiscoveryResult{
					{
						RepoURL:  "oci://ghcr.io/akuity/kargo-charts/kargo",
						Versions: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
					},
					{
						RepoURL:  "oci://ghcr.io/akuity/kargo-charts/kargo",
						Versions: []string{"v2.1.0", "v2.2.0", "v2.3.0"},
					},
				},
			},
			args: []any{"oci://ghcr.io/akuity/kargo-charts/kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				commit, ok := result.(*kargoapi.Chart)
				require.True(t, ok)
				require.Equal(t, "2.3.0", commit.Version)
			},
		},
		{
			name:      "no charts found",
			warehouse: new(kargoapi.Warehouse),
			args:      []any{"oci://ghcr.io/akuity/kargo-charts/kargo"},
			assertions: func(t *testing.T, result any, err error) {
				require.Nil(t, result)
				require.ErrorContains(t, err, "no charts found for repoURL")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
			require.NoError(t, err)
			ctx := logging.ContextWithLogger(t.Context(), logger)
			fn := getChartFromWarehouse(ctx, tc.warehouse, tc.artifacts)
			result, err := fn(tc.args...)
			tc.assertions(t, result, err)
		})
	}
}
