package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	refreshTime, err := time.Parse(time.RFC3339, annotation)
	require.NoError(t, err)
	// Verify the timestamp is close to now
	// Assume it doesn't take 3 seconds to run this unit test.
	require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
	require.Equal(t, testNamespace, wh.Namespace)
	require.Equal(t, testName, wh.Name)
}
