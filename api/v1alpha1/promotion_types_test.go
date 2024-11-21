package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromotionRetry_GetAttempts(t *testing.T) {
	tests := []struct {
		name     string
		retry    *PromotionStepRetry
		fallback int64
		want     int64
	}{
		{
			name:     "retry is nil",
			retry:    nil,
			fallback: 1,
			want:     1,
		},
		{
			name:     "attempts is not set",
			retry:    &PromotionStepRetry{},
			fallback: -1,
			want:     -1,
		},
		{
			name: "attempts is set",
			retry: &PromotionStepRetry{
				Attempts: 3,
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.retry.GetAttempts(tt.fallback))
	}
}
