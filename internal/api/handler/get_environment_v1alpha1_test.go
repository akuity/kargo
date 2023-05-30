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

func TestGetEnvironmentV1Alpha1(t *testing.T) {
	testSets := map[string]struct {
		project      string
		name         string
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty project": {
			project:      "",
			name:         "",
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"empty name": {
			project:      "kargo-demo",
			name:         "",
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing environment": {
			project: "kargo-demo",
			name:    "test",
		},
		"non-existing project": {
			project:      "kargo-x",
			name:         "test",
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"non-existing environment": {
			project:      "non-existing-project",
			name:         "test",
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
				WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "kargo-demo",
						},
					},
					&testEnv,
				).
				Build()

			ctx := context.TODO()
			res, err := GetEnvironmentV1Alpha1(kc)(ctx,
				connect.NewRequest(&svcv1alpha1.GetEnvironmentRequest{
					Project: ts.project,
					Name:    ts.name,
				}))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.NotNil(t, res.Msg.GetEnvironment())
			require.Equal(t, ts.project, res.Msg.Environment.Metadata.Namespace)
			require.Equal(t, ts.name, res.Msg.Environment.Metadata.Name)
		})
	}
}
