package secrets

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcile(t *testing.T) {
	const (
		testSrcNamespace  = "source-namespace"
		testDestNamespace = "destination-namespace"
		testSecretName    = "test-secret"
	)

	testCfg := ReconcilerConfig{
		SourceNamespace:      testSrcNamespace,
		DestinationNamespace: testDestNamespace,
	}

	testScheme := runtime.NewScheme()
	err := corev1.AddToScheme(testScheme)
	require.NoError(t, err)

	testSrcSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSrcNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("two-value"),
		},
	}

	testDestSecret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testDestNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("one-value"),
		},
	}

	testDestSecret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testDestNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("two-value"),
		},
	}

	testCases := []struct {
		name             string
		client           client.Client
		requestNamespace string
		assertions       func(t *testing.T, c client.Client, err error)
	}{
		{
			name:             "wrong namespace in request",
			client:           fake.NewClientBuilder().WithScheme(testScheme).Build(),
			requestNamespace: "wrong-namespace",
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error getting source",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithInterceptorFuncs(
					interceptor.Funcs{
						Get: func(
							context.Context,
							client.WithWatch,
							client.ObjectKey,
							client.Object,
							...client.GetOption,
						) error {
							return fmt.Errorf("something went wrong")
						},
					},
				).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "source does not exist, destination does; error deleting destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testDestSecret1).
				WithInterceptorFuncs(
					interceptor.Funcs{
						Delete: func(
							context.Context,
							client.WithWatch,
							client.Object,
							...client.DeleteOption,
						) error {
							return fmt.Errorf("something went wrong")
						},
					},
				).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "source does not exist, destination does; success deleting destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testDestSecret1).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "source and destination both exist; error patching destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testSrcSecret, testDestSecret1).
				WithInterceptorFuncs(
					interceptor.Funcs{
						Patch: func(
							context.Context,
							client.WithWatch,
							client.Object,
							client.Patch,
							...client.PatchOption,
						) error {
							return fmt.Errorf("something went wrong")
						},
					},
				).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "source and destination both exist; success patching destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testSrcSecret, testDestSecret1).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				require.Equal(t, testDestSecret2.Data, dest.Data)
			},
		},
		{
			name: "source exists, destination does not; error creating destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testSrcSecret).
				WithInterceptorFuncs(
					interceptor.Funcs{
						Create: func(
							context.Context,
							client.WithWatch,
							client.Object,
							...client.CreateOption,
						) error {
							return fmt.Errorf("something went wrong")
						},
					},
				).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "source exists, destination does not; success creating destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(testSrcSecret).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				require.Equal(t, testDestSecret2.Data, testSrcSecret.Data)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testCase.requestNamespace,
					Name:      testSecretName,
				},
			}
			if req.Namespace == "" {
				req.Namespace = testSrcNamespace
			}
			rec := newReconciler(testCase.client, testCfg)
			_, err := rec.Reconcile(t.Context(), req)
			testCase.assertions(t, testCase.client, err)
		})
	}
}
