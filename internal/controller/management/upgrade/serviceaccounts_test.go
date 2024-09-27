package upgrade

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
)

func TestHasOldAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name:        "No old annotations",
			annotations: map[string]string{"some-key": "some-value"},
			expected:    false,
		},
		{
			name:        "Has old sub annotation",
			annotations: map[string]string{oldAnnotationKeySub: "some-value"},
			expected:    true,
		},
		{
			name:        "Has old email annotation",
			annotations: map[string]string{oldAnnotationKeyEmail: "some-value"},
			expected:    true,
		},
		{
			name:        "Has old groups annotation",
			annotations: map[string]string{oldAnnotationKeyGroups: "some-value"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOldAnnotations(tt.annotations)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceAccountReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	oldAnnotations := map[string]string{
		oldAnnotationKeySub:    "some-sub",
		oldAnnotationKeyEmail:  "some-email",
		oldAnnotationKeyGroups: "some-groups",
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-sa",
			Namespace:   "default",
			Annotations: oldAnnotations,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sa).Build()

	reconciler := &ServiceAccountReconciler{
		Client: client,
	}

	ctx := context.Background()
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-sa",
			Namespace: "default",
		},
	})
	assert.NoError(t, err)

	updatedSA := &corev1.ServiceAccount{}
	err = client.Get(ctx, types.NamespacedName{
		Name:      "test-sa",
		Namespace: "default",
	}, updatedSA)
	assert.NoError(t, err)

	assert.Equal(t, "some-sub", updatedSA.Annotations[rbacapi.AnnotationKeyOIDCClaim("sub")])
	assert.Equal(t, "some-email", updatedSA.Annotations[rbacapi.AnnotationKeyOIDCClaim("email")])
	assert.Equal(t, "some-groups", updatedSA.Annotations[rbacapi.AnnotationKeyOIDCClaim("groups")])

	assert.NotContains(t, updatedSA.Annotations, oldAnnotationKeySub)
	assert.NotContains(t, updatedSA.Annotations, oldAnnotationKeyEmail)
	assert.NotContains(t, updatedSA.Annotations, oldAnnotationKeyGroups)
}
