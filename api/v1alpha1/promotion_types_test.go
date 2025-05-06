package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestPromotionStepStatus_Compare(t *testing.T) {
	tests := []struct {
		name     string
		lhs      PromotionStepStatus
		rhs      PromotionStepStatus
		expected int
	}{
		{
			name:     "Succeeded < Skipped",
			lhs:      PromotionStepStatusSucceeded,
			rhs:      PromotionStepStatusSkipped,
			expected: -1,
		},
		{
			name:     "Skipped < Running",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusRunning,
			expected: -1,
		},
		{
			name:     "Running < Aborted",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusAborted,
			expected: -1,
		},
		{
			name:     "Aborted < Failed",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusFailed,
			expected: -1,
		},
		{
			name:     "Failed < Errored",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusErrored,
			expected: -1,
		},
		{
			name:     "Succeeded == Succeeded",
			lhs:      PromotionStepStatusSucceeded,
			rhs:      PromotionStepStatusSucceeded,
			expected: 0,
		},
		{
			name:     "Skipped == Skipped",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusSkipped,
			expected: 0,
		},
		{
			name:     "Running == Running",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusRunning,
			expected: 0,
		},
		{
			name:     "Aborted == Aborted",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusAborted,
			expected: 0,
		},
		{
			name:     "Failed == Failed",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusFailed,
			expected: 0,
		},
		{
			name:     "Errored == Errored",
			lhs:      PromotionStepStatusErrored,
			rhs:      PromotionStepStatusErrored,
			expected: 0,
		},
		{
			name:     "Skipped > Succeeded",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusSucceeded,
			expected: 1,
		},
		{
			name:     "Running > Skipped",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusSkipped,
			expected: 1,
		},
		{
			name:     "Aborted > Running",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusRunning,
			expected: 1,
		},
		{
			name:     "Failed > Aborted",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusAborted,
			expected: 1,
		},
		{
			name:     "Errored > Failed",
			lhs:      PromotionStepStatusErrored,
			rhs:      PromotionStepStatusFailed,
			expected: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lhs.Compare(tt.rhs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPromotionRetry_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		retry    *PromotionStepRetry
		fallback *time.Duration
		want     *time.Duration
	}{
		{
			name:     "retry is nil",
			retry:    nil,
			fallback: ptr.To(time.Hour),
			want:     ptr.To(time.Hour),
		},
		{
			name:     "timeout is not set",
			retry:    &PromotionStepRetry{},
			fallback: ptr.To(time.Hour),
			want:     ptr.To(time.Hour),
		},
		{
			name: "timeout is set",
			retry: &PromotionStepRetry{
				Timeout: &metav1.Duration{
					Duration: 3 * time.Hour,
				},
			},
			want: ptr.To(3 * time.Hour),
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.retry.GetTimeout(tt.fallback))
	}
}

func TestPromotionRetry_GetErrorThreshold(t *testing.T) {
	tests := []struct {
		name     string
		retry    *PromotionStepRetry
		fallback uint32
		want     uint32
	}{
		{
			name:     "retry is nil",
			retry:    nil,
			fallback: 1,
			want:     1,
		},
		{
			name:     "threshold is not set",
			retry:    &PromotionStepRetry{},
			fallback: 1,
			want:     1,
		},
		{
			name: "threshold is set",
			retry: &PromotionStepRetry{
				ErrorThreshold: 3,
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.retry.GetErrorThreshold(tt.fallback))
	}
}

func TestStepExecutionMetadataList_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		metadata StepExecutionMetadataList
		expected bool
	}{
		{
			name: "no errors/failures at all",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
		{
			name: "has an error",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusErrored},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: true,
		},
		{
			name: "has an error with continueOnError == true",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{
					ContinueOnError: true,
					Status:          PromotionStepStatusErrored,
				},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
		{
			name: "has a failure",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusFailed},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: true,
		},
		{
			name: "has a failure with continueOnError == true",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{
					ContinueOnError: true,
					Status:          PromotionStepStatusFailed,
				},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.metadata.HasFailures())
		})
	}
}
