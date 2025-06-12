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

func TestRefreshProjectConfig(t *testing.T) {
	testSets := map[string]struct {
		req        *svcv1alpha1.RefreshProjectConfigRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.RefreshProjectConfigResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.RefreshProjectConfigRequest{
				Project: "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.RefreshProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.RefreshProjectConfigRequest{
				Project: "kargo-x",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.RefreshProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing ProjectConfig": {
			req: &svcv1alpha1.RefreshProjectConfigRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.RefreshProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ProjectConfig": {
			req: &svcv1alpha1.RefreshProjectConfigRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.ProjectConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProjectConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.RefreshProjectConfigResponse], err error) {
				require.NoError(t, err)

				config := r.Msg.GetProjectConfig()
				require.Equal(t, "kargo-demo", config.Name)

				annotation := config.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)

				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)

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
							WithObjects(ts.objects...).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client:                    client,
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := svr.RefreshProjectConfig(ctx, connect.NewRequest(ts.req))
			ts.assertions(t, res, err)
		})
	}
}
