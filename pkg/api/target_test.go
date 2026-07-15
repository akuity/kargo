package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestResolveStageTargets(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		selectors  []metav1.LabelSelector
		targets    []kargoapi.Target
		assertions func(*testing.T, []kargoapi.Target, error)
	}{
		{
			name: "classic Stage",
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, targets)
			},
		},
		{
			name: "unions and deduplicates matching Targets",
			selectors: []metav1.LabelSelector{
				{MatchLabels: map[string]string{"environment": "test"}},
				{MatchLabels: map[string]string{"region": "us-east-1"}},
			},
			targets: []kargoapi.Target{
				{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "project"}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:      "both",
					Namespace: "project",
					Labels: map[string]string{
						"environment": "test",
						"region":      "us-east-1",
					},
				}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:      "environment",
					Namespace: "project",
					Labels:    map[string]string{"environment": "test"},
				}},
			},
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"both", "environment"}, []string{
					targets[0].Name,
					targets[1].Name,
				})
			},
		},
		{
			name:      "no matching Targets",
			selectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"environment": "test"}}},
			assertions: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Empty(t, targets)
				require.NotNil(t, targets)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			stage := &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Name: "stage", Namespace: "project"},
				Spec:       kargoapi.StageSpec{TargetSelectors: testCase.selectors},
			}
			objects := make([]runtime.Object, 0, len(testCase.targets))
			for i := range testCase.targets {
				objects = append(objects, &testCase.targets[i])
			}
			client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()

			targets, err := ResolveStageTargets(context.Background(), client, stage)
			testCase.assertions(t, targets, err)
		})
	}
}
