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

func TestUpdateConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.UpdateConfigMapRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.UpdateConfigMapResponse], error)
	}{
		"nil config_map": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				ConfigMap: nil,
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"empty name": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cm",
						Namespace: "non-existing-project",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"update in project namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cm-1",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"foo": "updated-value",
					},
				},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "cm-1", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-demo", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "updated-value", r.Msg.GetConfigMap().Data["foo"])
			},
		},
		"update in shared namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "",
					},
					Data: map[string]string{
						"shared-key": "updated-shared-value",
					},
				},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "kargo-shared-resources",
					},
					Data: map[string]string{
						"shared-key": "original-value",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "shared-cm", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-shared-resources", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "updated-shared-value", r.Msg.GetConfigMap().Data["shared-key"])
			},
		},
		"update system-level": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				SystemLevel: true,
				ConfigMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "system-cm",
					},
					Data: map[string]string{
						"system-key": "updated-system-value",
					},
				},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "system-cm",
						Namespace: "kargo-system-resources",
					},
					Data: map[string]string{
						"system-key": "original-value",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.GetConfigMap())
				require.Equal(t, "system-cm", r.Msg.GetConfigMap().Name)
				require.Equal(t, "kargo-system-resources", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "updated-system-value", r.Msg.GetConfigMap().Data["system-key"])
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
			res, err := svr.UpdateConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
