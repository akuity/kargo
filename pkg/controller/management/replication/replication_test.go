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
	cp := obj.DeepCopyObject().(client.Object)
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
	cp := obj.DeepCopyObject().(client.Object)
	controllerutil.AddFinalizer(cp, kargoapi.FinalizerNameReplicated)
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
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"key": []byte("value")},
		}
	}
	hashOf := func(obj client.Object) string {
		return computeSecretHash(obj.(*corev1.Secret))
	}
	return reconcilerTestFixture{
		adapter: secretAdapter{},
		newSrc:  newSrc,
		withDeletion: func(obj client.Object) client.Object {
			cp := obj.(*corev1.Secret).DeepCopy()
			now := metav1.NewTime(time.Now())
			cp.DeletionTimestamp = &now
			return cp
		},
		newReplica: func(ns string, src client.Object) client.Object {
			s := src.(*corev1.Secret)
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
			cp := obj.(*corev1.Secret).DeepCopy()
			cp.Data = map[string][]byte{"key": []byte("new-value")}
			return cp
		},
		newExternallyModifiedReplica: func(ns string, src client.Object) client.Object {
			s := src.(*corev1.Secret)
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
			d, s := replica.(*corev1.Secret), expectedSrc.(*corev1.Secret)
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
		return computeConfigMapHash(obj.(*corev1.ConfigMap))
	}
	return reconcilerTestFixture{
		adapter: configMapAdapter{},
		newSrc:  newSrc,
		withDeletion: func(obj client.Object) client.Object {
			cp := obj.(*corev1.ConfigMap).DeepCopy()
			now := metav1.NewTime(time.Now())
			cp.DeletionTimestamp = &now
			return cp
		},
		newReplica: func(ns string, src client.Object) client.Object {
			cm := src.(*corev1.ConfigMap)
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
			cp := obj.(*corev1.ConfigMap).DeepCopy()
			cp.Data = map[string]string{"key": "new-value"}
			return cp
		},
		newExternallyModifiedReplica: func(ns string, src client.Object) client.Object {
			cm := src.(*corev1.ConfigMap)
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
			d, s := replica.(*corev1.ConfigMap), expectedSrc.(*corev1.ConfigMap)
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
		require.True(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerNameReplicated))

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
		src.SetLabels(map[string]string{"team": "infra"})
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
		require.False(t, controllerutil.ContainsFinalizer(src, kargoapi.FinalizerNameReplicated))
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

// ---- Hash function tests ----

func TestComputeSecretHash(t *testing.T) {
	mk := func(labels, annotations map[string]string, data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Data:       data,
		}
	}

	t.Run("deterministic", func(t *testing.T) {
		s := mk(nil, nil, map[string][]byte{"k": []byte("v")})
		require.Equal(t, computeSecretHash(s), computeSecretHash(s))
		require.Len(t, computeSecretHash(s), 16)
	})

	t.Run("data key order independent", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"a": []byte("1"), "b": []byte("2")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"b": []byte("2"), "a": []byte("1")}))
		require.Equal(t, h1, h2)
	})

	t.Run("different data produces different hashes", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"k": []byte("v1")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"k": []byte("v2")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("empty secret", func(t *testing.T) {
		require.Len(t, computeSecretHash(&corev1.Secret{}), 16)
	})

	t.Run("label change produces different hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{"env": "prod"}, nil, nil))
		h2 := computeSecretHash(mk(map[string]string{"env": "staging"}, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("annotation change produces different hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{"owner": "team-a"}, nil))
		h2 := computeSecretHash(mk(nil, map[string]string{"owner": "team-b"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("replicate-to annotation excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{kargoapi.AnnotationKeyReplicateTo: "*"}, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("last-applied-configuration excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{lastAppliedConfigAnnotation: `{"big":"json"}`}, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("replication labels excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{
			kargoapi.LabelKeyReplicatedFrom: "src",
			kargoapi.LabelKeyReplicatedSHA:  "abc123",
		}, nil, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("label order independent", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{"a": "1", "b": "2"}, nil, nil))
		h2 := computeSecretHash(mk(map[string]string{"b": "2", "a": "1"}, nil, nil))
		require.Equal(t, h1, h2)
	})
}

func TestComputeConfigMapHash(t *testing.T) {
	mk := func(labels, annotations map[string]string, data map[string]string, binaryData map[string][]byte) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Data:       data,
			BinaryData: binaryData,
		}
	}

	t.Run("deterministic", func(t *testing.T) {
		cm := mk(nil, nil, map[string]string{"k": "v"}, nil)
		require.Equal(t, computeConfigMapHash(cm), computeConfigMapHash(cm))
		require.Len(t, computeConfigMapHash(cm), 16)
	})

	t.Run("data key order independent", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"a": "1", "b": "2"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"b": "2", "a": "1"}, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("different data produces different hashes", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v1"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v2"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("binaryData included in hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"bin": []byte("data1")}))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"bin": []byte("data2")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("data and binaryData sections are distinct", func(t *testing.T) {
		// Same key+value in Data vs BinaryData should produce different hashes.
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"k": []byte("v")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("empty configmap", func(t *testing.T) {
		require.Len(t, computeConfigMapHash(&corev1.ConfigMap{}), 16)
	})

	t.Run("label change produces different hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(map[string]string{"env": "prod"}, nil, nil, nil))
		h2 := computeConfigMapHash(mk(map[string]string{"env": "staging"}, nil, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("replicate-to annotation excluded from hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, map[string]string{kargoapi.AnnotationKeyReplicateTo: "*"}, nil, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("replication labels excluded from hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(map[string]string{
			kargoapi.LabelKeyReplicatedFrom: "src",
			kargoapi.LabelKeyReplicatedSHA:  "abc123",
		}, nil, nil, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, nil))
		require.Equal(t, h1, h2)
	})
}

// ---- fakeWorkQueue ----

type fakeWorkQueue struct {
	items []reconcile.Request
}

var _ workqueue.TypedRateLimitingInterface[reconcile.Request] = &fakeWorkQueue{}

func (q *fakeWorkQueue) Add(item reconcile.Request)                       { q.items = append(q.items, item) }
func (q *fakeWorkQueue) AddAfter(item reconcile.Request, _ time.Duration) { q.items = append(q.items, item) }
func (q *fakeWorkQueue) AddRateLimited(item reconcile.Request)            { q.items = append(q.items, item) }
func (q *fakeWorkQueue) Forget(_ reconcile.Request)                       {}
func (q *fakeWorkQueue) NumRequeues(_ reconcile.Request) int              { return 0 }
func (q *fakeWorkQueue) Done(_ reconcile.Request)                         {}
func (q *fakeWorkQueue) Get() (reconcile.Request, bool)                   { return reconcile.Request{}, false }
func (q *fakeWorkQueue) Len() int                                         { return len(q.items) }
func (q *fakeWorkQueue) ShutDown()                                        {}
func (q *fakeWorkQueue) ShutDownWithDrain()                               {}
func (q *fakeWorkQueue) ShuttingDown() bool                               { return false }
