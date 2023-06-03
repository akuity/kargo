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

func TestListEnvironmentsV1Alpha1(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.ListEnvironmentsRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.ListEnvironmentsRequest{
				Project: "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"existing project": {
			req: &svcv1alpha1.ListEnvironmentsRequest{
				Project: "kargo-demo",
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListEnvironmentsRequest{
				Project: "non-existing-project",
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
				).
				WithLists(&kubev1alpha1.EnvironmentList{
					Items: []kubev1alpha1.Environment{
						*mustNewObject[kubev1alpha1.Environment]("testdata/environment.yaml"),
					},
				}).
				Build()

			ctx := context.TODO()
			res, err := ListEnvironmentsV1Alpha1(kc)(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}
			require.Len(t, res.Msg.GetEnvironments(), 1)
		})
	}
}
