package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestRefreshObject(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))
	testName := "test-warehouse"
	testNamespace := "test-project"
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := t.Context()

	wh := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: kargoapi.WarehouseSpec{},
	}
	require.NoError(t, c.Create(ctx, wh))

	require.NoError(t, RefreshObject(ctx, c, wh))
	annotation := wh.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
	refreshTime, err := time.Parse(time.RFC3339Nano, annotation)
	require.NoError(t, err)
	// Verify the timestamp is close to now
	// Assume it doesn't take 3 seconds to run this unit test.
	require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
	require.Equal(t, testNamespace, wh.Namespace)
	require.Equal(t, testName, wh.Name)

	require.NoError(t, RefreshObject(ctx, c, wh))
	second := wh.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
	require.NotEmpty(t, second)
	require.NotEqual(t, annotation, second)

	stored := &kargoapi.Warehouse{}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(wh), stored))
	require.Equal(t, second, stored.GetAnnotations()[kargoapi.AnnotationKeyRefresh])
}

func TestRefreshHelpers(t *testing.T) {
	const project = "fake-project"
	nn := types.NamespacedName{Namespace: project, Name: "fake-name"}
	meta := metav1.ObjectMeta{Namespace: nn.Namespace, Name: nn.Name}

	tests := []struct {
		name    string
		obj     client.Object
		refresh func(context.Context, client.Client) (client.Object, error)
	}{
		{
			name: "Warehouse",
			obj:  &kargoapi.Warehouse{ObjectMeta: meta},
			refresh: func(ctx context.Context, c client.Client) (client.Object, error) {
				return RefreshWarehouse(ctx, c, nn)
			},
		},
		{
			name: "Stage",
			obj:  &kargoapi.Stage{ObjectMeta: meta},
			refresh: func(ctx context.Context, c client.Client) (client.Object, error) {
				return RefreshStage(ctx, c, nn)
			},
		},
		{
			name: "Promotion",
			obj:  &kargoapi.Promotion{ObjectMeta: meta},
			refresh: func(ctx context.Context, c client.Client) (client.Object, error) {
				return RefreshPromotion(ctx, c, nn)
			},
		},
		{
			name: "ProjectConfig",
			obj: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: project},
			},
			refresh: func(ctx context.Context, c client.Client) (client.Object, error) {
				return RefreshProjectConfig(ctx, c, project)
			},
		},
		{
			name: "ClusterConfig",
			obj: &kargoapi.ClusterConfig{
				ObjectMeta: metav1.ObjectMeta{Name: ClusterConfigName},
			},
			refresh: func(ctx context.Context, c client.Client) (client.Object, error) {
				return RefreshClusterConfig(ctx, c)
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			_, err := tt.refresh(ctx, fake.NewClientBuilder().WithScheme(scheme).Build())
			require.Error(t, err)

			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.obj).Build()
			first, err := tt.refresh(ctx, c)
			require.NoError(t, err)
			second, err := tt.refresh(ctx, c)
			require.NoError(t, err)

			firstVal := first.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
			secondVal := second.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
			require.NotEmpty(t, firstVal)
			require.NotEmpty(t, secondVal)
			require.NotEqual(t, firstVal, secondVal)

			stored := tt.obj.DeepCopyObject().(client.Object) // nolint: forcetypeassert
			require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(tt.obj), stored))
			require.Equal(t, secondVal, stored.GetAnnotations()[kargoapi.AnnotationKeyRefresh])
		})
	}
}
