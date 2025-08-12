package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/api/v1alpha1"
)

func TestFreightEventMarshalAnnotations(t *testing.T) {
	testCases := map[string]struct {
		actor        string
		freight      *v1alpha1.Freight
		stageName    string
		message      string
		expected     map[string]string
		expectedFunc func(t *testing.T, result map[string]string)
	}{
		"freight with all fields": {
			actor:     "test-user",
			stageName: "test-stage",
			message:   "test message",
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-freight",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Alias:   "v1.2.3",
				Commits: []v1alpha1.GitCommit{{Tag: "v1.2.3", ID: "abc123"}},
				Images:  []v1alpha1.Image{{Tag: "v1.2.3", RepoURL: "example.com/app"}},
				Charts:  []v1alpha1.Chart{{Name: "my-chart", Version: "1.2.3"}},
			},
			expected: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:             "test-user",
				v1alpha1.AnnotationKeyEventFreightAlias:      "v1.2.3",
				v1alpha1.AnnotationKeyEventFreightCommits:    `[{"id":"abc123","tag":"v1.2.3"}]`,
				v1alpha1.AnnotationKeyEventFreightImages:     `[{"repoURL":"example.com/app","tag":"v1.2.3"}]`,
				v1alpha1.AnnotationKeyEventFreightCharts:     `[{"name":"my-chart","version":"1.2.3"}]`,
			},
		},
		"freight with verification timing": {
			actor:     "test-user",
			stageName: "test-stage",
			message:   "verification completed",
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-freight",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
				Alias: "v1.2.3",
			},
			expected: map[string]string{
				v1alpha1.AnnotationKeyEventProject:                "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:            "test-freight",
				v1alpha1.AnnotationKeyEventStageName:              "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime:      "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:                  "test-user",
				v1alpha1.AnnotationKeyEventFreightAlias:           "v1.2.3",
				v1alpha1.AnnotationKeyEventVerificationStartTime:  "2024-10-22T01:00:00Z",
				v1alpha1.AnnotationKeyEventVerificationFinishTime: "2024-10-22T02:00:00Z",
			},
		},
		"minimal freight": {
			actor:     "",
			stageName: "test-stage",
			message:   "minimal test",
			freight: &v1alpha1.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-freight",
					Namespace: "test-namespace",
					CreationTimestamp: metav1.Time{
						Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			expected: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
			},
		},
		"nil freight": {
			actor:     "test-user",
			stageName: "test-stage",
			message:   "nil freight test",
			freight:   nil,
			expectedFunc: func(t *testing.T, result map[string]string) {
				// For nil freight, NewFreightEvent returns an empty FreightEvent,
				// but MarshalAnnotations still creates annotations with empty values
				expectedKeys := []string{
					v1alpha1.AnnotationKeyEventProject,
					v1alpha1.AnnotationKeyEventFreightName,
					v1alpha1.AnnotationKeyEventStageName,
					v1alpha1.AnnotationKeyEventFreightCreateTime,
				}
				for _, key := range expectedKeys {
					require.Contains(t, result, key, "Expected annotation %s to be present", key)
				}
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			evt := NewFreightEvent(tc.actor, tc.freight, tc.stageName, tc.message)
			evt.Message = tc.message

			// Add verification timing for the specific test case
			if name == "freight with verification timing" {
				startTime := time.Date(2024, 10, 22, 1, 0, 0, 0, time.UTC)
				finishTime := time.Date(2024, 10, 22, 2, 0, 0, 0, time.UTC)
				evt.VerificationStartTime = &startTime
				evt.VerificationFinishTime = &finishTime
			}

			result := evt.MarshalAnnotations()

			if tc.expectedFunc != nil {
				tc.expectedFunc(t, result)
				return
			}

			require.Equal(t, len(tc.expected), len(result),
				"Number of annotations doesn't match:\nExpected: %+v\nActual: %+v", tc.expected, result)

			for key, expectedValue := range tc.expected {
				actualValue, exists := result[key]
				require.True(t, exists, "Expected annotation %s not found", key)
				require.Equal(t, expectedValue, actualValue, "Annotation %s value mismatch", key)
			}
		})
	}
}

func TestFreightEventToCloudEvent(t *testing.T) {
	testFreight := &v1alpha1.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-freight",
			Namespace: "test-namespace",
			CreationTimestamp: metav1.Time{
				Time: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		Alias: "v1.2.3",
	}

	evt := NewFreightEvent("test-actor", testFreight, "test-stage", "a message")
	evt.Message = "test message"

	cloudEvent := evt.ToCloudEvent(v1alpha1.EventTypeFreightApproved)

	// Test event type
	expectedType := EventTypePrefix + v1alpha1.EventTypeFreightApproved
	require.Equal(t, string(expectedType), cloudEvent.Type())

	// Test source
	expectedSource := Source("test-namespace", "Freight", "test-freight")
	require.Equal(t, expectedSource, cloudEvent.Source())

	// Test ID is set (should be a UUID)
	require.NotEmpty(t, cloudEvent.ID())

	// Test data
	var eventData FreightEvent
	err := cloudEvent.DataAs(&eventData)
	require.NoError(t, err)
	require.Equal(t, evt, eventData)

	// Test time is set
	require.False(t, cloudEvent.Time().IsZero())
}

func TestUnmarshalFreightEventAnnotations(t *testing.T) {
	testCases := map[string]struct {
		annotations  map[string]string
		expected     FreightEvent
		expectError  bool
		errorMessage string
	}{
		"complete annotations": {
			annotations: map[string]string{
				v1alpha1.AnnotationKeyEventProject:                "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:            "test-freight",
				v1alpha1.AnnotationKeyEventStageName:              "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime:      "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventActor:                  "test-user",
				v1alpha1.AnnotationKeyEventFreightAlias:           "v1.2.3",
				v1alpha1.AnnotationKeyEventFreightCommits:         `[{"tag":"v1.2.3","id":"abc123"}]`,
				v1alpha1.AnnotationKeyEventFreightImages:          `[{"tag":"v1.2.3","repoURL":"example.com/app"}]`,
				v1alpha1.AnnotationKeyEventFreightCharts:          `[{"name":"my-chart","version":"1.2.3"}]`,
				v1alpha1.AnnotationKeyEventVerificationStartTime:  "2024-10-22T01:00:00Z",
				v1alpha1.AnnotationKeyEventVerificationFinishTime: "2024-10-22T02:00:00Z",
			},
			expected: FreightEvent{
				Project:                "test-namespace",
				Name:                   "test-freight",
				StageName:              "test-stage",
				FreightCreateTime:      time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
				Actor:                  stringPtr("test-user"),
				FreightAlias:           stringPtr("v1.2.3"),
				FreightCommits:         []v1alpha1.GitCommit{{ID: "abc123", Tag: "v1.2.3"}},
				FreightImages:          []v1alpha1.Image{{RepoURL: "example.com/app", Tag: "v1.2.3"}},
				FreightCharts:          []v1alpha1.Chart{{Name: "my-chart", Version: "1.2.3"}},
				VerificationStartTime:  timePtr(time.Date(2024, 10, 22, 1, 0, 0, 0, time.UTC)),
				VerificationFinishTime: timePtr(time.Date(2024, 10, 22, 2, 0, 0, 0, time.UTC)),
			},
		},
		"minimal annotations": {
			annotations: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
			},
			expected: FreightEvent{
				Project:           "test-namespace",
				Name:              "test-freight",
				StageName:         "test-stage",
				FreightCreateTime: time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC),
			},
		},
		"invalid commits JSON": {
			annotations: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventFreightCommits:    `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight commits",
		},
		"invalid images JSON": {
			annotations: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventFreightImages:     `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight images",
		},
		"invalid charts JSON": {
			annotations: map[string]string{
				v1alpha1.AnnotationKeyEventProject:           "test-namespace",
				v1alpha1.AnnotationKeyEventFreightName:       "test-freight",
				v1alpha1.AnnotationKeyEventStageName:         "test-stage",
				v1alpha1.AnnotationKeyEventFreightCreateTime: "2024-10-22T00:00:00Z",
				v1alpha1.AnnotationKeyEventFreightCharts:     `invalid json`,
			},
			expectError:  true,
			errorMessage: "failed to unmarshal freight charts",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := UnmarshalFreightEventAnnotations(tc.annotations)

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

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
