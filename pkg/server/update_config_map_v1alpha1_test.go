package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		"empty name": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "",
				Data: map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"empty data": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "test-cm",
				Data: map[string]string{},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"nil data": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "test-cm",
				Data: nil,
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "test-cm",
				Project: "non-existing-project",
				Data:    map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"update in project namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:        "cm-1",
				Project:     "kargo-demo",
				Data:        map[string]string{"updated": "data"},
				Description: "updated description",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "cm-1", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-demo", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated": "data"}, r.Msg.ConfigMap.Data)
				assert.Equal(t, "updated description", r.Msg.ConfigMap.Annotations[kargoapi.AnnotationKeyDescription])
			},
		},
		"update in shared namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "shared-cm",
				Project: "",
				Data:    map[string]string{"updated-shared": "data"},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "kargo-shared-resources",
					},
					Data: map[string]string{"old": "data"},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "shared-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-shared-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated-shared": "data"}, r.Msg.ConfigMap.Data)
			},
		},
		"update system-level": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				SystemLevel: true,
				Name:        "system-cm",
				Data:        map[string]string{"updated-system": "config"},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "system-cm",
						Namespace: "kargo-system-resources",
					},
					Data: map[string]string{"old-system": "config"},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "system-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-system-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated-system": "config"}, r.Msg.ConfigMap.Data)
			},
		},
		"update non-existing ConfigMap": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "non-existing-cm",
				Project: "kargo-demo",
				Data:    map[string]string{"new": "data"},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "update configmap")
				require.Nil(t, r)
			},
		},
		"update with multiple data keys": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
				Data: map[string]string{
					"newKey1": "newValue1",
					"newKey2": "newValue2",
					"newKey3": "newValue3",
				},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "cm-1", r.Msg.ConfigMap.Name)
				assert.Len(t, r.Msg.ConfigMap.Data, 3)
				assert.Equal(t, "newValue1", r.Msg.ConfigMap.Data["newKey1"])
				assert.Equal(t, "newValue2", r.Msg.ConfigMap.Data["newKey2"])
				assert.Equal(t, "newValue3", r.Msg.ConfigMap.Data["newKey3"])
			},
		},
		"update clears old data and sets new": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "multi-key-cm",
				Project: "kargo-demo",
				Data:    map[string]string{"onlyKey": "onlyValue"},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-key-cm",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"oldKey1": "oldValue1",
						"oldKey2": "oldValue2",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "multi-key-cm", r.Msg.ConfigMap.Name)
				assert.Len(t, r.Msg.ConfigMap.Data, 1)
				assert.Equal(t, "onlyValue", r.Msg.ConfigMap.Data["onlyKey"])
				// Verify old keys are not present
				_, hasOldKey1 := r.Msg.ConfigMap.Data["oldKey1"]
				_, hasOldKey2 := r.Msg.ConfigMap.Data["oldKey2"]
				assert.False(t, hasOldKey1)
				assert.False(t, hasOldKey2)
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
