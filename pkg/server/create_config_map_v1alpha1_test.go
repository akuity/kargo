package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestCreateConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.CreateConfigMapRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.CreateConfigMapResponse], error)
	}{
		"nil config_map": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				ConfigMap: nil,
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"empty name": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cm",
						Namespace: "non-existing-project",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"create in project namespace": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cm",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"key": "value",
					},
				},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "test-cm", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-demo", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "value", r.Msg.GetConfigMap().Data["key"])
			},
		},
		"create in shared namespace": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "",
					},
					Data: map[string]string{
						"shared-key": "shared-value",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "shared-cm", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-shared-resources", r.Msg.GetConfigMap().Namespace)
			},
		},
		"create system-level": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				SystemLevel: true,
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "system-cm",
					},
					Data: map[string]string{
						"system-key": "system-value",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "system-cm", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-system-resources", r.Msg.GetConfigMap().Namespace)
			},
		},
	}

	for name, testCase := range testCases {
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
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
				cfg: config.ServerConfig{
					SharedResourcesNamespace: "kargo-shared-resources",
					SystemResourcesNamespace: "kargo-system-resources",
				},
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := svr.CreateConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
