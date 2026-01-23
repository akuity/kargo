package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

func TestCreateConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.CreateConfigMapRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.CreateConfigMapResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name: "",
				Data: map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"empty data": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name: "test-cm",
				Data: map[string]string{},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"nil data": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name: "test-cm",
				Data: nil,
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name:    "test-cm",
				Project: "non-existing-project",
				Data:    map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"create in project namespace": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name:        "new-cm",
				Project:     "kargo-demo",
				Data:        map[string]string{"foo": "bar"},
				Description: "test description",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "new-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-demo", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"foo": "bar"}, r.Msg.ConfigMap.Data)
				assert.Equal(t, "test description", r.Msg.ConfigMap.Annotations[kargoapi.AnnotationKeyDescription])
			},
		},
		"create in shared namespace": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name:    "shared-cm",
				Project: "",
				Data:    map[string]string{"shared": "data"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "shared-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-shared-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"shared": "data"}, r.Msg.ConfigMap.Data)
			},
		},
		"create system-level": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				SystemLevel: true,
				Name:        "system-cm",
				Data:        map[string]string{"system": "config"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "system-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-system-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"system": "config"}, r.Msg.ConfigMap.Data)
			},
		},
		"create already existing ConfigMap": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
				Data:    map[string]string{"new": "data"},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "already exists")
				require.Nil(t, r)
			},
		},
		"create with multiple data keys": {
			req: &svcv1alpha1.CreateConfigMapRequest{
				Name:    "multi-key-cm",
				Project: "kargo-demo",
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.CreateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "multi-key-cm", r.Msg.ConfigMap.Name)
				assert.Len(t, r.Msg.ConfigMap.Data, 3)
				assert.Equal(t, "value1", r.Msg.ConfigMap.Data["key1"])
				assert.Equal(t, "value2", r.Msg.ConfigMap.Data["key2"])
				assert.Equal(t, "value3", r.Msg.ConfigMap.Data["key3"])
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

func TestConfigMapToK8sConfigMap(t *testing.T) {
	testCases := map[string]struct {
		input    configMap
		cfg      config.ServerConfig
		expected *corev1.ConfigMap
	}{
		"system level": {
			input: configMap{
				systemLevel: true,
				project:     "ignored",
				name:        "system-cm",
				description: "system description",
				data:        map[string]string{"key": "value"},
			},
			cfg: config.ServerConfig{
				SystemResourcesNamespace: "kargo-system",
				SharedResourcesNamespace: "kargo-shared",
			},
			expected: &corev1.ConfigMap{
				Data: map[string]string{"key": "value"},
			},
		},
		"project level": {
			input: configMap{
				systemLevel: false,
				project:     "my-project",
				name:        "project-cm",
				description: "project description",
				data:        map[string]string{"key": "value"},
			},
			cfg: config.ServerConfig{
				SystemResourcesNamespace: "kargo-system",
				SharedResourcesNamespace: "kargo-shared",
			},
			expected: &corev1.ConfigMap{
				Data: map[string]string{"key": "value"},
			},
		},
		"shared level (empty project)": {
			input: configMap{
				systemLevel: false,
				project:     "",
				name:        "shared-cm",
				description: "shared description",
				data:        map[string]string{"key": "value"},
			},
			cfg: config.ServerConfig{
				SystemResourcesNamespace: "kargo-system",
				SharedResourcesNamespace: "kargo-shared",
			},
			expected: &corev1.ConfigMap{
				Data: map[string]string{"key": "value"},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			svr := &server{cfg: testCase.cfg}
			result := svr.configMapToK8sConfigMap(testCase.input)

			assert.Equal(t, testCase.input.name, result.Name)
			assert.Equal(t, testCase.input.description, result.Annotations[kargoapi.AnnotationKeyDescription])
			assert.Equal(t, testCase.input.data, result.Data)

			// Verify namespace
			if testCase.input.systemLevel {
				assert.Equal(t, testCase.cfg.SystemResourcesNamespace, result.Namespace)
			} else if testCase.input.project != "" {
				assert.Equal(t, testCase.input.project, result.Namespace)
			} else {
				assert.Equal(t, testCase.cfg.SharedResourcesNamespace, result.Namespace)
			}
		})
	}
}
