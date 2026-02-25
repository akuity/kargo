package sharedsecrets

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	testSourceNS   = "kargo-shared-resources"
	testProject1   = "project-alpha"
	testProject2   = "project-beta"
	testSecretName = "my-shared-secret"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, kargoapi.AddToScheme(s))
	return s
}

// srcSecret builds a base source Secret in the shared resources namespace.
func srcSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSourceNS,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"key": []byte("value")},
	}
}

// withReplicateTo adds the replicate-to: "*" annotation.
func withReplicateTo(s *corev1.Secret) *corev1.Secret {
	c := s.DeepCopy()
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	c.Annotations[kargoapi.AnnotationKeyReplicateTo] = kargoapi.AnnotationValueReplicateToAll
	return c
}

// withFinalizer adds FinalizerNameReplicated to the secret.
func withFinalizer(s *corev1.Secret) *corev1.Secret {
	c := s.DeepCopy()
	controllerutil.AddFinalizer(c, kargoapi.FinalizerNameReplicated)
	return c
}

// withDeletionTimestamp marks the secret as being deleted.
func withDeletionTimestamp(s *corev1.Secret) *corev1.Secret {
	c := s.DeepCopy()
	now := metav1.NewTime(time.Now())
	c.DeletionTimestamp = &now
	return c
}

// project builds a Project resource.
func project(name string) *kargoapi.Project {
	return &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// replicatedSecret builds a replicated Secret in the given namespace whose
// SHA label matches what the reconciler would compute for a source with the
// given data (and no labels/annotations).
func replicatedSecret(namespace string, data map[string][]byte) *corev1.Secret {
	src := &corev1.Secret{Data: data}
	hash := computeDataHash(src)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      testSecretName,
			Labels: map[string]string{
				kargoapi.LabelKeyReplicatedFrom: testSecretName,
				kargoapi.LabelKeyReplicatedSHA:  hash,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

// reconcilerForTest builds a reconciler backed by a fake client.
// The same fake client is used for both r.client and r.apiReader since fake
// clients support all operations regardless of namespace.
func reconcilerForTest(fc client.Client) *reconciler {
	return &reconciler{
		cfg: ReconcilerConfig{
			SharedResourcesNamespace: testSourceNS,
			MaxConcurrentReconciles:  4,
		},
		client:    fc,
		apiReader: fc,
	}
}

// doReconcile is a convenience wrapper.
func doReconcile(t *testing.T, r *reconciler) (ctrl.Result, error) {
	t.Helper()
	l, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
	require.NoError(t, err)
	ctx := logging.ContextWithLogger(t.Context(), l)
	return r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: testSourceNS,
			Name:      testSecretName,
		},
	})
}

func TestReconcile_SourceNotFound(t *testing.T) {
	fc := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()
	r := reconcilerForTest(fc)
	_, err := doReconcile(t, r)
	require.NoError(t, err)
}

func TestReconcile_NoAnnotationNoFinalizer(t *testing.T) {
	// Secret exists but has no replicate-to annotation and no finalizer — no-op.
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(srcSecret()).
		Build()
	r := reconcilerForTest(fc)
	result, err := doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)

	// Verify no additional secrets were created.
	secretList := &corev1.SecretList{}
	require.NoError(t, fc.List(t.Context(), secretList))
	require.Len(t, secretList.Items, 1) // only the source
}

func TestReconcile_AnnotationPresent_NoProjects_AddsFinalizerAndRequeues(t *testing.T) {
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(withReplicateTo(srcSecret())).
		Build()
	r := reconcilerForTest(fc)

	// First reconcile: should add finalizer and requeue.
	result, err := doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, 100*time.Millisecond, result.RequeueAfter)

	// Verify finalizer was added.
	src := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testSourceNS, Name: testSecretName,
	}, src))
	require.True(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerNameReplicated))

	// Second reconcile (finalizer already present, no projects): no replicated secrets.
	result, err = doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)

	secretList := &corev1.SecretList{}
	require.NoError(t, fc.List(t.Context(), secretList, client.InNamespace(testProject1)))
	require.Empty(t, secretList.Items)
}

func TestReconcile_AnnotationPresent_TwoProjects_CreatesReplicas(t *testing.T) {
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withFinalizer(withReplicateTo(srcSecret())),
			project(testProject1),
			project(testProject2),
		).
		Build()
	r := reconcilerForTest(fc)

	result, err := doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)

	sourceHash := computeDataHash(srcSecret())

	for _, ns := range []string{testProject1, testProject2} {
		dest := &corev1.Secret{}
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: ns, Name: testSecretName,
		}, dest), "expected replicated secret in namespace %q", ns)
		require.Equal(t, srcSecret().Data, dest.Data)
		require.Equal(t, srcSecret().Type, dest.Type)
		require.Equal(t, testSecretName, dest.Labels[kargoapi.LabelKeyReplicatedFrom])
		require.Equal(t, sourceHash, dest.Labels[kargoapi.LabelKeyReplicatedSHA])
	}
}

func TestReconcile_LabelsAndAnnotationsCarriedOver(t *testing.T) {
	// Source has extra labels and annotations that should be copied to replicas,
	// minus the excluded ones.
	src := withFinalizer(withReplicateTo(srcSecret()))
	src.Labels = map[string]string{"team": "infra"}
	src.Annotations = map[string]string{
		kargoapi.AnnotationKeyReplicateTo: "*",
		lastAppliedConfigAnnotation:       `{"big":"json"}`,
		"custom.io/owner":                 "ops",
	}

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(src, project(testProject1)).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	dest := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest))

	// Source labels merged with replication labels.
	require.Equal(t, "infra", dest.Labels["team"])
	require.Equal(t, testSecretName, dest.Labels[kargoapi.LabelKeyReplicatedFrom])
	require.NotEmpty(t, dest.Labels[kargoapi.LabelKeyReplicatedSHA])

	// Custom annotation carried over; excluded ones stripped.
	require.Equal(t, "ops", dest.Annotations["custom.io/owner"])
	require.NotContains(t, dest.Annotations, kargoapi.AnnotationKeyReplicateTo)
	require.NotContains(t, dest.Annotations, lastAppliedConfigAnnotation)
}

func TestReconcile_AlreadyUpToDate_NoUpdate(t *testing.T) {
	sourceData := srcSecret().Data
	updateCalled := false

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withFinalizer(withReplicateTo(srcSecret())),
			project(testProject1),
			replicatedSecret(testProject1, sourceData),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(
				_ context.Context,
				_ client.WithWatch,
				_ client.Object,
				_ ...client.UpdateOption,
			) error {
				updateCalled = true
				return fmt.Errorf("update should not have been called")
			},
		}).Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)
	require.False(t, updateCalled, "expected no update when replica is already up to date")
}

func TestReconcile_SourceUpdated_UpdatesReplica(t *testing.T) {
	oldData := map[string][]byte{"key": []byte("old-value")}
	newData := map[string][]byte{"key": []byte("new-value")}
	newHash := computeDataHash(&corev1.Secret{Data: newData})

	updatedSrc := withFinalizer(withReplicateTo(srcSecret()))
	updatedSrc.Data = newData

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			updatedSrc,
			project(testProject1),
			replicatedSecret(testProject1, oldData),
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	dest := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest))
	require.Equal(t, newData, dest.Data)
	require.Equal(t, newHash, dest.Labels[kargoapi.LabelKeyReplicatedSHA])
}

func TestReconcile_ExternallyModified_Skipped(t *testing.T) {
	originalData := srcSecret().Data
	modifiedData := map[string][]byte{"key": []byte("modified-externally")}

	// SHA label records the hash of the original source secret's content, but
	// the data was subsequently changed externally.
	existingReplica := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject1,
			Name:      testSecretName,
			Labels: map[string]string{
				kargoapi.LabelKeyReplicatedFrom: testSecretName,
				kargoapi.LabelKeyReplicatedSHA:  computeDataHash(&corev1.Secret{Data: originalData}),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: modifiedData,
	}

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withFinalizer(withReplicateTo(srcSecret())),
			project(testProject1),
			existingReplica,
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	// Data should NOT have been reverted.
	dest := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest))
	require.Equal(t, modifiedData, dest.Data)
}

func TestReconcile_NoReplicatedFromLabel_Conflict_Skipped(t *testing.T) {
	// A user-created secret with the same name — no replicated-from label.
	userSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject1,
			Name:      testSecretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"user-key": []byte("user-value")},
	}

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withFinalizer(withReplicateTo(srcSecret())),
			project(testProject1),
			userSecret,
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	// User's secret data should be preserved.
	dest := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest))
	require.Equal(t, userSecret.Data, dest.Data)
}

func TestReconcile_DeletionTimestamp_CleansUpAndRemovesFinalizer(t *testing.T) {
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withDeletionTimestamp(withFinalizer(withReplicateTo(srcSecret()))),
			replicatedSecret(testProject1, srcSecret().Data),
			replicatedSecret(testProject2, srcSecret().Data),
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	// Replicated secrets should be deleted.
	for _, ns := range []string{testProject1, testProject2} {
		dest := &corev1.Secret{}
		getErr := fc.Get(t.Context(), types.NamespacedName{
			Namespace: ns, Name: testSecretName,
		}, dest)
		require.True(t, apierrors.IsNotFound(getErr), "expected replicated secret in %q to be deleted", ns)
	}

	// Source secret should have its finalizer removed (fake client deletes it
	// since there are no other finalizers).
	src := &corev1.Secret{}
	getErr := fc.Get(t.Context(), types.NamespacedName{
		Namespace: testSourceNS, Name: testSecretName,
	}, src)
	require.True(t, apierrors.IsNotFound(getErr))
}

func TestReconcile_AnnotationRemoved_CleansUpAndRemovesFinalizer(t *testing.T) {
	// Source secret has finalizer but annotation was removed.
	srcWithoutAnnotation := withFinalizer(srcSecret()) // no replicate-to annotation

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			srcWithoutAnnotation,
			replicatedSecret(testProject1, srcSecret().Data),
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	// Replicated secret should be deleted.
	dest := &corev1.Secret{}
	getErr := fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest)
	require.True(t, apierrors.IsNotFound(getErr))

	// Source should have its replication finalizer removed.
	src := &corev1.Secret{}
	require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
		Namespace: testSourceNS, Name: testSecretName,
	}, src))
	require.False(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerNameReplicated))
}

func TestReconcile_DeletionTimestamp_DeletesAllReplicas(t *testing.T) {
	// Cleanup deletes all replicas regardless of external modification.
	modifiedData := map[string][]byte{"key": []byte("modified-externally")}
	modifiedReplica := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject1,
			Name:      testSecretName,
			Labels: map[string]string{
				kargoapi.LabelKeyReplicatedFrom: testSecretName,
				kargoapi.LabelKeyReplicatedSHA:  computeDataHash(srcSecret()),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: modifiedData,
	}

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withDeletionTimestamp(withFinalizer(withReplicateTo(srcSecret()))),
			modifiedReplica,
		).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.NoError(t, err)

	// All replicas are deleted unconditionally during cleanup.
	dest := &corev1.Secret{}
	getErr := fc.Get(t.Context(), types.NamespacedName{
		Namespace: testProject1, Name: testSecretName,
	}, dest)
	require.True(t, apierrors.IsNotFound(getErr))
}


func TestReconcile_CreateError(t *testing.T) {
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withFinalizer(withReplicateTo(srcSecret())),
			project(testProject1),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(
				_ context.Context,
				_ client.WithWatch,
				obj client.Object,
				_ ...client.CreateOption,
			) error {
				if obj.GetNamespace() == testProject1 {
					return fmt.Errorf("something went wrong")
				}
				return nil
			},
		}).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.ErrorContains(t, err, "something went wrong")
}

func TestReconcile_UpdateError(t *testing.T) {
	oldData := map[string][]byte{"key": []byte("old-value")}
	newData := map[string][]byte{"key": []byte("new-value")}

	updatedSrc := withFinalizer(withReplicateTo(srcSecret()))
	updatedSrc.Data = newData

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			updatedSrc,
			project(testProject1),
			replicatedSecret(testProject1, oldData),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(
				_ context.Context,
				_ client.WithWatch,
				_ client.Object,
				_ ...client.UpdateOption,
			) error {
				return fmt.Errorf("something went wrong")
			},
		}).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.ErrorContains(t, err, "something went wrong")
}

func TestReconcile_DeleteError_DuringCleanup(t *testing.T) {
	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(
			withDeletionTimestamp(withFinalizer(withReplicateTo(srcSecret()))),
			replicatedSecret(testProject1, srcSecret().Data),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(
				_ context.Context,
				_ client.WithWatch,
				_ client.Object,
				_ ...client.DeleteOption,
			) error {
				return fmt.Errorf("something went wrong")
			},
		}).
		Build()
	r := reconcilerForTest(fc)

	_, err := doReconcile(t, r)
	require.ErrorContains(t, err, "something went wrong")
}

func TestProjectCreatedEnqueuer(t *testing.T) {
	annotatedSecret := withReplicateTo(srcSecret())
	unannotatedSecret := srcSecret()
	unannotatedSecret.Name = "unannotated"

	fc := fake.NewClientBuilder().
		WithScheme(testScheme(t)).
		WithObjects(annotatedSecret, unannotatedSecret).
		Build()

	enqueuer := &projectCreatedEnqueuer{
		client:   fc,
		sourceNS: testSourceNS,
	}

	wq := &fakeWorkQueue{}
	l, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
	require.NoError(t, err)
	ctx := logging.ContextWithLogger(t.Context(), l)

	enqueuer.Create(
		ctx,
		event.TypedCreateEvent[*kargoapi.Project]{
			Object: project(testProject1),
		},
		wq,
	)

	require.Len(t, wq.items, 1)
	require.Equal(t, testSecretName, wq.items[0].Name)
	require.Equal(t, testSourceNS, wq.items[0].Namespace)
}

func TestComputeDataHash(t *testing.T) {
	secret := func(labels, annotations map[string]string, data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Data:       data,
		}
	}

	t.Run("deterministic", func(t *testing.T) {
		s := secret(nil, nil, map[string][]byte{"k": []byte("v")})
		require.Equal(t, computeDataHash(s), computeDataHash(s))
		require.Len(t, computeDataHash(s), 16)
	})

	t.Run("data key order independent", func(t *testing.T) {
		h1 := computeDataHash(secret(nil, nil, map[string][]byte{"a": []byte("1"), "b": []byte("2")}))
		h2 := computeDataHash(secret(nil, nil, map[string][]byte{"b": []byte("2"), "a": []byte("1")}))
		require.Equal(t, h1, h2)
	})

	t.Run("different data produces different hashes", func(t *testing.T) {
		h1 := computeDataHash(secret(nil, nil, map[string][]byte{"k": []byte("v1")}))
		h2 := computeDataHash(secret(nil, nil, map[string][]byte{"k": []byte("v2")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("empty secret", func(t *testing.T) {
		h := computeDataHash(&corev1.Secret{})
		require.Len(t, h, 16)
	})

	t.Run("label change produces different hash", func(t *testing.T) {
		h1 := computeDataHash(secret(map[string]string{"env": "prod"}, nil, nil))
		h2 := computeDataHash(secret(map[string]string{"env": "staging"}, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("annotation change produces different hash", func(t *testing.T) {
		h1 := computeDataHash(secret(nil, map[string]string{"owner": "team-a"}, nil))
		h2 := computeDataHash(secret(nil, map[string]string{"owner": "team-b"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("replicate-to annotation excluded from hash", func(t *testing.T) {
		h1 := computeDataHash(secret(nil, map[string]string{kargoapi.AnnotationKeyReplicateTo: "*"}, nil))
		h2 := computeDataHash(secret(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("last-applied-configuration annotation excluded from hash", func(t *testing.T) {
		h1 := computeDataHash(secret(nil, map[string]string{lastAppliedConfigAnnotation: `{"big":"json"}`}, nil))
		h2 := computeDataHash(secret(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("replication labels excluded from hash", func(t *testing.T) {
		h1 := computeDataHash(secret(map[string]string{
			kargoapi.LabelKeyReplicatedFrom: "src",
			kargoapi.LabelKeyReplicatedSHA:  "abc123",
		}, nil, nil))
		h2 := computeDataHash(secret(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("label order independent", func(t *testing.T) {
		h1 := computeDataHash(secret(map[string]string{"a": "1", "b": "2"}, nil, nil))
		h2 := computeDataHash(secret(map[string]string{"b": "2", "a": "1"}, nil, nil))
		require.Equal(t, h1, h2)
	})
}

// ---- test helpers ----

// fakeWorkQueue is a minimal TypedRateLimitingInterface[reconcile.Request]
// implementation that records enqueued items.
type fakeWorkQueue struct {
	items []reconcile.Request
}

var _ workqueue.TypedRateLimitingInterface[reconcile.Request] = &fakeWorkQueue{}

func (q *fakeWorkQueue) Add(item reconcile.Request) {
	q.items = append(q.items, item)
}
func (q *fakeWorkQueue) AddAfter(item reconcile.Request, _ time.Duration) {
	q.items = append(q.items, item)
}
func (q *fakeWorkQueue) AddRateLimited(item reconcile.Request) {
	q.items = append(q.items, item)
}
func (q *fakeWorkQueue) Forget(_ reconcile.Request)          {}
func (q *fakeWorkQueue) NumRequeues(_ reconcile.Request) int { return 0 }
func (q *fakeWorkQueue) Done(_ reconcile.Request)            {}
func (q *fakeWorkQueue) Get() (reconcile.Request, bool)      { return reconcile.Request{}, false }
func (q *fakeWorkQueue) Len() int                            { return len(q.items) }
func (q *fakeWorkQueue) ShutDown()                           {}
func (q *fakeWorkQueue) ShutDownWithDrain()                  {}
func (q *fakeWorkQueue) ShuttingDown() bool                  { return false }
