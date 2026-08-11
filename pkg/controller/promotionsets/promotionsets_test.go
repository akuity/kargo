package promotionsets

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestReconcile(t *testing.T) {
	t.Parallel()

	const (
		namespace = "fake-project"
		name      = "fake-promotion-set"
	)

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testCases := []struct {
		name string
		objs []client.Object
	}{
		{
			name: "existing PromotionSet",
			objs: []client.Object{&kargoapi.PromotionSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: kargoapi.PromotionSetSpec{
					Stage:   "fake-stage",
					Freight: "fake-freight",
					Targets: []kargoapi.PromotionSetTarget{{
						Name: "fake-target",
					}},
				},
			}},
		},
		{
			name: "PromotionSet not found",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := &reconciler{
				client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testCase.objs...).
					Build(),
			}

			result, err := r.Reconcile(t.Context(), ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: namespace,
					Name:      name,
				},
			})

			require.NoError(t, err)
			require.Empty(t, result)
		})
	}
}
