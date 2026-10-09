package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGeneratePromotionRequestName(t *testing.T) {
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
			testCase.assertions(t, GeneratePromotionRequestName(testCase.stageName, testCase.freight))
		})
	}
}

func TestGenerateChildPromotionName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		stageName  string
		targetName string
		freight    string
		assertions func(*testing.T, string)
	}{
		{
			name:       "empty Target name",
			stageName:  "fake-stage",
			targetName: "",
			freight:    "fake-freight",
			assertions: func(t *testing.T, name string) {
				require.Empty(t, name)
			},
		},
		{
			name:       "empty Stage name",
			stageName:  "",
			targetName: "fake-target",
			freight:    "fake-freight",
			assertions: func(t *testing.T, name string) {
				require.Empty(t, name)
			},
		},
		{
			name:       "Stage and Target name",
			stageName:  "prod",
			targetName: "us-east",
			freight:    "abcdef1234567890",
			assertions: func(t *testing.T, name string) {
				parts := strings.Split(name, ".")
				require.Len(t, parts, 4)
				require.Equal(t, "prod", parts[0])
				require.Equal(t, "us-east", parts[1])
				require.Equal(t, "abcdef1", parts[3])
			},
		},
		{
			name:       "repeated calls do not collide",
			stageName:  "prod",
			targetName: "us-east",
			freight:    "abcdef1234567890",
			assertions: func(t *testing.T, name string) {
				other := GenerateChildPromotionName("prod", "us-east", "abcdef1234567890")
				require.NotEqual(t, name, other)
			},
		},
		{
			name:       "over-long names are truncated to a valid resource name",
			stageName:  strings.Repeat("a", 200),
			targetName: strings.Repeat("b", 200),
			freight:    "abcdef1234567890",
			assertions: func(t *testing.T, name string) {
				require.LessOrEqual(t, len(name), 253)
				// Truncation must not leave a separator or hyphen adjacent to
				// the ULID, which would make the name invalid.
				require.NotContains(t, name, "..")
				require.NotContains(t, name, "-.")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assertions(t, GenerateChildPromotionName(
				testCase.stageName,
				testCase.targetName,
				testCase.freight,
			))
		})
	}
}

func TestComparePromotionRequestPhase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		a        kargoapi.PromotionRequestPhase
		b        kargoapi.PromotionRequestPhase
		expected int
	}{
		{
			name:     "Running before Pending",
			a:        kargoapi.PromotionRequestPhaseRunning,
			b:        kargoapi.PromotionRequestPhasePending,
			expected: -1,
		},
		{
			name:     "Pending after Running",
			a:        kargoapi.PromotionRequestPhasePending,
			b:        kargoapi.PromotionRequestPhaseRunning,
			expected: 1,
		},
		{
			name:     "non-terminal before terminal",
			a:        kargoapi.PromotionRequestPhasePending,
			b:        kargoapi.PromotionRequestPhaseSucceeded,
			expected: -1,
		},
		{
			name:     "terminal after non-terminal",
			a:        kargoapi.PromotionRequestPhaseSucceeded,
			b:        kargoapi.PromotionRequestPhasePending,
			expected: 1,
		},
		{
			name: "a PromotionRequest without a phase yet is non-terminal",
			a:    "",
			b:    kargoapi.PromotionRequestPhaseErrored,
			// The reconciler has yet to record a phase, so the request still has
			// work ahead of it.
			expected: -1,
		},
		{
			name:     "terminal phases are equal to one another",
			a:        kargoapi.PromotionRequestPhaseSucceeded,
			b:        kargoapi.PromotionRequestPhaseFailed,
			expected: 0,
		},
		{
			name:     "identical phases",
			a:        kargoapi.PromotionRequestPhaseRunning,
			b:        kargoapi.PromotionRequestPhaseRunning,
			expected: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				ComparePromotionRequestPhase(testCase.a, testCase.b),
			)
		})
	}
}

func TestComparePromotionRequestByPhaseAndCreationTime(t *testing.T) {
	t.Parallel()

	// Generated in this order, so the ULID in older precedes the ULID in newer.
	older := GeneratePromotionRequestName("test-stage", "fake-freight")
	newer := GeneratePromotionRequestName("test-stage", "fake-freight")

	request := func(name string, phase kargoapi.PromotionRequestPhase) kargoapi.PromotionRequest {
		return kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Status:     kargoapi.PromotionRequestStatus{Phase: phase},
		}
	}

	testCases := []struct {
		name       string
		a          kargoapi.PromotionRequest
		b          kargoapi.PromotionRequest
		assertions func(*testing.T, int)
	}{
		{
			name: "phase is compared before name",
			// The Running request is the newer of the two, so name order alone
			// would put the other one first.
			a: request(newer, kargoapi.PromotionRequestPhaseRunning),
			b: request(older, kargoapi.PromotionRequestPhasePending),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "older of two non-terminal requests comes first",
			a:    request(older, kargoapi.PromotionRequestPhasePending),
			b:    request(newer, kargoapi.PromotionRequestPhasePending),
			assertions: func(t *testing.T, result int) {
				require.Negative(t, result)
			},
		},
		{
			name: "newer of two terminal requests comes first",
			a:    request(newer, kargoapi.PromotionRequestPhaseSucceeded),
			b:    request(older, kargoapi.PromotionRequestPhaseFailed),
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
				ComparePromotionRequestByPhaseAndCreationTime(testCase.a, testCase.b),
			)
			// The comparator must be antisymmetric, or slices.SortFunc gives no
			// guarantees about the order it produces.
			forward := ComparePromotionRequestByPhaseAndCreationTime(testCase.a, testCase.b)
			reverse := ComparePromotionRequestByPhaseAndCreationTime(testCase.b, testCase.a)
			require.Equal(t, forward, -reverse)
		})
	}
}

func TestNewPromotionRequest(t *testing.T) {
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

	newTarget := func(name string) kargoapi.Target {
		return kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: name},
		}
	}
	usEast := newTarget("us-east")
	usWest := newTarget("us-west")

	t.Run("records the Stage, the Freight and the Targets it was given", func(t *testing.T) {
		t.Parallel()

		promoReq := NewPromotionRequest(testStage, freight, []kargoapi.Target{usEast, usWest})

		require.Equal(t, project, promoReq.Namespace)
		require.Equal(t, stage, promoReq.Spec.Stage)
		require.Equal(t, freight, promoReq.Spec.Freight)
		require.True(t, strings.HasPrefix(promoReq.Name, stage+"."))
		// In the order given: the caller has already filtered and sorted.
		require.Equal(
			t,
			[]kargoapi.PromotionRequestTarget{
				{Name: "us-east"},
				{Name: "us-west"},
			},
			promoReq.Spec.Targets,
		)
	})

	t.Run("the Stage is the controlling owner", func(t *testing.T) {
		t.Parallel()

		promoReq := NewPromotionRequest(testStage, freight, []kargoapi.Target{usEast})

		require.Len(t, promoReq.OwnerReferences, 1)
		ownerRef := promoReq.OwnerReferences[0]
		require.Equal(t, stage, ownerRef.Name)
		require.Equal(t, "Stage", ownerRef.Kind)
		require.Equal(t, kargoapi.GroupVersion.String(), ownerRef.APIVersion)
		require.NotNil(t, ownerRef.Controller)
		require.True(t, *ownerRef.Controller)
	})

	t.Run("identifying labels", func(t *testing.T) {
		t.Parallel()

		promoReq := NewPromotionRequest(testStage, freight, []kargoapi.Target{usEast})
		require.Equal(t, stage, promoReq.Labels[kargoapi.LabelKeyStage])
		// Without the shard label, only the default controller would ever
		// reconcile this PromotionRequest.
		require.Equal(t, "my-shard", promoReq.Labels[kargoapi.LabelKeyShard])
	})

	t.Run("an unsharded Stage yields no shard label", func(t *testing.T) {
		t.Parallel()

		unsharded := testStage.DeepCopy()
		unsharded.Spec.Shard = ""
		promoReq := NewPromotionRequest(unsharded, freight, []kargoapi.Target{usEast})
		require.NotContains(t, promoReq.Labels, kargoapi.LabelKeyShard)
	})

	t.Run("no Targets yield an empty list, not nil", func(t *testing.T) {
		t.Parallel()

		// spec.targets is required and has no omitempty, so a nil slice would
		// serialize as null and be rejected by the API server. An empty list
		// records that the Stage governed no Targets at this moment.
		promoReq := NewPromotionRequest(testStage, freight, nil)
		require.NotNil(t, promoReq.Spec.Targets)
		require.Empty(t, promoReq.Spec.Targets)
	})
}

// Two separate properties, and only the first one holds.
//
// Promotion ordering elsewhere in Kargo compares whole names and treats the
// result as creation order -- see ComparePromotionByPhaseAndCreationTime, whose
// output decides which Promotion a Stage marks as current. That is sound only
// while the names being compared share everything to the left of the ULID.
// Naming a child after its Target breaks that for two children of different
// Targets, so the property is pinned here rather than assumed.
func TestGenerateChildPromotionNameOrdering(t *testing.T) {
	t.Parallel()

	const (
		stage   = "fake-stage"
		freight = "abc1234567890"
	)

	t.Run("same Target: lex order is creation order", func(t *testing.T) {
		t.Parallel()

		// ulid.Make draws from a process-wide monotonic entropy source, so ULIDs
		// minted within one millisecond still increase. Enough names to be sure
		// of landing several in the same millisecond.
		const n = 50
		names := make([]string, n)
		for i := range names {
			names[i] = GenerateChildPromotionName(stage, "us-east", freight)
		}
		for i := 1; i < n; i++ {
			require.Less(
				t, names[i-1], names[i],
				"name %d sorts before its predecessor; lex order no longer tracks creation order",
			)
		}
	})

	t.Run("different Targets: lex order is Target order, not creation order", func(t *testing.T) {
		t.Parallel()

		// Created second, but sorts first, because comparison resolves at the
		// Target segment and never reaches the ULID.
		west := GenerateChildPromotionName(stage, "us-west", freight)
		east := GenerateChildPromotionName(stage, "us-east", freight)
		require.Less(t, east, west)
	})
}
