package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestPromoteEnvironmentV1Alpha1(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.PromoteEnvironmentRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty state": {
			req: &svcv1alpha1.PromoteEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing environment": {
			req: &svcv1alpha1.PromoteEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "testx",
				State:   "73024971ee9c6daac0ad78aea87803bf332cfdb7",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing environment with non-existing state": {
			req: &svcv1alpha1.PromoteEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "test",
				State:   "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing environment": {
			req: &svcv1alpha1.PromoteEnvironmentRequest{
				Project: "kargo-demo",
				Name:    "test",
				State:   "73024971ee9c6daac0ad78aea87803bf332cfdb7",
			},
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
			res, err := PromoteEnvironmentV1Alpha1(kc)(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}

			require.True(t, strings.HasPrefix(res.Msg.GetPromotion().GetMetadata().GetName(), ts.req.GetName()+"-"))
			require.Equal(t, ts.req.GetName(), res.Msg.GetPromotion().GetSpec().GetEnvironment())
			require.Equal(t, ts.req.GetState(), res.Msg.GetPromotion().GetSpec().GetState())

			var actual kubev1alpha1.Promotion
			require.NoError(t, kc.Get(ctx, client.ObjectKey{
				Namespace: ts.req.GetProject(),
				Name:      res.Msg.GetPromotion().GetMetadata().GetName(),
			}, &actual))
		})
	}
}
