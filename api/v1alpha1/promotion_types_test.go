package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

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
