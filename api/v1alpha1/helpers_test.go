package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_patchAnnotation(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	t.Parallel()
	newFakeClient := func(obj ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj...).
			Build()
	}

	testCases := map[string]struct {
		obj         client.Object
		client      client.Client
		key         string
		value       string
		errExpected bool
	}{
		"stage": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			key:   "key",
			value: "value",
		},
		"stage with existing annotation": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			}),
			key:   "key",
			value: "value2",
		},
		"non-existing stage": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client:      newFakeClient(),
			key:         "key",
			value:       "value",
			errExpected: true,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := patchAnnotation(context.Background(), tc.client, tc.obj, tc.key, tc.value)
			if tc.errExpected {
				require.Error(t, err)
				return
			}
			require.Contains(t, tc.obj.GetAnnotations(), tc.key)
			require.Equal(t, tc.obj.GetAnnotations()[tc.key], tc.value)
		})
	}
}
