package v1alpha1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGitCommitDeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *GitCommit
		b              *GitCommit
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoURLs differ",
			a: &GitCommit{
				RepoURL: "foo",
			},
			b: &GitCommit{
				RepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "branch names differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "tags differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "bar",
			},
			expectedResult: false,
		},
		{
			name: "messages differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "bar",
			},
			expectedResult: false,
		},
		{
			name: "authors differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "committers differ",
			a: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "foo",
			},
			b: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			a: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Branch:    "fake-branch",
				Tag:       "fake-tag",
				Message:   "fake-message",
				Author:    "fake-author",
				Committer: "fake-committer",
			},
			b: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Branch:    "fake-branch",
				Tag:       "fake-tag",
				Message:   "fake-message",
				Author:    "fake-author",
				Committer: "fake-committer",
			},
			expectedResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.a.DeepEquals(testCase.b))
			require.Equal(t, testCase.expectedResult, testCase.b.DeepEquals(testCase.a))
		})
	}
}

func TestGitCommitEquals(t *testing.T) {
	testCases := []struct {
		name           string
		lhs            *GitCommit
		rhs            *GitCommit
		expectedResult bool
	}{
		{
			name:           "lhs and rhs both nil",
			expectedResult: true,
		},
		{
			name:           "only lhs is nil",
			rhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only rhs is nil",
			lhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoUrls differ",
			lhs: &GitCommit{
				RepoURL: "foo",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "bar",
				ID:      "fake-commit-id",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			expectedResult: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedResult,
				testCase.lhs.Equals(testCase.rhs),
			)
		})
	}
}

func TestFreight_IsCurrentlyIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsCurrentlyIn(testStage))
	freight.Status.CurrentlyIn = map[string]CurrentStage{testStage: {}}
	require.True(t, freight.IsCurrentlyIn(testStage))
}

func TestFreight_IsVerifiedIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsVerifiedIn(testStage))
	freight.Status.VerifiedIn = map[string]VerifiedStage{testStage: {}}
	require.True(t, freight.IsVerifiedIn(testStage))
}

func TestFreight_IsApprovedFor(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsApprovedFor(testStage))
	freight.Status.ApprovedFor = map[string]ApprovedStage{testStage: {}}
	require.True(t, freight.IsApprovedFor(testStage))
}

func TestFreight_GetLongestSoak(t *testing.T) {
	testStage := "fake-stage"
	testCases := []struct {
		name       string
		status     FreightStatus
		assertions func(t *testing.T, status FreightStatus, longestSoak time.Duration)
	}{
		{
			name: "Freight is not currently in the Stage and was never verified there",
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is not currently in the Stage but was verified there",
			status: FreightStatus{
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Equal(t, time.Hour, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage but was never verified there",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; current soak is longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; a previous soak was longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight := &Freight{
				Status: testCase.status,
			}
			testCase.assertions(t, freight.Status, freight.GetLongestSoak(testStage))
		})
	}
}

func TestFreightStatus_AddCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already in current", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddCurrentStage(testStage, newTime)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, oldTime, record.Since.Time)
	})
	t.Run("not already in current", func(t *testing.T) {
		status := FreightStatus{}
		status.AddCurrentStage(testStage, now)
		require.NotNil(t, status.CurrentlyIn)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, now, record.Since.Time)
	})
}

func TestFreightStatus_RemoveCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	t.Run("not verified", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
	})
	t.Run("verified; old soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, 2*time.Hour, record.LongestCompletedSoak.Duration)
	})
	t.Run("verified; new soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		// Expect these to be equal within a second. TODO(krancour): There's probably a
		// more elegant way to do this, but I consider good enough.
		require.GreaterOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour)
		require.LessOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour+time.Second)
	})
	t.Run("verified; no previous longest soak", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: nil}, // No previous soak time
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.NotNil(t, record.LongestCompletedSoak)
		// Expect the soak time to be approximately 1 hour
		require.GreaterOrEqual(t, record.LongestCompletedSoak.Duration, time.Hour)
		require.LessOrEqual(t, record.LongestCompletedSoak.Duration, time.Hour+time.Second)
	})
}

func TestFreightStatus_AddVerifiedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already verified", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			VerifiedIn: map[string]VerifiedStage{
				testStage: {VerifiedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddVerifiedStage(testStage, newTime)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, oldTime, record.VerifiedAt.Time)
	})
	t.Run("not already verified", func(t *testing.T) {
		status := FreightStatus{}
		testTime := time.Now()
		status.AddVerifiedStage(testStage, testTime)
		require.NotNil(t, status.VerifiedIn)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, testTime, record.VerifiedAt.Time)
	})
}

func TestFreightStatus_AddApprovedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already approved", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			ApprovedFor: map[string]ApprovedStage{
				testStage: {ApprovedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddApprovedStage(testStage, newTime)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, oldTime, record.ApprovedAt.Time)
	})
	t.Run("not already approved", func(t *testing.T) {
		status := FreightStatus{}
		status.AddApprovedStage(testStage, now)
		require.NotNil(t, status.ApprovedFor)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, now, record.ApprovedAt.Time)
	})
}

func TestFreightStatus_UpsertAndGetMetadata_Integration(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{
			name: "string value",
			data: "test-string",
		},
		{
			name: "integer value",
			data: 42,
		},
		{
			name: "boolean value",
			data: true,
		},
		{
			name: "complex struct",
			data: struct {
				Name    string            `json:"name"`
				Age     int               `json:"age"`
				Active  bool              `json:"active"`
				Tags    []string          `json:"tags"`
				Configs map[string]string `json:"configs"`
			}{
				Name:   "test-user",
				Age:    30,
				Active: true,
				Tags:   []string{"admin", "developer"},
				Configs: map[string]string{
					"theme": "dark",
					"lang":  "en",
				},
			},
		},
		{
			name: "slice of integers",
			data: []int{1, 2, 3, 4, 5},
		},
		{
			name: "nested map",
			data: map[string]any{
				"level1": map[string]any{
					"level2": map[string]string{
						"key": "value",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := FreightStatus{}
			key := "integration-test-key"

			// Test Upsert
			err1 := status.UpsertMetadata(key, tt.data)
			require.NoError(t, err1)

			// Verify metadata was stored
			assert.NotNil(t, status.Metadata)
			assert.Contains(t, status.Metadata, key)

			// Test Get with correct type
			switch expected := tt.data.(type) {
			case string:
				var result string
				found, err := status.GetMetadata(key, &result)
				assert.True(t, found)
				assert.NoError(t, err)
				assert.Equal(t, expected, result)

			case int:
				var result int
				found, err := status.GetMetadata(key, &result)
				assert.True(t, found)
				assert.NoError(t, err)
				assert.Equal(t, expected, result)

			case bool:
				var result bool
				found, err := status.GetMetadata(key, &result)
				assert.True(t, found)
				assert.NoError(t, err)
				assert.Equal(t, expected, result)

			case []int:
				var result []int
				found, err := status.GetMetadata(key, &result)
				assert.True(t, found)
				assert.NoError(t, err)
				assert.Equal(t, expected, result)

			default:
				// For complex types, use any and compare JSON representation
				var result any
				found, err := status.GetMetadata(key, &result)
				assert.True(t, found)
				assert.NoError(t, err)

				// Compare by marshaling both to JSON
				expectedJSON, err := json.Marshal(expected)
				require.NoError(t, err)
				resultJSON, err := json.Marshal(result)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedJSON), string(resultJSON))
			}

			// Test updating the same key
			newData := "updated-value"
			err := status.UpsertMetadata(key, newData)
			require.NoError(t, err)

			var updatedResult string
			found, err := status.GetMetadata(key, &updatedResult)
			assert.True(t, found)
			assert.NoError(t, err)
			assert.Equal(t, newData, updatedResult)
		})
	}
}

func TestFreightStatus_MetadataEdgeCases(t *testing.T) {
	t.Run("empty key", func(t *testing.T) {
		status := FreightStatus{}
		err := status.UpsertMetadata("", "value")
		assert.Error(t, err)
	})

	t.Run("nil target for Get", func(t *testing.T) {
		status := FreightStatus{
			Metadata: map[string]apiextensionsv1.JSON{
				"test-key": {Raw: []byte(`"test-value"`)},
			},
		}

		// This should cause a panic or error during unmarshal
		found, err := status.GetMetadata("test-key", nil)
		assert.False(t, found)
		assert.Error(t, err)
	})
}
