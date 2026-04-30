package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestCommon_MarshalAnnotationsTo(t *testing.T) {
	testCases := map[string]struct {
		common   Common
		expected map[string]string
	}{
		"with actor": {
			common: Common{
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject: "test-project",
				kargoapi.AnnotationKeyEventActor:   "test-actor",
			},
		},
		"without actor": {
			common: Common{
				Project: "test-project",
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject: "test-project",
			},
		},
		"empty project": {
			common: Common{
				Project: "",
				Actor:   ptr.To("test-actor"),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventProject: "",
				kargoapi.AnnotationKeyEventActor:   "test-actor",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := make(map[string]string)
			tc.common.MarshalAnnotationsTo(annotations)
			require.Equal(t, tc.expected, annotations)
		})
	}
}

// Safety test to make sure the Kind doesn change on us accidentally
func TestPromotion_Kind(t *testing.T) {
	promotion := Promotion{}
	require.Equal(t, "Promotion", promotion.Kind())
}

func TestPromotion_MarshalAnnotationsTo(t *testing.T) {
	testCases := map[string]struct {
		promotion Promotion
		expected  map[string]string
	}{
		"complete promotion with freight": {
			promotion: Promotion{
				Freight: &Freight{
					Name:       "test-freight",
					StageName:  "test-stage",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					Alias:      ptr.To("v1.0.0"),
				},
				Name:       "test-promotion",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Applications: []types.NamespacedName{
					{Namespace: "argocd", Name: "app1"},
					{Namespace: "argocd", Name: "app2"},
				},
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",

				kargoapi.AnnotationKeyEventApplications:      `[{"Namespace":"argocd","Name":"app1"},{"Namespace":"argocd","Name":"app2"}]`, //nolint:lll
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventFreightAlias:      "v1.0.0",
			},
		},
		"promotion without freight": {
			promotion: Promotion{
				Name:       "test-promotion",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
		},
		"promotion with empty applications": {
			promotion: Promotion{
				Name:         "test-promotion",
				StageName:    "test-stage",
				CreateTime:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Applications: []types.NamespacedName{},
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := make(map[string]string)
			tc.promotion.MarshalAnnotationsTo(annotations)
			require.Equal(t, tc.expected, annotations)
		})
	}
}

// Safety test to make sure the Kind doesn change on us accidentally
func TestFreight_Kind(t *testing.T) {
	freight := Freight{}
	require.Equal(t, "Freight", freight.Kind())
}

func TestFreight_MarshalAnnotationsTo(t *testing.T) {
	testCases := map[string]struct {
		freight  Freight
		expected map[string]string
	}{
		"complete freight": {
			freight: Freight{
				Name:       "test-freight",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Alias:      ptr.To("v1.0.0"),
				Commits:    []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:     []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:     []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightAlias:      "v1.0.0",
				kargoapi.AnnotationKeyEventFreightCommits:    `[{"id":"abc123","tag":"v1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightImages:     `[{"repoURL":"example.com/app","tag":"v1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightCharts:     `[{"name":"my-chart","version":"1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightArtifacts:  `[{"artifactType":"my-type","subscriptionName":"my-sub","version":"v1.0.0"}]`, // nolint: lll
			},
		},
		"minimal freight": {
			freight: Freight{
				Name:       "test-freight",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
			},
		},
		"freight with empty slices": {
			freight: Freight{
				Name:       "test-freight",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Commits:    []kargoapi.GitCommit{},
				Images:     []kargoapi.Image{},
				Charts:     []kargoapi.Chart{},
				Artifacts:  []kargoapi.ArtifactReference{},
			},
			expected: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			annotations := make(map[string]string)
			tc.freight.MarshalAnnotationsTo(annotations)
			require.Equal(t, tc.expected, annotations)
		})
	}
}

func TestUnmarshalCommonAnnotations(t *testing.T) {
	testCases := map[string]struct {
		id          string
		annotations map[string]string
		expected    Common
	}{
		"complete annotations": {
			id: "event-id",
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject: "test-project",
				kargoapi.AnnotationKeyEventActor:   "test-actor",
			},
			expected: Common{
				ID:      "event-id",
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
			},
		},
		"missing actor": {
			id: "event-id",
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventProject: "test-project",
			},
			expected: Common{
				ID:      "event-id",
				Project: "test-project",
			},
		},
		"empty annotations": {
			id:          "event-id",
			annotations: map[string]string{},
			expected: Common{
				ID: "event-id",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalCommonAnnotations(tc.id, tc.annotations)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalPromotionAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations  map[string]string
		expected     Promotion
		expectError  bool
		errorMessage string
	}{
		"complete annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventApplications:        `[{"name":"app1","namespace":"argocd"}]`,
				kargoapi.AnnotationKeyEventFreightName:         "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime:   "2024-01-01T00:00:00Z",
			},
			expected: Promotion{
				Freight: &Freight{
					Name:       "test-freight",
					CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					StageName:  "test-stage",
				},
				Name:       "test-promotion",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Applications: []types.NamespacedName{
					{Name: "app1", Namespace: "argocd"},
				},
			},
		},
		"minimal annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
			},
			expected: Promotion{
				Name:       "test-promotion",
				StageName:  "test-stage",
				CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		"invalid time format": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "invalid-time",
			},
			expectError:  true,
			errorMessage: "failed to parse promotion create time",
		},
		"invalid applications JSON": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventPromotionName:       "test-promotion",
				kargoapi.AnnotationKeyEventStageName:           "test-stage",
				kargoapi.AnnotationKeyEventPromotionCreateTime: "2024-01-01T12:00:00Z",
				kargoapi.AnnotationKeyEventApplications:        `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal applications",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalPromotionAnnotations(tc.annotations)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalFreightAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations  map[string]string
		expected     Freight
		expectError  bool
		errorMessage string
	}{
		"complete annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightAlias:      "v1.0.0",
				kargoapi.AnnotationKeyEventFreightCommits:    `[{"id":"abc123","tag":"v1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightImages:     `[{"repoURL":"example.com/app","tag":"v1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightCharts:     `[{"name":"my-chart","version":"1.0.0"}]`,
				kargoapi.AnnotationKeyEventFreightArtifacts:  `[{"artifactType":"my-type","subscriptionName":"my-sub","version":"v1.0.0"}]`, // nolint: lll
			},
			expected: Freight{
				Name:       "test-freight",
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StageName:  "test-stage",
				Alias:      ptr.To("v1.0.0"),
				Commits:    []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:     []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:     []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
		},
		"minimal annotations": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
			},
			expected: Freight{
				Name:       "test-freight",
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StageName:  "test-stage",
			},
		},
		"invalid time format": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "invalid-time",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
			},
			expectError:  true,
			errorMessage: "failed to parse freight create time",
		},
		"invalid commits JSON": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightCommits:    `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight commits",
		},
		"invalid images JSON": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightImages:     `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight images",
		},
		"invalid charts JSON": {
			annotations: map[string]string{
				kargoapi.AnnotationKeyEventFreightName:       "test-freight",
				kargoapi.AnnotationKeyEventFreightCreateTime: "2024-01-01T00:00:00Z",
				kargoapi.AnnotationKeyEventStageName:         "test-stage",
				kargoapi.AnnotationKeyEventFreightCharts:     `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight charts",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalFreightAnnotations(tc.annotations)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMessage)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewCommonFromPromotion(t *testing.T) {
	testCases := map[string]struct {
		message   string
		actor     string
		promotion *kargoapi.Promotion
		expected  Common
	}{
		"promotion with actor annotation": {
			message: "test message",
			actor:   "external-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: "promotion-actor",
					},
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("promotion-actor"), // annotation takes precedence
			},
		},
		"promotion without actor annotation": {
			message: "test message",
			actor:   "external-actor",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("external-actor"),
			},
		},
		"nil promotion": {
			message:   "test message",
			actor:     "external-actor",
			promotion: nil,
			expected:  Common{},
		},
		"empty actor": {
			message: "test message",
			actor:   "",
			promotion: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newCommonFromPromotion(tc.message, tc.actor, tc.promotion)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewCommonFromFreight(t *testing.T) {
	testCases := map[string]struct {
		message  string
		actor    string
		freight  *kargoapi.Freight
		expected Common
	}{
		"complete freight": {
			message: "test message",
			actor:   "test-actor",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
		},
		"empty actor": {
			message: "test message",
			actor:   "",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			expected: Common{
				Project: "test-project",
				Message: "test message",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newCommonFromFreight(tc.message, tc.actor, tc.freight)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewFreight(t *testing.T) {
	testCases := map[string]struct {
		freight   *kargoapi.Freight
		stageName string
		expected  Freight
	}{
		"complete freight": {
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
				Alias:   "v1.0.0",
				Commits: []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:  []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:  []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
			stageName: "test-stage",
			expected: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
				Alias:      ptr.To("v1.0.0"),
				Commits:    []kargoapi.GitCommit{{ID: "abc123", Tag: "v1.0.0"}},
				Images:     []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.0.0"}},
				Charts:     []kargoapi.Chart{{Name: "my-chart", Version: "1.0.0"}},
				Artifacts: []kargoapi.ArtifactReference{
					{ArtifactType: "my-type", SubscriptionName: "my-sub", Version: "v1.0.0"},
				},
			},
		},
		"minimal freight": {
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			stageName: "test-stage",
			expected: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
			},
		},
		"nil freight": {
			freight:   nil,
			stageName: "test-stage",
			expected:  Freight{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newFreight(tc.freight, tc.stageName)
			require.Equal(t, tc.expected, result)
		})
	}
}
