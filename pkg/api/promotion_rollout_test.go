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

func TestComparePromotionRolloutPhase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		a        kargoapi.PromotionRolloutPhase
		b        kargoapi.PromotionRolloutPhase
		expected int
	}{
		{
			name:     "Running before Pending",
			a:        kargoapi.PromotionRolloutPhaseRunning,
			b:        kargoapi.PromotionRolloutPhasePending,
			expected: -1,
		},
		{
			name:     "Pending after Running",
			a:        kargoapi.PromotionRolloutPhasePending,
			b:        kargoapi.PromotionRolloutPhaseRunning,
			expected: 1,
		},
		{
			name:     "non-terminal before terminal",
			a:        kargoapi.PromotionRolloutPhasePending,
			b:        kargoapi.PromotionRolloutPhaseSucceeded,
			expected: -1,
		},
		{
			name:     "terminal after non-terminal",
			a:        kargoapi.PromotionRolloutPhaseSucceeded,
			b:        kargoapi.PromotionRolloutPhasePending,
			expected: 1,
		},
		{
			name: "a PromotionRollout without a phase yet is non-terminal",
			a:    "",
			b:    kargoapi.PromotionRolloutPhaseErrored,
			// The reconciler has yet to record a phase, so the rollout still has
			// work ahead of it.
			expected: -1,
		},
		{
			name:     "terminal phases are equal to one another",
			a:        kargoapi.PromotionRolloutPhaseSucceeded,
			b:        kargoapi.PromotionRolloutPhaseFailed,
			expected: 0,
		},
		{
			name:     "identical phases",
			a:        kargoapi.PromotionRolloutPhaseRunning,
			b:        kargoapi.PromotionRolloutPhaseRunning,
			expected: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				ComparePromotionRolloutPhase(testCase.a, testCase.b),
			)
		})
	}
}

func TestComparePromotionRolloutByPhaseAndNumber(t *testing.T) {
	t.Parallel()

	// Generated in this order, so the ULID in older precedes the ULID in newer.
	older := GeneratePromotionRolloutName("test-stage", "fake-freight")
	newer := GeneratePromotionRolloutName("test-stage", "fake-freight")

	request := func(name string, phase kargoapi.PromotionRolloutPhase) kargoapi.PromotionRollout {
		return kargoapi.PromotionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Status:     kargoapi.PromotionRolloutStatus{Phase: phase},
		}
	}
	numbered := func(number int64, phase kargoapi.PromotionRolloutPhase) kargoapi.PromotionRollout {
		return kargoapi.PromotionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "fake-stage.irrelevant"},
			Number:     number,
			Status:     kargoapi.PromotionRolloutStatus{Phase: phase},
		}
	}

	testCases := []struct {
		name       string
		a          kargoapi.PromotionRollout
		b          kargoapi.PromotionRollout
		assertions func(*testing.T, int)
	}{
		{
			name: "phase is compared before name",
			// The Running rollout is the newer of the two, so name order alone
			// would put the other one first.
			a: request(newer, kargoapi.PromotionRolloutPhaseRunning),
			b: request(older, kargoapi.PromotionRolloutPhasePending),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "older of two non-terminal rollouts comes first",
			a:    request(older, kargoapi.PromotionRolloutPhasePending),
			b:    request(newer, kargoapi.PromotionRolloutPhasePending),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "newer of two terminal rollouts comes first",
			a:    request(newer, kargoapi.PromotionRolloutPhaseSucceeded),
			b:    request(older, kargoapi.PromotionRolloutPhaseFailed),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "numbers decide before names",
			// Number 1 was created first however the names compare.
			a: numbered(1, kargoapi.PromotionRolloutPhasePending),
			b: numbered(2, kargoapi.PromotionRolloutPhasePending),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "higher-numbered of two terminal rollouts comes first",
			a:    numbered(2, kargoapi.PromotionRolloutPhaseSucceeded),
			b:    numbered(1, kargoapi.PromotionRolloutPhaseFailed),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assertions(
				t,
				ComparePromotionRolloutByPhaseAndNumber(testCase.a, testCase.b),
			)
			// The comparator must be antisymmetric, or slices.SortFunc gives no
			// guarantees about the order it produces.
			forward := ComparePromotionRolloutByPhaseAndNumber(testCase.a, testCase.b)
			reverse := ComparePromotionRolloutByPhaseAndNumber(testCase.b, testCase.a)
			require.Equal(t, forward, -reverse)
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

	t.Run("records the Stage and the Freight, and no Targets yet", func(t *testing.T) {
		t.Parallel()

		rollout := NewPromotionRollout(testStage, freight)

		require.Equal(t, project, rollout.Namespace)
		require.Equal(t, stage, rollout.Spec.Stage)
		require.Equal(t, freight, rollout.Spec.Freight)
		require.True(t, strings.HasPrefix(rollout.Name, stage+"."))
		// The reconciler resolves the Targets once it sees the rollout.
		require.Nil(t, rollout.Spec.Targets)
	})

	t.Run("carries no owner references or labels", func(t *testing.T) {
		t.Parallel()

		// A PromotionRollout is not a Kubernetes resource: nothing garbage
		// collects it and no controller filters it by label.
		rollout := NewPromotionRollout(testStage, freight)
		require.Empty(t, rollout.OwnerReferences)
		require.Empty(t, rollout.Labels)
	})

	t.Run("records the Stage's update strategy as given", func(t *testing.T) {
		t.Parallel()

		withStrategy := testStage.DeepCopy()
		withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent = new(intstr.FromString("50%"))
		withStrategy.Spec.Targets.UpdateStrategy.MaxFailures = new(intstr.FromInt32(2))
		rollout := NewPromotionRollout(withStrategy, freight)
		// A percentage is carried as such: it counts the rollout's Targets,
		// which are resolved only later.
		require.Equal(t, new(intstr.FromString("50%")), rollout.Spec.UpdateStrategy.MaxConcurrent)
		require.Equal(t, new(intstr.FromInt32(2)), rollout.Spec.UpdateStrategy.MaxFailures)
		// Copies, not the Stage's own values.
		require.NotSame(
			t,
			withStrategy.Spec.Targets.UpdateStrategy.MaxConcurrent,
			rollout.Spec.UpdateStrategy.MaxConcurrent,
		)
	})

	t.Run("a Stage without an update strategy gives the rollout none", func(t *testing.T) {
		t.Parallel()
		rollout := NewPromotionRollout(testStage, freight)
		require.Zero(t, rollout.Spec.UpdateStrategy)
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

func TestPromotionRolloutTargetStatuses(t *testing.T) {
	t.Parallel()

	const rollout = "fake-stage.01ROLLOUT.abcdef1"
	child := func(name, target, of string, phase kargoapi.PromotionPhase) kargoapi.Promotion {
		return kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: map[string]string{kargoapi.AnnotationKeyPromotionRollout: of},
			},
			Spec:   kargoapi.PromotionSpec{Stage: "fake-stage", Target: target},
			Status: kargoapi.PromotionStatus{Phase: phase},
		}
	}

	testCases := []struct {
		name       string
		promos     []kargoapi.Promotion
		assertions func(*testing.T, []kargoapi.PromotionRolloutTargetStatus, *kargoapi.PromotionRolloutSummary)
	}{
		{
			name: "no Promotions",
			assertions: func(
				t *testing.T,
				statuses []kargoapi.PromotionRolloutTargetStatus,
				summary *kargoapi.PromotionRolloutSummary,
			) {
				require.Nil(t, statuses)
				require.Nil(t, summary)
			},
		},
		{
			name: "Promotions of other rollouts and to no Target are ignored",
			promos: []kargoapi.Promotion{
				child("fake-stage.us-east.01A.abc", "us-east", "other-rollout", kargoapi.PromotionPhaseSucceeded),
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "fake-stage.01B.abc",
						Annotations: map[string]string{kargoapi.AnnotationKeyPromotionRollout: rollout},
					},
					Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded},
				},
			},
			assertions: func(
				t *testing.T,
				statuses []kargoapi.PromotionRolloutTargetStatus,
				summary *kargoapi.PromotionRolloutSummary,
			) {
				require.Nil(t, statuses)
				require.Nil(t, summary)
			},
		},
		{
			name: "one entry per Target, the newest Promotion to it, in Target order",
			promos: []kargoapi.Promotion{
				child("fake-stage.us-west.01A.abc", "us-west", rollout, kargoapi.PromotionPhaseRunning),
				// A retry: the newer of the two Promotions to us-east wins.
				child("fake-stage.us-east.01C.abc", "us-east", rollout, kargoapi.PromotionPhaseSucceeded),
				child("fake-stage.us-east.01A.abc", "us-east", rollout, kargoapi.PromotionPhaseFailed),
				child("fake-stage.eu-west.01A.abc", "eu-west", rollout, ""),
			},
			assertions: func(
				t *testing.T,
				statuses []kargoapi.PromotionRolloutTargetStatus,
				summary *kargoapi.PromotionRolloutSummary,
			) {
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "eu-west", Promotion: "fake-stage.eu-west.01A.abc"},
					{Name: "us-east", Promotion: "fake-stage.us-east.01C.abc", Phase: kargoapi.PromotionPhaseSucceeded},
					{Name: "us-west", Promotion: "fake-stage.us-west.01A.abc", Phase: kargoapi.PromotionPhaseRunning},
				}, statuses)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{Pending: 1, Running: 1, Succeeded: 1}, summary)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			statuses, summary := PromotionRolloutTargetStatuses(rollout, testCase.promos)
			testCase.assertions(t, statuses, summary)
		})
	}
}
