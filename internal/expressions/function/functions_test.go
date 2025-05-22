package function

import (
	"context"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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

func Test_getCommit(t *testing.T) {
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
						Subscriptions: []kargoapi.RepoSubscription{
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

			fn := getCommit(
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

func Test_getImage(t *testing.T) {
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
						Subscriptions: []kargoapi.RepoSubscription{
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

			fn := getImage(
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

func Test_getChart(t *testing.T) {
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
						Subscriptions: []kargoapi.RepoSubscription{
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
						Subscriptions: []kargoapi.RepoSubscription{
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

			fn := getChart(
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
			name: "success",
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
			name: "success with cache",
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
			name: "success from cache",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			fn := getSecret(ctx, c, tt.cache, testProject)

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

			secretFn := getSecret(ctx, c, tt.cache, testProject)
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
