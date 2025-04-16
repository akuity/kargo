package server

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestRefreshWarehouse(t *testing.T) {
	testSets := map[string]struct {
		req          *svcv1alpha1.RefreshWarehouseRequest
		errExpected  bool
		expectedCode connect.Code
	}{
		"empty project": {
			req: &svcv1alpha1.RefreshWarehouseRequest{
				Project: "",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"empty name": {
			req: &svcv1alpha1.RefreshWarehouseRequest{
				Project: "kargo-demo",
				Name:    "",
			},
			errExpected:  true,
			expectedCode: connect.CodeInvalidArgument,
		},
		"non-existing project": {
			req: &svcv1alpha1.RefreshWarehouseRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"non-existing warehouse": {
			req: &svcv1alpha1.RefreshWarehouseRequest{
				Project: "non-existing-project",
				Name:    "test",
			},
			errExpected:  true,
			expectedCode: connect.CodeNotFound,
		},
		"existing warehouse": {
			req: &svcv1alpha1.RefreshWarehouseRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[kargoapi.Stage]("testdata/stage.yaml"),
							).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			if !ts.errExpected {
				err = client.Create(ctx, &kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ts.req.GetProject(),
						Name:      ts.req.GetName(),
					},
					Spec: kargoapi.WarehouseSpec{},
				})
				require.NoError(t, err)
			}

			svr := &server{
				client: client,
			}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := svr.RefreshWarehouse(ctx, connect.NewRequest(ts.req))
			if ts.errExpected {
				require.Error(t, err)
				require.Equal(t, ts.expectedCode, connect.CodeOf(err))
				return
			}

			require.NoError(t, err)
			stage := res.Msg.GetWarehouse()
			annotation := stage.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
			refreshTime, err := time.Parse(time.RFC3339, annotation)
			require.NoError(t, err)
			// Make sure we set timestamp is close to now
			// Assume it doesn't take 3 seconds to run this unit test.
			require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
			require.Equal(t, ts.req.GetProject(), stage.Namespace)
			require.Equal(t, ts.req.GetName(), stage.Name)
		})
	}
}
