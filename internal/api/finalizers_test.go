package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestEnsureFinalizer(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testStageName = "fake-stage"

	ctx := context.Background()

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testStageName,
		},
	}

	scheme := k8sruntime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()

	updated, err := EnsureFinalizer(ctx, c, stage)
	require.NoError(t, err)
	require.True(t, updated)

	patchedStage := &kargoapi.Stage{}
	err = c.Get(
		ctx,
		types.NamespacedName{
			Namespace: testNamespace,
			Name:      testStageName,
		},
		patchedStage,
	)
	require.NoError(t, err)

	require.True(t, controllerutil.ContainsFinalizer(patchedStage, kargoapi.FinalizerName))
}

func TestRemoveFinalizer(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testStageName = "fake-stage"

	ctx := context.Background()

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  testNamespace,
			Name:       testStageName,
			Finalizers: []string{kargoapi.FinalizerName},
		},
	}

	scheme := k8sruntime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()

	err = RemoveFinalizer(ctx, c, stage)
	require.NoError(t, err)

	patchedStage := &kargoapi.Stage{}
	err = c.Get(
		ctx,
		types.NamespacedName{
			Namespace: testNamespace,
			Name:      testStageName,
		},
		patchedStage,
	)
	require.NoError(t, err)

	require.False(t, controllerutil.ContainsFinalizer(patchedStage, kargoapi.FinalizerName))
}

func TestPatchOwnerReferences(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testProjectName = "fake-project"

	ctx := context.Background()

	initialNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}

	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: testProjectName,
		},
	}

	scheme := k8sruntime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(scheme)
	require.NoError(t, err)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		initialNS,
		testProject,
	).Build()

	newNS := initialNS.DeepCopy()

	ownerRef := metav1.NewControllerRef(
		testProject,
		kargoapi.GroupVersion.WithKind("Project"),
	)
	ownerRef.BlockOwnerDeletion = ptr.To(false)

	newNS.OwnerReferences = []metav1.OwnerReference{
		*ownerRef,
	}

	err = PatchOwnerReferences(ctx, c, newNS)
	require.NoError(t, err)

	patchedNS := &corev1.Namespace{}
	err = c.Get(
		ctx,
		types.NamespacedName{
			Name: testNamespace,
		},
		patchedNS,
	)
	require.NoError(t, err)

	require.Equal(t, newNS.OwnerReferences, patchedNS.OwnerReferences)
}

func Test_patchAnnotation(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	newFakeClient := func(obj ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj...).
			Build()
	}

	testCases := map[string]struct {
		obj         client.Object
		client      client.Client
		key         string
		value       string
		errExpected bool
	}{
		"stage": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			key:   "key",
			value: "value",
		},
		"stage with existing annotation": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			}),
			key:   "key",
			value: "value2",
		},
		"non-existing stage": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client:      newFakeClient(),
			key:         "key",
			value:       "value",
			errExpected: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := patchAnnotation(context.Background(), tc.client, tc.obj, tc.key, tc.value)
			if tc.errExpected {
				require.Error(t, err)
				return
			}
			require.Contains(t, tc.obj.GetAnnotations(), tc.key)
			require.Equal(t, tc.obj.GetAnnotations()[tc.key], tc.value)
		})
	}
}

func Test_deleteAnnotation(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	newFakeClient := func(obj ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj...).
			Build()
	}

	testCases := map[string]struct {
		obj         client.Object
		client      client.Client
		key         string
		value       string
		errExpected bool
	}{
		"stage without annotation": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			key: "key",
		},
		"stage with existing annotation": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						"key": "value",
					},
				},
			}),
			key: "key",
		},
		"non-existing stage": {
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client:      newFakeClient(),
			key:         "key",
			value:       "value",
			errExpected: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := deleteAnnotation(context.Background(), tc.client, tc.obj, tc.key)
			if tc.errExpected {
				require.Error(t, err)
				return
			}
			require.NotContains(t, tc.obj.GetAnnotations(), tc.key)
		})
	}
}
