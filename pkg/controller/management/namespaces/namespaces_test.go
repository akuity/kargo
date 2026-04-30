package namespaces

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func TestNewReconciler(t *testing.T) {
	r := newReconciler(fake.NewClientBuilder().Build())
	require.NotNil(t, r.client)
	require.NotNil(t, r.getNamespaceFn)
	require.NotNil(t, r.deleteProjectFn)
	require.NotNil(t, r.removeFinalizerFn)
	require.NotNil(t, r.patchOwnerReferencesFn)
}

func TestReconcile(t *testing.T) {
	const testProjectName = "fake-project"
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(testScheme))
	kClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testProjectName,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "Project",
						Name:       testProjectName,
					},
				},
			},
		},
	).Build()

	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, ctrl.Result, error)
	}{
		{
			name: "namespace not not found",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "error getting namespace",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ensure project ownership relationship is removed if present",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					o client.Object,
					_ ...client.GetOption,
				) error {
					// return the Namespace with a Project owner reference to test that the reconciler removes it.
					ns := o.(*corev1.Namespace) // nolint: forcetypeassert
					ns.Name = testProjectName
					ns.OwnerReferences = []metav1.OwnerReference{
						{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
							Name:       testProjectName,
						},
					}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				patchOwnerReferencesFn: api.PatchOwnerReferences,
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
				ns := new(corev1.Namespace)
				name := types.NamespacedName{Name: testProjectName}
				require.NoError(t, kClient.Get(t.Context(), name, ns))
				require.Len(t, ns.OwnerReferences, 0)
				require.Empty(t, ns.OwnerReferences)
			},
		},
		{
			name: "error patching ownership references",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					o client.Object,
					_ ...client.GetOption,
				) error {
					// return the Namespace with a Project owner reference.
					ns := o.(*corev1.Namespace) // nolint: forcetypeassert
					ns.Name = testProjectName
					ns.OwnerReferences = []metav1.OwnerReference{
						{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
							Name:       testProjectName,
						},
					}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				patchOwnerReferencesFn: func(
					context.Context,
					client.Client,
					client.Object) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err,
					"failed to patch owner references for namespace \"fake-project\": something went wrong",
				)
			},
		},
		{
			name: "namespace is not being deleted",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					// The empty ns object that gets passed in should already not have
					// a deletion timestamp set.
					return nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "namespace does not need finalizing",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					// nolint: forcetypeassert
					obj.(*corev1.Namespace).DeletionTimestamp = &metav1.Time{}
					return nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "project not found",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.DeletionTimestamp = &metav1.Time{}
					ns.Finalizers = []string{kargoapi.FinalizerName}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "error deleting project",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.DeletionTimestamp = &metav1.Time{}
					ns.Finalizers = []string{kargoapi.FinalizerName}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.ErrorContains(t, err, "error deleting Project")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error removing finalizer",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.DeletionTimestamp = &metav1.Time{}
					ns.Finalizers = []string{kargoapi.FinalizerName}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.ErrorContains(t, err, "error removing finalizer")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				client: kClient,
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.DeletionTimestamp = &metav1.Time{}
					ns.Finalizers = []string{kargoapi.FinalizerName}
					return nil
				},
				deleteProjectFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.reconciler.Reconcile(t.Context(), ctrl.Request{})
			testCase.assertions(t, res, err)
		})
	}
}
