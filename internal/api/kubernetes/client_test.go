package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/internal/api/user"
)

func TestSetOptionsDefaults(t *testing.T) {
	opts, err := setOptionsDefaults(ClientOptions{})
	require.NoError(t, err)
	require.NotNil(t, opts.NewInternalClient)
	require.NotNil(t, opts.NewInternalDynamicClient)
	require.NotNil(t, opts.Scheme)
}

func TestNewClient(t *testing.T) {
	testInternalClient := fake.NewClientBuilder().Build()
	c, err := NewClient(
		context.Background(),
		&rest.Config{},
		ClientOptions{
			// Override this because the default behavior will fail without real REST
			// config.
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
			) (libClient.Client, error) {
				return testInternalClient, nil
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, c)
	client, ok := c.(*client)
	require.True(t, ok)
	require.Equal(t, testInternalClient, client.internalClient)
	require.NotNil(t, client.internalDynamicClient)
	require.NotNil(t, client.getAuthorizedClientFn)
}

func TestAllClientOperations(t *testing.T) {
	getOp := func(client *client) error {
		return client.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: "test-namespace",
				Name:      "test-name",
			},
			&corev1.Pod{},
		)
	}

	listOp := func(client *client) error {
		return client.List(
			context.Background(),
			&corev1.PodList{},
			libClient.InNamespace("test-namespace"),
		)
	}

	createOp := func(client *client) error {
		return client.Create(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
		)
	}

	deleteOp := func(client *client) error {
		return client.Delete(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
		)
	}

	updateOp := func(client *client) error {
		return client.Update(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
		)
	}

	patchOp := func(client *client) error {
		return client.Patch(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			libClient.MergeFrom(&corev1.Pod{}),
		)
	}

	deleteAllOp := func(client *client) error {
		return client.DeleteAllOf(
			context.Background(),
			&corev1.Pod{},
			libClient.InNamespace("test-namespace"),
		)
	}

	updateStatusOp := func(client *client) error {
		return client.Status().Update(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
		)
	}

	patchStatusOp := func(client *client) error {
		return client.Status().Patch(
			context.Background(),
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			libClient.MergeFrom(&corev1.Pod{}),
		)
	}

	watchOp := func(client *client) error {
		_, err := client.Watch(
			context.Background(),
			&corev1.Pod{},
			"test-namespace",
			metav1.ListOptions{},
		)
		return err
	}

	testCases := []struct {
		name       string
		op         func(client *client) error
		allowed    bool
		assertions func(t *testing.T, err error)
	}{
		{
			name: "get unauthorized",
			op:   getOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "get authorized",
			op:      getOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "list unauthorized",
			op:   listOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "list authorized",
			op:      listOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "create unauthorized",
			op:   createOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "create authorized",
			op:      createOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "delete unauthorized",
			op:   deleteOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "delete authorized",
			op:      deleteOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "update unauthorized",
			op:   updateOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "update authorized",
			op:      updateOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "patch unauthorized",
			op:   patchOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "patch authorized",
			op:      patchOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "delete all of unauthorized",
			op:   deleteAllOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "delete all of authorized",
			op:      deleteAllOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "update status unauthorized",
			op:   updateStatusOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "update status authorized",
			op:      updateStatusOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "patch status unauthorized",
			op:   patchStatusOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "patch status authorized",
			op:      patchStatusOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},

		{
			name: "watch unauthorized",
			op:   watchOp,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},

		{
			name:    "watch authorized",
			op:      watchOp,
			allowed: true,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c, err := NewClient(
				context.Background(),
				nil,
				ClientOptions{
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
					) (libClient.Client, error) {
						return fake.NewClientBuilder().Build(), nil
					},
					NewInternalDynamicClient: func(
						*rest.Config,
					) (dynamic.Interface, error) {
						return fakeDynamic.NewSimpleDynamicClient(runtime.NewScheme()), nil
					},
				},
			)
			require.NoError(t, err)
			client, ok := c.(*client)
			require.True(t, ok)
			if !testCase.allowed {
				client.getAuthorizedClientFn = func(
					context.Context,
					libClient.Client,
					string,
					schema.GroupVersionResource,
					string,
					libClient.ObjectKey,
				) (libClient.Client, error) {
					return nil, errors.New("not allowed")
				}
			} else {
				client.getAuthorizedClientFn = func(
					context.Context,
					libClient.Client,
					string,
					schema.GroupVersionResource,
					string,
					libClient.ObjectKey,
				) (libClient.Client, error) {
					return client.internalClient, nil
				}
			}
			testCase.assertions(t, testCase.op(client))
		})
	}
}

func TestGetAuthorizedClient(t *testing.T) {
	testInternalClient := fake.NewClientBuilder().Build()
	testCases := []struct {
		name     string
		userInfo *user.Info
		assert   func(*testing.T, libClient.Client, error)
	}{
		{
			name: "no context-bound user.Info",
			assert: func(t *testing.T, _ libClient.Client, err error) {
				require.Error(t, err)
				require.Equal(t, "not allowed", err.Error())
			},
		},
		{
			name: "admin user",
			userInfo: &user.Info{
				IsAdmin: true,
			},
			assert: func(t *testing.T, client libClient.Client, err error) {
				require.NoError(t, err)
				require.Same(t, testInternalClient, client)
			},
		},
		{
			name: "sso user",
			userInfo: &user.Info{
				Claims: map[string]any{
					"sub": "test-user",
				},
			},
			assert: func(t *testing.T, _ libClient.Client, err error) {
				require.True(t, kubeerr.IsForbidden(err))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.userInfo != nil {
				ctx := user.ContextWithInfo(context.Background(), *testCase.userInfo)
				client, err := getAuthorizedClient(nil)(
					ctx,
					testInternalClient,
					"", // Verb doesn't matter for these tests
					schema.GroupVersionResource{},
					"",                    // Subresource doesn't matter for these tests
					libClient.ObjectKey{}, // Object key doesn't matter for these tests
				)
				testCase.assert(t, client, err)
			}
		})
	}
}
