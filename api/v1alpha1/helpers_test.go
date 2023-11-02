package v1alpha1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_refreshObject(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	t.Parallel()
	newFakeClient := func(obj ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj...).
			Build()
	}
	mockNow := func() time.Time {
		return time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC)
	}
	testCases := map[string]struct {
		obj         client.Object
		cli         client.Client
		nowFunc     func() time.Time
		errExpected bool
	}{
		"stage": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			cli: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			nowFunc: mockNow,
		},
		"stage with refresh annotation key": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			cli: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						AnnotationKeyRefresh: time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
					},
				},
			}),
			nowFunc: mockNow,
		},
		"non-existing stage": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			cli:         newFakeClient(),
			nowFunc:     mockNow,
			errExpected: true,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := refreshObject(context.Background(), tc.cli, tc.obj, tc.nowFunc)
			if tc.errExpected {
				require.Error(t, err)
				return
			}
			require.Contains(t, tc.obj.GetAnnotations(), AnnotationKeyRefresh)
			actual, err := time.Parse(time.RFC3339, tc.obj.GetAnnotations()[AnnotationKeyRefresh])
			require.NoError(t, err)
			require.Equal(t, tc.nowFunc(), actual)
		})
	}
}
