package api

import (
	"fmt"
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

func TestNewPromotionRolloutForAllTargets(t *testing.T) {
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
		rollout, err := NewPromotionRolloutForAllTargets(testStage, freight, stageTargets, nil)
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

		rollout, err := NewPromotionRolloutForAllTargets(testStage, freight, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, rollout.Spec.Targets)
		require.Empty(t, rollout.Spec.Targets)
	})

	t.Run("carries no owner references or labels", func(t *testing.T) {
		t.Parallel()

		// A PromotionRollout is not a Kubernetes resource: nothing garbage
		// collects it and no controller filters it by label.
		rollout, err := NewPromotionRolloutForAllTargets(testStage, freight, nil, nil)
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
		rollout, err := NewPromotionRolloutForAllTargets(withStrategy, freight, stageTargets, nil)
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
		_, err := NewPromotionRolloutForAllTargets(withStrategy, freight, make([]kargoapi.Target, 10), nil)
		require.Error(t, err)
	})

	t.Run("a Stage without an update strategy gives the rollout none", func(t *testing.T) {
		t.Parallel()
		rollout, err := NewPromotionRolloutForAllTargets(testStage, freight, nil, nil)
		require.NoError(t, err)
		require.Zero(t, rollout.Spec.UpdateStrategy)
	})

	t.Run("an explicit update strategy overrides the calculation", func(t *testing.T) {
		t.Parallel()

		// The Stage would resolve to MaxConcurrent 5 (50% of 10), but the
		// caller-supplied strategy is used verbatim instead.
		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("50%"))
		rollout, err := NewPromotionRolloutForAllTargets(
			withStrategy,
			freight,
			make([]kargoapi.Target, 10),
			&kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: 10,
				MaxFailures:   10,
			},
		)
		require.NoError(t, err)
		require.Equal(
			t,
			kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: 10,
				MaxFailures:   10,
			},
			rollout.Spec.UpdateStrategy,
		)
	})

	t.Run("a nil Stage is an error", func(t *testing.T) {
		t.Parallel()
		_, err := NewPromotionRolloutForAllTargets(nil, freight, nil, nil)
		require.Error(t, err)
	})

	t.Run("a Stage without a targets block is an error", func(t *testing.T) {
		t.Parallel()

		classic := testStage.DeepCopy()
		classic.Spec.Targets = nil
		_, err := NewPromotionRolloutForAllTargets(classic, freight, nil, nil)
		require.Error(t, err)
	})
}

func TestNewPromotionRolloutForSelectTargets(t *testing.T) {
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

	stageTargets := []kargoapi.Target{
		{ObjectMeta: metav1.ObjectMeta{Name: "us-east"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "us-west"}},
	}

	t.Run("records only the selected Targets", func(t *testing.T) {
		t.Parallel()

		rollout, err := NewPromotionRolloutForSelectTargets(
			testStage,
			freight,
			stageTargets,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}},
			nil,
		)
		require.NoError(t, err)

		require.Equal(t, project, rollout.Namespace)
		require.Equal(t, stage, rollout.Spec.Stage)
		require.Equal(t, freight, rollout.Spec.Freight)
		require.True(t, strings.HasPrefix(rollout.Name, stage+"."))
		require.Equal(
			t,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}},
			rollout.Spec.Targets,
		)
	})

	t.Run("a Target not selected by the Stage is an error", func(t *testing.T) {
		t.Parallel()

		_, err := NewPromotionRolloutForSelectTargets(
			testStage,
			freight,
			stageTargets,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-north"}},
			nil,
		)
		require.Error(t, err)
	})

	t.Run("resolves the update strategy against all Stage Targets", func(t *testing.T) {
		t.Parallel()

		// The strategy resolves against the 10 Stage Targets (50% -> 5), not
		// the single selected Target.
		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("50%"))
		allTargets := make([]kargoapi.Target, 10)
		for i := range allTargets {
			allTargets[i].Name = fmt.Sprintf("us-%d", i)
		}
		rollout, err := NewPromotionRolloutForSelectTargets(
			withStrategy,
			freight,
			allTargets,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-0"}},
			nil,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			kargoapi.PromotionRolloutUpdateStrategy{MaxConcurrent: 5},
			rollout.Spec.UpdateStrategy,
		)
	})

	t.Run("an explicit update strategy overrides the calculation", func(t *testing.T) {
		t.Parallel()

		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("50%"))
		rollout, err := NewPromotionRolloutForSelectTargets(
			withStrategy,
			freight,
			stageTargets,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}},
			&kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: 10,
				MaxFailures:   10,
			},
		)
		require.NoError(t, err)
		require.Equal(
			t,
			kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: 10,
				MaxFailures:   10,
			},
			rollout.Spec.UpdateStrategy,
		)
	})

	t.Run("a Stage without a targets block is an error", func(t *testing.T) {
		t.Parallel()

		classic := testStage.DeepCopy()
		classic.Spec.Targets = nil
		_, err := NewPromotionRolloutForSelectTargets(
			classic,
			freight,
			stageTargets,
			[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}},
			nil,
		)
		require.Error(t, err)
	})
}

func TestPromotionRolloutOf(t *testing.T) {
	t.Parallel()
	require.Empty(t, PromotionRolloutOf(&kargoapi.Promotion{}))
	require.Equal(t, "fake-rollout", PromotionRolloutOf(&kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{kargoapi.AnnotationKeyPromotionRollout: "fake-rollout"},
		},
	}))
}
