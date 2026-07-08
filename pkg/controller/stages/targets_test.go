package stages

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestRegularStageReconciler_syncTarget(t *testing.T) {
	const (
		testNamespace = "kargo-demo"
		testStageName = "test"
	)

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	newStage := func() *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testStageName,
				Namespace: testNamespace,
				UID:       types.UID("stage-uid"),
			},
		}
	}

	getTarget := func(t *testing.T, c client.Client) *kargoapi.Target {
		t.Helper()
		target := &kargoapi.Target{}
		require.NoError(t, c.Get(
			context.Background(),
			types.NamespacedName{Namespace: testNamespace, Name: testStageName},
			target,
		))
		return target
	}

	requireNoTarget := func(t *testing.T, c client.Client) {
		t.Helper()
		err := c.Get(
			context.Background(),
			types.NamespacedName{Namespace: testNamespace, Name: testStageName},
			&kargoapi.Target{},
		)
		require.True(t, apierrors.IsNotFound(err))
	}

	testCases := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name:  "no selector: creates a Target owned by the Stage",
			stage: newStage(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				target := getTarget(t, c)
				require.Equal(t, testStageName, target.Labels[kargoapi.LabelKeyStage])
				require.Empty(t, target.Spec.Params)
				require.Len(t, target.OwnerReferences, 1)
				ref := target.OwnerReferences[0]
				require.Equal(t, "Stage", ref.Kind)
				require.Equal(t, types.UID("stage-uid"), ref.UID)
				require.NotNil(t, ref.Controller)
				require.True(t, *ref.Controller)
			},
		},
		{
			name: "no selector: mirrors the Stage shard label",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Labels = map[string]string{kargoapi.LabelKeyShard: "shard-1"}
				return s
			}(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				require.Equal(t, "shard-1", getTarget(t, c).Labels[kargoapi.LabelKeyShard])
			},
		},
		{
			name:  "no selector: leaves an already-correct Target untouched",
			stage: newStage(),
			objects: []client.Object{
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:            testStageName,
						Namespace:       testNamespace,
						ResourceVersion: "999",
						Labels:          map[string]string{kargoapi.LabelKeyStage: testStageName},
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Stage",
							Name:       testStageName,
							UID:        types.UID("stage-uid"),
							Controller: ptr.To(true),
						}},
					},
					Spec: kargoapi.TargetSpec{
						Params: map[string]apiextensionsv1.JSON{
							"cluster": {Raw: []byte(`"https://c-00427.example.com"`)},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				target := getTarget(t, c)
				require.Equal(t, "999", target.ResourceVersion)
				require.Contains(t, target.Spec.Params, "cluster")
			},
		},
		{
			name:  "no selector: repairs managed metadata but preserves params",
			stage: newStage(),
			objects: []client.Object{
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
					},
					Spec: kargoapi.TargetSpec{
						Params: map[string]apiextensionsv1.JSON{
							"branch": {Raw: []byte(`"clusters/c-00427"`)},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				target := getTarget(t, c)
				require.Equal(t, testStageName, target.Labels[kargoapi.LabelKeyStage])
				require.Len(t, target.OwnerReferences, 1)
				require.True(t, *target.OwnerReferences[0].Controller)
				require.Contains(t, target.Spec.Params, "branch")
			},
		},
		{
			name:  "no selector: propagates errors getting the Target",
			stage: newStage(),
			interceptor: interceptor.Funcs{
				Get: func(
					ctx context.Context,
					c client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if _, ok := obj.(*kargoapi.Target); ok {
						return errors.New("something went wrong")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:  "no selector: treats AlreadyExists on create as success",
			stage: newStage(),
			interceptor: interceptor.Funcs{
				Create: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ ...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(
						schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "targets"},
						obj.GetName(),
					)
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "with selector: deletes the auto-created Target",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Spec.TargetSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				return s
			}(),
			objects: []client.Object{
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Stage",
							Name:       testStageName,
							UID:        types.UID("stage-uid"),
							Controller: ptr.To(true),
						}},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				requireNoTarget(t, c)
			},
		},
		{
			name: "with selector: leaves a Target it does not own",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Spec.TargetSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				return s
			}(),
			objects: []client.Object{
				// A user-managed Target that happens to share the Stage's name
				// but is not controlled by the Stage.
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testStageName,
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				// It must survive: not owned by the Stage.
				require.NotNil(t, getTarget(t, c))
			},
		},
		{
			name: "with selector: no auto Target to delete is a no-op",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Spec.TargetSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				return s
			}(),
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				requireNoTarget(t, c)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				WithInterceptorFuncs(tc.interceptor).
				Build()

			r := &RegularStageReconciler{client: c}
			err := r.syncTarget(context.Background(), tc.stage)
			tc.assertions(t, c, err)
		})
	}
}
