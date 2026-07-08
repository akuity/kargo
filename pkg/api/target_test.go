package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetTargetsForStage(t *testing.T) {
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
			},
		}
	}

	testCases := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, []kargoapi.Target, error)
	}{
		{
			name:  "no selector: synthesizes an ephemeral default Target",
			stage: newStage(),
			objects: []client.Object{
				// An unrelated Target that must NOT be returned when no selector
				// is set.
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other",
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				},
			},
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Len(t, targets, 1)
				target := targets[0]
				require.Equal(t, testStageName, target.Name)
				require.Equal(t, testNamespace, target.Namespace)
				require.Equal(t, testStageName, target.Labels[kargoapi.LabelKeyStage])
				// Ephemeral: never persisted, so it has no resource version.
				require.Empty(t, target.ResourceVersion)
			},
		},
		{
			name: "no selector: mirrors the Stage shard label",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Labels = map[string]string{kargoapi.LabelKeyShard: "shard-1"}
				return s
			}(),
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Len(t, targets, 1)
				require.Equal(t, "shard-1", targets[0].Labels[kargoapi.LabelKeyShard])
			},
		},
		{
			name: "selector: returns matching Targets in the namespace",
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
						Name:      "prod-a",
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				},
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prod-b",
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "prod"},
					},
				},
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dev",
						Namespace: testNamespace,
						Labels:    map[string]string{"env": "dev"},
					},
				},
				// Matching labels but a different namespace: must be excluded.
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prod-elsewhere",
						Namespace: "other-ns",
						Labels:    map[string]string{"env": "prod"},
					},
				},
			},
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				names := make([]string, len(targets))
				for i, tg := range targets {
					names[i] = tg.Name
				}
				require.ElementsMatch(t, []string{"prod-a", "prod-b"}, names)
			},
		},
		{
			name: "selector: no matches returns an empty list",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Spec.TargetSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				return s
			}(),
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "selector: propagates list errors",
			stage: func() *kargoapi.Stage {
				s := newStage()
				s.Spec.TargetSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{"env": "prod"},
				}
				return s
			}(),
			interceptor: interceptor.Funcs{
				List: func(
					_ context.Context,
					_ client.WithWatch,
					_ client.ObjectList,
					_ ...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.Target, err error) {
				require.ErrorContains(t, err, "something went wrong")
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

			targets, err := GetTargetsForStage(context.Background(), c, tc.stage)
			tc.assertions(t, targets, err)
		})
	}
}
