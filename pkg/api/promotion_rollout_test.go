package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGeneratePromotionRolloutName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		stageName  string
		freight    string
		assertions func(*testing.T, string)
	}{
		{
			name:      "empty Stage name",
			stageName: "",
			freight:   "fake-freight",
			assertions: func(t *testing.T, name string) {
				require.Empty(t, name)
			},
		},
		{
			name:      "empty Freight",
			stageName: "fake-stage",
			freight:   "",
			assertions: func(t *testing.T, name string) {
				require.Empty(t, name)
			},
		},
		{
			name:      "Stage name and short Freight hash",
			stageName: "fake-stage",
			freight:   "abcdef1234567890",
			assertions: func(t *testing.T, name string) {
				parts := strings.Split(name, ".")
				require.Len(t, parts, 3)
				require.Equal(t, "fake-stage", parts[0])
				require.Equal(t, "abcdef1", parts[2])
			},
		},
		{
			name:      "over-long Stage name is truncated",
			stageName: strings.Repeat("a", 300),
			freight:   "abcdef1234567890",
			assertions: func(t *testing.T, name string) {
				require.LessOrEqual(t, len(name), 253)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assertions(t, GeneratePromotionRolloutName(testCase.stageName, testCase.freight))
		})
	}
}

func TestNewPromotionRollout(t *testing.T) {
	t.Parallel()

	const (
		project = "fake-project"
		stage   = "fake-stage"
		freight = "abcdef1234567890"
	)

	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project,
			Name:      stage,
			UID:       "fake-uid",
		},
		Spec: kargoapi.StageSpec{
			Shard: "my-shard",
			Targets: &kargoapi.StageTargets{
				Selectors: []metav1.LabelSelector{{
					MatchLabels: map[string]string{"region": "us"},
				}},
			},
		},
	}

	t.Run("records the Stage, the Freight and the Stage's Targets", func(t *testing.T) {
		t.Parallel()

		stageTargets := []kargoapi.Target{
			{ObjectMeta: metav1.ObjectMeta{Name: "us-east"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "us-west"}},
		}
		rollout, err := NewPromotionRollout(testStage, freight, stageTargets)
		require.NoError(t, err)

		require.Equal(t, project, rollout.Namespace)
		require.Equal(t, stage, rollout.Spec.Stage)
		require.Equal(t, freight, rollout.Spec.Freight)
		require.True(t, strings.HasPrefix(rollout.Name, stage+"."))
		require.Equal(
			t,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}, {Name: "us-west"}},
			rollout.Spec.Targets,
		)
	})

	t.Run("records an empty list for a Stage with no Targets", func(t *testing.T) {
		t.Parallel()

		rollout, err := NewPromotionRollout(testStage, freight, nil)
		require.NoError(t, err)
		require.NotNil(t, rollout.Spec.Targets)
		require.Empty(t, rollout.Spec.Targets)
	})

	t.Run("carries no owner references or labels", func(t *testing.T) {
		t.Parallel()

		// A PromotionRollout is not a Kubernetes resource: nothing garbage
		// collects it and no controller filters it by label.
		rollout, err := NewPromotionRollout(testStage, freight, nil)
		require.NoError(t, err)
		require.Empty(t, rollout.OwnerReferences)
		require.Empty(t, rollout.Labels)
	})

	t.Run("resolves the Stage's update strategy against its Targets", func(t *testing.T) {
		t.Parallel()

		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("50%"))
		withStrategy.Spec.Targets.UpdateStrategy.MaxFailures = new(intstr.FromInt32(2))
		stageTargets := make([]kargoapi.Target, 10)
		rollout, err := NewPromotionRollout(withStrategy, freight, stageTargets)
		require.NoError(t, err)
		require.Equal(
			t,
			kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: 5,
				MaxFailures:   2,
			},
			rollout.Spec.UpdateStrategy,
		)
	})

	t.Run("an invalid update strategy is an error", func(t *testing.T) {
		t.Parallel()

		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("half"))
		_, err := NewPromotionRollout(withStrategy, freight, make([]kargoapi.Target, 10))
		require.Error(t, err)
	})

	t.Run("a Stage without an update strategy gives the rollout none", func(t *testing.T) {
		t.Parallel()
		rollout, err := NewPromotionRollout(testStage, freight, nil)
		require.NoError(t, err)
		require.Zero(t, rollout.Spec.UpdateStrategy)
	})

}
