package secrets

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func TestComputeDataHash(t *testing.T) {
	testCases := []struct {
		name     string
		data     map[string][]byte
		expected string
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: computeDataHash(nil), // consistent hash for nil
		},
		{
			name:     "empty data",
			data:     map[string][]byte{},
			expected: computeDataHash(map[string][]byte{}),
		},
		{
			name: "single key",
			data: map[string][]byte{"key": []byte("value")},
		},
		{
			name: "multiple keys - order independent",
			data: map[string][]byte{
				"z-key": []byte("z-value"),
				"a-key": []byte("a-value"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := computeDataHash(tc.data)
			hash2 := computeDataHash(tc.data)
			// Hash should be deterministic
			require.Equal(t, hash1, hash2)
			// Hash should be 16 characters (truncated hex)
			require.Len(t, hash1, 16)
		})
	}

	// Test that different data produces different hashes
	t.Run("different data produces different hashes", func(t *testing.T) {
		hash1 := computeDataHash(map[string][]byte{"key": []byte("value1")})
		hash2 := computeDataHash(map[string][]byte{"key": []byte("value2")})
		require.NotEqual(t, hash1, hash2)
	})

	// Test order independence
	t.Run("key order does not affect hash", func(t *testing.T) {
		data1 := map[string][]byte{
			"a": []byte("1"),
			"b": []byte("2"),
			"c": []byte("3"),
		}
		data2 := map[string][]byte{
			"c": []byte("3"),
			"a": []byte("1"),
			"b": []byte("2"),
		}
		require.Equal(t, computeDataHash(data1), computeDataHash(data2))
	})
}

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

	// Helper to add finalizer to a secret
	withFinalizer := func(s *corev1.Secret) *corev1.Secret {
		secretCopy := s.DeepCopy()
		secretCopy.Finalizers = []string{kargoapi.FinalizerName}
		return secretCopy
	}

	// Helper to add hash annotation to a secret based on its data
	withHashAnnotation := func(s *corev1.Secret) *corev1.Secret {
		secretCopy := s.DeepCopy()
		if secretCopy.Annotations == nil {
			secretCopy.Annotations = make(map[string]string)
		}
		secretCopy.Annotations[syncedDataHashAnnotation] = computeDataHash(secretCopy.Data)
		return secretCopy
	}

	// Helper to add a specific hash annotation (for mismatch testing)
	withSpecificHash := func(s *corev1.Secret, hash string) *corev1.Secret {
		secretCopy := s.DeepCopy()
		if secretCopy.Annotations == nil {
			secretCopy.Annotations = make(map[string]string)
		}
		secretCopy.Annotations[syncedDataHashAnnotation] = hash
		return secretCopy
	}

	// Helper to mark a secret as being deleted
	withDeletionTimestamp := func(s *corev1.Secret) *corev1.Secret {
		secretCopy := s.DeepCopy()
		now := metav1.Now()
		secretCopy.DeletionTimestamp = &now
		return secretCopy
	}

	testSrcSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSrcNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("source-value"),
		},
	}

	testDestSecretDifferentData := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testDestNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("different-value"),
		},
	}

	testDestSecretMatchingData := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testDestNamespace,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("source-value"),
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
			name:   "source does not exist",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "source with no finalizer gets finalizer added",
			client: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(testSrcSecret).
				Build(),
			assertions: func(t *testing.T, c client.Client, _ error) {
				secret := &corev1.Secret{}
				getErr := c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, secret)
				require.NoError(t, getErr)
				require.Contains(t, secret.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "create: destination does not exist; creates with hash annotation",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(withFinalizer(testSrcSecret)).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				require.Equal(t, testSrcSecret.Data, dest.Data)
				// Verify hash annotation was added
				require.Contains(t, dest.Annotations, syncedDataHashAnnotation)
				require.Equal(t, computeDataHash(testSrcSecret.Data), dest.Annotations[syncedDataHashAnnotation])
				// Verify origin annotation was added
				require.Equal(t, testSrcNamespace, dest.Annotations[originNamespaceAnnotation])
			},
		},
		{
			name: "update: skipped when destination has no hash annotation",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withFinalizer(testSrcSecret),
					testDestSecretDifferentData, // no hash annotation
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Data should NOT have been updated
				require.Equal(t, testDestSecretDifferentData.Data, dest.Data)
			},
		},
		{
			name: "update: skipped when destination was modified externally (hash mismatch)",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withFinalizer(testSrcSecret),
					// Destination has hash annotation but data was modified
					withSpecificHash(testDestSecretDifferentData, "old-hash-value"),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Data should NOT have been updated
				require.Equal(t, testDestSecretDifferentData.Data, dest.Data)
			},
		},
		{
			name: "update: succeeds when hash matches",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withFinalizer(testSrcSecret),
					// Destination has matching hash (as if previously synced with old data)
					withHashAnnotation(testDestSecretDifferentData),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Data should have been updated to match source
				require.Equal(t, testSrcSecret.Data, dest.Data)
				// Hash annotation should be updated
				require.Equal(t, computeDataHash(testSrcSecret.Data), dest.Annotations[syncedDataHashAnnotation])
			},
		},
		{
			name: "update: error updating destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withFinalizer(testSrcSecret),
					withHashAnnotation(testDestSecretDifferentData),
				).
				WithInterceptorFuncs(
					interceptor.Funcs{
						Update: func(
							context.Context,
							client.WithWatch,
							client.Object,
							...client.UpdateOption,
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
			name: "delete: successful when hash matches",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
					withHashAnnotation(testDestSecretMatchingData),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// Destination Secret should be deleted
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.ErrorContains(t, err, "not found")
				// Source should be gone (finalizer removed allowing deletion)
				src := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, src)
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "delete: skipped when destination has no hash annotation",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
					testDestSecretMatchingData, // no hash annotation
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// Destination should still exist
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Source finalizer should still be removed
				src := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, src)
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "delete: skipped when destination was modified externally",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
					// Hash doesn't match current data
					withSpecificHash(testDestSecretMatchingData, "old-hash"),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// Destination should still exist
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Source finalizer should still be removed
				src := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, src)
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "delete: destination does not exist; just removes finalizer",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// Source should be gone (finalizer removed)
				src := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, src)
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "delete: error deleting destination",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
					withHashAnnotation(testDestSecretMatchingData),
				).
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
			name: "delete: preserves destination when source namespace is being deleted",
			client: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testSrcNamespace,
							DeletionTimestamp: &metav1.Time{
								Time: time.Now(),
							},
							Finalizers: []string{"kubernetes"}, // Required by fake client
						},
					},
					withDeletionTimestamp(withFinalizer(testSrcSecret)),
					withHashAnnotation(testDestSecretMatchingData),
				).Build(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// Destination should still exist (preserved)
				dest := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testDestNamespace,
					Name:      testSecretName,
				}, dest)
				require.NoError(t, err)
				// Source finalizer should be removed
				src := &corev1.Secret{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: testSrcNamespace,
					Name:      testSecretName,
				}, src)
				require.ErrorContains(t, err, "not found")
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
			l, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
			require.NoError(t, err)
			ctx := logging.ContextWithLogger(t.Context(), l)
			_, err = rec.Reconcile(ctx, req)
			testCase.assertions(t, testCase.client, err)
		})
	}
}
