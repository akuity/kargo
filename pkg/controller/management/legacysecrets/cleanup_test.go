package legacysecrets

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestRemoveOrphanedFinalizers(t *testing.T) {
	orphaned := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "orphaned",
			Namespace:  "kargo-cluster-secrets",
			Finalizers: []string{kargoapi.FinalizerName},
		},
	}
	noFinalizer := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-finalizer",
			Namespace: "kargo-cluster-secrets",
		},
	}
	activeReplicationSource := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "actively-managed",
			Namespace:  "kargo-shared-resources",
			Finalizers: []string{kargoapi.FinalizerName},
		},
	}
	otherFinalizer := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "unrelated-finalizer",
			Namespace:  "kargo-global-creds",
			Finalizers: []string{"some-other-controller/finalizer"},
		},
	}

	c := fake.NewClientBuilder().
		WithObjects(orphaned, noFinalizer, activeReplicationSource, otherFinalizer).
		Build()

	err := RemoveOrphanedFinalizers(
		t.Context(),
		c,
		c,
		"kargo-system-resources",
		"kargo-shared-resources",
	)
	require.NoError(t, err)

	assertFinalizers := func(t *testing.T, name, namespace string, want []string) {
		t.Helper()
		secret := &corev1.Secret{}
		require.NoError(t, c.Get(t.Context(), types.NamespacedName{Name: name, Namespace: namespace}, secret))
		require.Equal(t, want, secret.Finalizers)
	}

	assertFinalizers(t, "orphaned", "kargo-cluster-secrets", nil)
	assertFinalizers(t, "no-finalizer", "kargo-cluster-secrets", nil)
	assertFinalizers(t, "actively-managed", "kargo-shared-resources", []string{kargoapi.FinalizerName})
	assertFinalizers(t, "unrelated-finalizer", "kargo-global-creds", []string{"some-other-controller/finalizer"})
}

func TestRemoveOrphanedFinalizers_noSecrets(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	err := RemoveOrphanedFinalizers(t.Context(), c, c, "kargo-system-resources", "kargo-shared-resources")
	require.NoError(t, err)
}
