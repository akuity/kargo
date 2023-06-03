package handler

import (
	"context"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetEnvironmentV1Alpha1(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.GetEnvironmentRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.GetEnvironmentRequest{
				Project: "",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"empty name": {
			req: &svcv1alpha1.GetEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing environment": {
			req: &svcv1alpha1.GetEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetEnvironmentRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"non-existing environment": {
			req: &svcv1alpha1.GetEnvironmentRequest{
				Project: "non-existing-project",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kc := fake.NewClientBuilder().
				WithScheme(mustNewScheme()).
				WithObjects(
					mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
					mustNewObject[kubev1alpha1.Environment]("testdata/environment.yaml"),
				).
				Build()

			ctx := context.TODO()
			res, err := GetEnvironmentV1Alpha1(kc)(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.NotNil(t, res.Msg.GetEnvironment())
			require.Equal(t, ts.req.GetProject(), res.Msg.Environment.Metadata.Namespace)
			require.Equal(t, ts.req.GetName(), res.Msg.Environment.Metadata.Name)
		})
	}
}
