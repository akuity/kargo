package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestAddFinalizer(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testStageName = "fake-stage"

	ctx := context.Background()

	stage := &Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testStageName,
		},
	}

	scheme := k8sruntime.NewScheme()
	err := AddToScheme(scheme)
	require.NoError(t, err)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()

	err = AddFinalizer(ctx, c, stage)
	require.NoError(t, err)

	patchedStage := &Stage{}
	err = c.Get(
		ctx,
		types.NamespacedName{
			Namespace: testNamespace,
			Name:      testStageName,
		},
		patchedStage,
	)
	require.NoError(t, err)

	require.True(t, controllerutil.ContainsFinalizer(patchedStage, FinalizerName))
}

func TestClearAnnotations(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	newFakeClient := func(obj ...client.Object) client.Client {
		return fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj...).
			Build()
	}

	testCases := []struct {
		name       string
		client     client.Client
		obj        client.Object
		keys       []string
		assertions func(*testing.T, client.Object, error)
	}{
		{
			name:   "no keys",
			client: newFakeClient(),
			obj:    nil,
			keys:   nil,
			assertions: func(t *testing.T, _ client.Object, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "no annotations",
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			keys: []string{"key"},
			assertions: func(t *testing.T, _ client.Object, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:   "not found",
			client: newFakeClient(),
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			keys: []string{"key"},
			assertions: func(t *testing.T, _ client.Object, err error) {
				require.ErrorContains(t, err, "patch annotation")
			},
		},
		{
			name: "clear one",
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			}),
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			keys: []string{"key1"},
			assertions: func(t *testing.T, obj client.Object, err error) {
				require.NoError(t, err)
				require.Contains(t, obj.GetAnnotations(), "key2")
				require.NotContains(t, obj.GetAnnotations(), "key1")
			},
		},
		{
			name: "clear two",
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			}),
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			keys: []string{"key1", "key2"},
			assertions: func(t *testing.T, obj client.Object, err error) {
				require.NoError(t, err)
				require.NotContains(t, obj.GetAnnotations(), "key2")
				require.NotContains(t, obj.GetAnnotations(), "key1")
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ClearAnnotations(context.TODO(), tc.client, tc.obj, tc.keys...)
			tc.assertions(t, tc.obj, err)
		})
	}
}

func Test_patchAnnotation(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

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
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			}),
			key:   "key",
			value: "value",
		},
		"stage with existing annotation": {
			obj: &Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "stage",
				},
			},
			client: newFakeClient(&Stage{
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
			obj: &Stage{
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
		tc := tc
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
