package replication

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
	testSourceNS     = "kargo-shared-resources"
	testProject1     = "project-alpha"
	testProject2     = "project-beta"
	testResourceName = "my-shared-resource"
)

// ---- Shared test helpers ----

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, kargoapi.AddToScheme(s))
	return s
}

func project(name string) *kargoapi.Project {
	return &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

// withAnnotation adds the replicate-to: "*" annotation to a deep copy.
func withAnnotation(obj client.Object) client.Object {
	cp, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		panic("failed to cast DeepCopyObject to client.Object")
	}
	anns := cp.GetAnnotations()
	if anns == nil {
		anns = make(map[string]string)
	}
	anns[kargoapi.AnnotationKeyReplicateTo] = kargoapi.AnnotationValueReplicateToAll
	cp.SetAnnotations(anns)
	return cp
}

// withFinalizer adds FinalizerNameReplicated to a deep copy.
func withFinalizer(obj client.Object) client.Object {
	cp, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		panic("failed to cast DeepCopyObject to client.Object")
	}
	controllerutil.AddFinalizer(cp, kargoapi.FinalizerName)
	return cp
}

func reconcilerForTest(fc client.Client, f reconcilerTestFixture) *reconciler {
	return &reconciler{
		cfg: ReconcilerConfig{
			SharedResourcesNamespace: testSourceNS,
			MaxConcurrentReconciles:  4,
		},
		client:    fc,
		apiReader: fc,
		adapter:   f.adapter,
	}
}

func doReconcile(t *testing.T, r *reconciler) (ctrl.Result, error) {
	t.Helper()
	l, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
	require.NoError(t, err)
	ctx := logging.ContextWithLogger(t.Context(), l)
	return r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: testSourceNS,
			Name:      testResourceName,
		},
	})
}

// ---- Fixture definition ----

// reconcilerTestFixture holds adapter-specific constructors so that
// runReconcilerTests can exercise the same logic for both Secrets and
// ConfigMaps.
type reconcilerTestFixture struct {
	adapter resourceAdapter

	// newSrc returns a base source object with default data (no
	// annotations/finalizers).
	newSrc func() client.Object

	// withDeletion returns a deep copy of the object with DeletionTimestamp
	// set. DeletionTimestamp cannot be set via the client.Object interface, so
	// each fixture provides a type-specific implementation.
	withDeletion func(client.Object) client.Object

	// newReplica returns an up-to-date replica in ns whose SHA label matches
	// the hash of src.
	newReplica func(ns string, src client.Object) client.Object

	// withUpdatedData returns a deep copy of the source object with modified
	// data (used to simulate a source update).
	withUpdatedData func(client.Object) client.Object

	// newExternallyModifiedReplica returns a replica whose LabelKeyReplicatedSHA
	// equals the source hash but whose actual content was changed externally.
	newExternallyModifiedReplica func(ns string, src client.Object) client.Object

	// newConflictingResource returns a resource with the same name but no
	// LabelKeyReplicatedFrom label (user-created conflict).
	newConflictingResource func(ns string) client.Object

	// checkReplica asserts that the replica's type-specific fields match those
	// of expectedSrc.
	checkReplica func(t *testing.T, replica, expectedSrc client.Object)

	// computeHash returns the hash for the given source object.
	computeHash func(client.Object) string
}

// ---- Fixture implementations ----

func secretFixture() reconcilerTestFixture {
	newSrc := func() client.Object {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testSourceNS,
				Name:      testResourceName,
				Labels: map[string]string{
					kargoapi.LabelKeyCredentialType: "generic",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"key": []byte("value")},
		}
	}
	hashOf := func(obj client.Object) string {
		s, ok := obj.(*corev1.Secret)
		if !ok {
			panic("failed to cast object to Secret")
		}
		return computeSecretHash(s)
	}
	return reconcilerTestFixture{
		adapter: secretAdapter{},
		newSrc:  newSrc,
		withDeletion: func(obj client.Object) client.Object {
			s, ok := obj.(*corev1.Secret)
			if !ok {
				panic("failed to cast object to Secret")
			}
			cp := s.DeepCopy()
			now := metav1.NewTime(time.Now())
			cp.DeletionTimestamp = &now
			return cp
		},
		newReplica: func(ns string, src client.Object) client.Object {
			s, ok := src.(*corev1.Secret)
			if !ok {
				panic("failed to cast source to Secret")
			}
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      testResourceName,
					Labels: map[string]string{
						kargoapi.LabelKeyReplicatedFrom: testResourceName,
						kargoapi.LabelKeyReplicatedSHA:  computeSecretHash(s),
					},
				},
				Type: s.Type,
				Data: s.Data,
			}
		},
		withUpdatedData: func(obj client.Object) client.Object {
			s, ok := obj.(*corev1.Secret)
			if !ok {
				panic("failed to cast object to Secret")
			}
			cp := s.DeepCopy()
			cp.Data = map[string][]byte{"key": []byte("new-value")}
			return cp
		},
		newExternallyModifiedReplica: func(ns string, src client.Object) client.Object {
			s, ok := src.(*corev1.Secret)
			if !ok {
				panic("failed to cast source to Secret")
			}
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      testResourceName,
					Labels: map[string]string{
						kargoapi.LabelKeyReplicatedFrom: testResourceName,
						kargoapi.LabelKeyReplicatedSHA:  computeSecretHash(s),
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{"key": []byte("externally-modified")},
			}
		},
		newConflictingResource: func(ns string) client.Object {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: testResourceName},
				Type:       corev1.SecretTypeOpaque,
				Data:       map[string][]byte{"user-key": []byte("user-value")},
			}
		},
		checkReplica: func(t *testing.T, replica, expectedSrc client.Object) {
			t.Helper()
			d, ok := replica.(*corev1.Secret)
			if !ok {
				panic("failed to cast replica to Secret")
			}
			s, ok := expectedSrc.(*corev1.Secret)
			if !ok {
				panic("failed to cast expectedSrc to Secret")
			}
			require.Equal(t, s.Data, d.Data)
			require.Equal(t, s.Type, d.Type)
		},
		computeHash: hashOf,
	}
}

func configMapFixture() reconcilerTestFixture {
	newSrc := func() client.Object {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testSourceNS,
				Name:      testResourceName,
			},
			Data: map[string]string{"key": "value"},
		}
	}
	hashOf := func(obj client.Object) string {
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok {
			panic("failed to cast object to ConfigMap")
		}
		return computeConfigMapHash(cm)
	}
	return reconcilerTestFixture{
		adapter: configMapAdapter{},
		newSrc:  newSrc,
		withDeletion: func(obj client.Object) client.Object {
			cm, ok := obj.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast object to ConfigMap")
			}
			cp := cm.DeepCopy()
			now := metav1.NewTime(time.Now())
			cp.DeletionTimestamp = &now
			return cp
		},
		newReplica: func(ns string, src client.Object) client.Object {
			cm, ok := src.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast source to ConfigMap")
			}
			return &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      testResourceName,
					Labels: map[string]string{
						kargoapi.LabelKeyReplicatedFrom: testResourceName,
						kargoapi.LabelKeyReplicatedSHA:  computeConfigMapHash(cm),
					},
				},
				Data:       cm.Data,
				BinaryData: cm.BinaryData,
			}
		},
		withUpdatedData: func(obj client.Object) client.Object {
			cm, ok := obj.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast object to ConfigMap")
			}
			cp := cm.DeepCopy()
			cp.Data = map[string]string{"key": "new-value"}
			return cp
		},
		newExternallyModifiedReplica: func(ns string, src client.Object) client.Object {
			cm, ok := src.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast source to ConfigMap")
			}
			return &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      testResourceName,
					Labels: map[string]string{
						kargoapi.LabelKeyReplicatedFrom: testResourceName,
						kargoapi.LabelKeyReplicatedSHA:  computeConfigMapHash(cm),
					},
				},
				Data: map[string]string{"key": "externally-modified"},
			}
		},
		newConflictingResource: func(ns string) client.Object {
			return &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: testResourceName},
				Data:       map[string]string{"user-key": "user-value"},
			}
		},
		checkReplica: func(t *testing.T, replica, expectedSrc client.Object) {
			t.Helper()
			d, ok := replica.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast replica to ConfigMap")
			}
			s, ok := expectedSrc.(*corev1.ConfigMap)
			if !ok {
				panic("failed to cast expectedSrc to ConfigMap")
			}
			require.Equal(t, s.Data, d.Data)
			require.Equal(t, s.BinaryData, d.BinaryData)
		},
		computeHash: hashOf,
	}
}

// ---- Top-level test entry points ----

func TestReconciler_Secret(t *testing.T) {
	runReconcilerTests(t, secretFixture())
}

func TestReconciler_ConfigMap(t *testing.T) {
	runReconcilerTests(t, configMapFixture())
}

// ---- Parameterized reconciler test suite ----

func runReconcilerTests(t *testing.T, f reconcilerTestFixture) {
	t.Helper()

	t.Run("SourceNotFound", func(t *testing.T) {
		fc := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()
		r := reconcilerForTest(fc, f)
		_, err := doReconcile(t, r)
		require.NoError(t, err)
	})

	t.Run("NoAnnotationNoFinalizer", func(t *testing.T) {
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(f.newSrc()).
			Build()
		r := reconcilerForTest(fc, f)
		result, err := doReconcile(t, r)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)

		// Verify no additional resources were created.
		list := f.adapter.newList()
		require.NoError(t, fc.List(t.Context(), list))
		require.Len(t, f.adapter.getItems(list), 1) // only the source
	})

	t.Run("AnnotationPresent_NoProjects_AddsFinalizerAndRequeues", func(t *testing.T) {
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(withAnnotation(f.newSrc())).
			Build()
		r := reconcilerForTest(fc, f)

		// First reconcile: should add finalizer and requeue.
		result, err := doReconcile(t, r)
		require.NoError(t, err)
		require.Equal(t, 100*time.Millisecond, result.RequeueAfter)

		// Verify finalizer was added.
		src := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testSourceNS, Name: testResourceName,
		}, src))
		require.True(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerName))

		// Second reconcile (finalizer present, no projects): no replicas created.
		result, err = doReconcile(t, r)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)

		list := f.adapter.newList()
		require.NoError(t, fc.List(t.Context(), list, client.InNamespace(testProject1)))
		require.Empty(t, f.adapter.getItems(list))
	})

	t.Run("AnnotationPresent_TwoProjects_CreatesReplicas", func(t *testing.T) {
		src := withFinalizer(withAnnotation(f.newSrc()))
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(src, project(testProject1), project(testProject2)).
			Build()
		r := reconcilerForTest(fc, f)

		result, err := doReconcile(t, r)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)

		expectedHash := f.computeHash(f.newSrc())
		for _, ns := range []string{testProject1, testProject2} {
			dest := f.adapter.newObject()
			require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
				Namespace: ns, Name: testResourceName,
			}, dest), "expected replicated resource in namespace %q", ns)
			require.Equal(t, testResourceName, dest.GetLabels()[kargoapi.LabelKeyReplicatedFrom])
			require.Equal(t, expectedHash, dest.GetLabels()[kargoapi.LabelKeyReplicatedSHA])
			f.checkReplica(t, dest, f.newSrc())
		}
	})

	t.Run("LabelsAndAnnotationsCarriedOver", func(t *testing.T) {
		src := f.newSrc()
		src.SetLabels(map[string]string{
			"team":                          "infra",
			kargoapi.LabelKeyCredentialType: "generic",
		})
		src.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyReplicateTo: "*",
			lastAppliedConfigAnnotation:       `{"big":"json"}`,
			"custom.io/owner":                 "ops",
		})
		src = withFinalizer(src)

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(src, project(testProject1)).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		dest := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest))

		// Source labels merged with replication labels.
		require.Equal(t, "infra", dest.GetLabels()["team"])
		require.Equal(t, testResourceName, dest.GetLabels()[kargoapi.LabelKeyReplicatedFrom])
		require.NotEmpty(t, dest.GetLabels()[kargoapi.LabelKeyReplicatedSHA])

		// Custom annotation carried over; excluded ones stripped.
		require.Equal(t, "ops", dest.GetAnnotations()["custom.io/owner"])
		require.NotContains(t, dest.GetAnnotations(), kargoapi.AnnotationKeyReplicateTo)
		require.NotContains(t, dest.GetAnnotations(), lastAppliedConfigAnnotation)
	})

	t.Run("AlreadyUpToDate_NoUpdate", func(t *testing.T) {
		updateCalled := false
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				withFinalizer(withAnnotation(f.newSrc())),
				project(testProject1),
				f.newReplica(testProject1, f.newSrc()),
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
			}).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)
		require.False(t, updateCalled, "expected no update when replica is already up to date")
	})

	t.Run("SourceUpdated_UpdatesReplica", func(t *testing.T) {
		baseSrc := f.newSrc()
		updatedSrc := withFinalizer(withAnnotation(f.withUpdatedData(f.newSrc())))
		oldReplica := f.newReplica(testProject1, baseSrc)

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(updatedSrc, project(testProject1), oldReplica).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		expectedHash := f.computeHash(f.withUpdatedData(f.newSrc()))
		dest := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest))
		require.Equal(t, expectedHash, dest.GetLabels()[kargoapi.LabelKeyReplicatedSHA])
		f.checkReplica(t, dest, f.withUpdatedData(f.newSrc()))
	})

	t.Run("ExternallyModified_Skipped", func(t *testing.T) {
		updateCalled := false
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				withFinalizer(withAnnotation(f.newSrc())),
				project(testProject1),
				f.newExternallyModifiedReplica(testProject1, f.newSrc()),
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
			}).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)
		require.False(t, updateCalled, "externally modified replica should not be updated")
	})

	t.Run("NoReplicatedFromLabel_Conflict_Skipped", func(t *testing.T) {
		updateCalled := false
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				withFinalizer(withAnnotation(f.newSrc())),
				project(testProject1),
				f.newConflictingResource(testProject1),
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
			}).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)
		require.False(t, updateCalled, "user-created resource should not be updated")

		// Verify user's resource data was preserved.
		dest := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest))
		f.checkReplica(t, dest, f.newConflictingResource(testProject1))
	})

	t.Run("DeletionTimestamp_CleansUpAndRemovesFinalizer", func(t *testing.T) {
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				f.withDeletion(withFinalizer(withAnnotation(f.newSrc()))),
				f.newReplica(testProject1, f.newSrc()),
				f.newReplica(testProject2, f.newSrc()),
			).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		// Replicated resources should be deleted.
		for _, ns := range []string{testProject1, testProject2} {
			dest := f.adapter.newObject()
			getErr := fc.Get(t.Context(), types.NamespacedName{
				Namespace: ns, Name: testResourceName,
			}, dest)
			require.True(t, apierrors.IsNotFound(getErr),
				"expected replicated resource in %q to be deleted", ns)
		}

		// Source should have its finalizer removed (fake client deletes it
		// when no other finalizers remain).
		src := f.adapter.newObject()
		getErr := fc.Get(t.Context(), types.NamespacedName{
			Namespace: testSourceNS, Name: testResourceName,
		}, src)
		require.True(t, apierrors.IsNotFound(getErr))
	})

	t.Run("FinalizerPresentNoAnnotation_CleansUpOnStartup", func(t *testing.T) {
		// Simulates startup after the replicate-to annotation was removed while
		// the controller was down. The CreateFunc predicate now passes these
		// objects through, so the cleanup path must run.
		srcWithFinalizer := withFinalizer(f.newSrc()) // finalizer present, no annotation

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(srcWithFinalizer, f.newReplica(testProject1, f.newSrc())).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		// Replicated resource should be deleted.
		dest := f.adapter.newObject()
		getErr := fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest)
		require.True(t, apierrors.IsNotFound(getErr))

		// Source should have its replication finalizer removed.
		src := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testSourceNS, Name: testResourceName,
		}, src))
		require.False(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerName))
	})

	t.Run("AnnotationRemoved_CleansUpAndRemovesFinalizer", func(t *testing.T) {
		srcWithoutAnnotation := withFinalizer(f.newSrc()) // finalizer present, no annotation

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				srcWithoutAnnotation,
				f.newReplica(testProject1, f.newSrc()),
			).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		// Replicated resource should be deleted.
		dest := f.adapter.newObject()
		getErr := fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest)
		require.True(t, apierrors.IsNotFound(getErr))

		// Source should have its replication finalizer removed.
		src := f.adapter.newObject()
		require.NoError(t, fc.Get(t.Context(), types.NamespacedName{
			Namespace: testSourceNS, Name: testResourceName,
		}, src))
		require.False(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerName))
	})

	t.Run("DeletionTimestamp_DeletesAllReplicas", func(t *testing.T) {
		// Cleanup deletes all replicas regardless of external modification.
		modifiedReplica := f.newExternallyModifiedReplica(testProject1, f.newSrc())

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				f.withDeletion(withFinalizer(withAnnotation(f.newSrc()))),
				modifiedReplica,
			).
			Build()
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.NoError(t, err)

		dest := f.adapter.newObject()
		getErr := fc.Get(t.Context(), types.NamespacedName{
			Namespace: testProject1, Name: testResourceName,
		}, dest)
		require.True(t, apierrors.IsNotFound(getErr))
	})

	t.Run("CreateError", func(t *testing.T) {
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(withFinalizer(withAnnotation(f.newSrc())), project(testProject1)).
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
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.ErrorContains(t, err, "something went wrong")
	})

	t.Run("UpdateError", func(t *testing.T) {
		updatedSrc := withFinalizer(withAnnotation(f.withUpdatedData(f.newSrc())))
		oldReplica := f.newReplica(testProject1, f.newSrc())

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(updatedSrc, project(testProject1), oldReplica).
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
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.ErrorContains(t, err, "something went wrong")
	})

	t.Run("DeleteError_DuringCleanup", func(t *testing.T) {
		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(
				f.withDeletion(withFinalizer(withAnnotation(f.newSrc()))),
				f.newReplica(testProject1, f.newSrc()),
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
		r := reconcilerForTest(fc, f)

		_, err := doReconcile(t, r)
		require.ErrorContains(t, err, "something went wrong")
	})

	t.Run("ProjectCreatedEnqueuer", func(t *testing.T) {
		annotated := withAnnotation(f.newSrc())
		unannotated := f.newSrc()
		unannotated.SetName("unannotated")

		fc := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(annotated, unannotated).
			Build()

		enqueuer := &projectCreatedEnqueuer{
			client:   fc,
			sourceNS: testSourceNS,
			adapter:  f.adapter,
		}

		wq := &fakeWorkQueue{}
		l, err := logging.NewLogger(logging.DebugLevel, logging.DefaultFormat)
		require.NoError(t, err)
		ctx := logging.ContextWithLogger(t.Context(), l)

		enqueuer.Create(
			ctx,
			event.TypedCreateEvent[*kargoapi.Project]{Object: project(testProject1)},
			wq,
		)

		require.Len(t, wq.items, 1)
		require.Equal(t, testResourceName, wq.items[0].Name)
		require.Equal(t, testSourceNS, wq.items[0].Namespace)
	})
}

// ---- Adapter tests ----

func TestSecretAdapter_ShouldReconcile(t *testing.T) {
	adapter := secretAdapter{}

	t.Run("Secret without credential type label should not reconcile", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-ns",
				Labels:    map[string]string{"other": "label"},
			},
		}
		require.False(t, adapter.shouldReconcile(secret))
	})

	t.Run("Secret with credential type label should reconcile", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-ns",
				Labels:    map[string]string{kargoapi.LabelKeyCredentialType: "generic"},
			},
		}
		require.True(t, adapter.shouldReconcile(secret))
	})

	t.Run("Secret with no labels should not reconcile", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-ns",
			},
		}
		require.False(t, adapter.shouldReconcile(secret))
	})

	t.Run("Non-secret object should not reconcile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "test-ns",
			},
		}
		require.False(t, adapter.shouldReconcile(cm))
	})
}

func TestConfigMapAdapter_ShouldReconcile(t *testing.T) {
	adapter := configMapAdapter{}

	t.Run("ConfigMap should always reconcile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "test-ns",
			},
		}
		require.True(t, adapter.shouldReconcile(cm))
	})

	t.Run("ConfigMap with no labels should reconcile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "test-ns",
			},
		}
		require.True(t, adapter.shouldReconcile(cm))
	})

	t.Run("ConfigMap with labels should reconcile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "test-ns",
				Labels:    map[string]string{"env": "prod"},
			},
		}
		require.True(t, adapter.shouldReconcile(cm))
	})
}

func TestReconcile_SkipsReconcileWhenAdapterRetursFalse(t *testing.T) {
	scheme := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a non-credential Secret in the shared resources namespace (no credential type label)
	nonCredentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testSourceNS,
			Annotations: map[string]string{
				kargoapi.AnnotationKeyReplicateTo: kargoapi.AnnotationValueReplicateToAll,
			},
			Labels: map[string]string{}, // No credential type label
		},
		Data: map[string][]byte{"key": []byte("value")},
		Type: corev1.SecretTypeOpaque,
	}
	require.NoError(t, fc.Create(t.Context(), nonCredentialSecret))

	// Add the finalizer so it won't try to re-reconcile
	controllerutil.AddFinalizer(nonCredentialSecret, kargoapi.FinalizerName)
	require.NoError(t, fc.Update(t.Context(), nonCredentialSecret))

	// Create a test project
	require.NoError(t, fc.Create(t.Context(), project("test-project")))

	r := reconcilerForTest(fc, secretFixture())

	// Reconcile should succeed but not replicate because the Secret lacks the credential type label
	result, err := doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)

	// Verify the Secret was not replicated to the project namespace
	replicatedList := &corev1.SecretList{}
	require.NoError(t, fc.List(t.Context(), replicatedList, client.InNamespace("test-project")))
	require.Empty(t, replicatedList.Items)
}

func TestReconcile_ReplicatesWhenAdapterReturnsTrue(t *testing.T) {
	scheme := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a credential Secret in the shared resources namespace (has credential type label)
	credentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testSourceNS,
			Annotations: map[string]string{
				kargoapi.AnnotationKeyReplicateTo: kargoapi.AnnotationValueReplicateToAll,
			},
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: "generic", // Has credential type label
			},
		},
		Data: map[string][]byte{"key": []byte("value")},
		Type: corev1.SecretTypeOpaque,
	}
	require.NoError(t, fc.Create(t.Context(), credentialSecret))

	// Add the finalizer
	controllerutil.AddFinalizer(credentialSecret, kargoapi.FinalizerName)
	require.NoError(t, fc.Update(t.Context(), credentialSecret))

	// Create a test project
	require.NoError(t, fc.Create(t.Context(), project("test-project")))

	r := reconcilerForTest(fc, secretFixture())

	// Reconcile should succeed and replicate because the Secret has the credential type label
	result, err := doReconcile(t, r)
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{}, result)

	// Verify the Secret was replicated to the project namespace
	replicatedList := &corev1.SecretList{}
	require.NoError(t, fc.List(t.Context(), replicatedList, client.InNamespace("test-project")))
	require.Len(t, replicatedList.Items, 1)
	require.Equal(t, testResourceName, replicatedList.Items[0].GetName())
}

// ---- Hash function tests ----

// ---- fakeWorkQueue ----

type fakeWorkQueue struct {
	items []reconcile.Request
}

var _ workqueue.TypedRateLimitingInterface[reconcile.Request] = &fakeWorkQueue{}

func (q *fakeWorkQueue) Add(item reconcile.Request) { q.items = append(q.items, item) }
func (q *fakeWorkQueue) AddAfter(item reconcile.Request, _ time.Duration) {
	q.items = append(q.items, item)
}
func (q *fakeWorkQueue) AddRateLimited(item reconcile.Request) { q.items = append(q.items, item) }
func (q *fakeWorkQueue) Forget(_ reconcile.Request)            {}
func (q *fakeWorkQueue) NumRequeues(_ reconcile.Request) int   { return 0 }
func (q *fakeWorkQueue) Done(_ reconcile.Request)              {}
func (q *fakeWorkQueue) Get() (reconcile.Request, bool)        { return reconcile.Request{}, false }
func (q *fakeWorkQueue) Len() int                              { return len(q.items) }
func (q *fakeWorkQueue) ShutDown()                             {}
func (q *fakeWorkQueue) ShutDownWithDrain()                    {}
func (q *fakeWorkQueue) ShuttingDown() bool                    { return false }
