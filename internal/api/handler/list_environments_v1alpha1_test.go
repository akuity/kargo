package handler

import (
	"context"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestListEnvironmentsV1Alpha1(t *testing.T) {
	testSets := map[string]struct {
		namespace    string
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty namespace": {
			namespace:    "",
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing namespace": {
			namespace: "kargo-demo",
		},
		"non-existing namespace": {
			namespace:    "non-existing-namespace",
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			scheme := k8sruntime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, kubev1alpha1.AddToScheme(scheme))

			rawEnv, err := testData.ReadFile("testdata/environment.yaml")
			require.NoError(t, err)

			var testEnv kubev1alpha1.Environment
			require.NoError(t, yaml.Unmarshal(rawEnv, &testEnv))

			kc := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
					},
				}).
				WithLists(&kubev1alpha1.EnvironmentList{
					Items: []kubev1alpha1.Environment{
						testEnv,
					},
				}).Build()

			ctx := context.TODO()
			req := connect.NewRequest(&svcv1alpha1.ListEnvironmentsRequest{
				Namespace: ts.namespace,
			})
			res, err := ListEnvironmentsV1Alpha1(kc)(ctx, req)
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.Len(t, res.Msg.GetEnvironments(), 1)
		})
	}
}
