package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateProject(t *testing.T) {
	const testPodName = "test-pod"
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	testCases := []struct {
		name        string
		objects     []client.Object
		targetObj   client.Object
		expectedErr func(error) bool
	}{
		{
			name: "success: project exists",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
						Labels: map[string]string{
							"kargo.akuity.io/project": "true",
						},
					},
				},
			},
			targetObj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      testPodName,
				},
			},
			expectedErr: nil,
		},
		{
			name:    "error: project not found (namespace missing)",
			objects: []client.Object{},
			targetObj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "missing-ns",
					Name:      testPodName,
				},
			},
			expectedErr: apierrors.IsNotFound,
		},
		{
			name: "error: namespace exists but is not a project (label missing)",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "other-ns",
					},
				},
			},
			targetObj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "other-ns",
					Name:      testPodName,
				},
			},
			expectedErr: func(err error) bool {
				var fieldErr *field.Error
				return errors.As(err, &fieldErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				Build()

			err := ValidateProject(context.Background(), k8sClient, tc.targetObj)

			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.True(t, tc.expectedErr(err), "unexpected error type: %v", err)
			}
		})
	}
}
