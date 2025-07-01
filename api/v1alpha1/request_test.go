package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAbortPromotionRequest_Equals(t *testing.T) {
	tests := []struct {
		name     string
		r1       *AbortPromotionRequest
		r2       *AbortPromotionRequest
		expected bool
	}{
		{
			name:     "both nil",
			r1:       nil,
			r2:       nil,
			expected: true,
		},
		{
			name:     "one nil",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			r2:       nil,
			expected: false,
		},
		{
			name:     "other nil",
			r1:       nil,
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actions",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			r2:       &AbortPromotionRequest{Action: "other-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actors",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "other-actor", ControlPlane: true},
			expected: false,
		},
		{
			name:     "different control plane flags",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "equal",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.r1.Equals(tt.r2), tt.expected)
		})
	}
}

func TestAbortPromotionRequest_String(t *testing.T) {
	t.Run("abort request is nil", func(t *testing.T) {
		var r *AbortPromotionRequest
		require.Empty(t, r.String())
	})

	t.Run("abort request is empty", func(t *testing.T) {
		r := &AbortPromotionRequest{}
		require.Empty(t, r.String())
	})

	t.Run("abort request has empty action", func(t *testing.T) {
		r := &AbortPromotionRequest{
			Action: "",
		}
		require.Empty(t, r.String())
	})

	t.Run("abort request has data", func(t *testing.T) {
		r := &AbortPromotionRequest{
			Action:       "foo",
			Actor:        "fake-actor",
			ControlPlane: true,
		}
		require.Equal(t, `{"action":"foo","actor":"fake-actor","controlPlane":true}`, r.String())
	})
}

func TestVerificationRequest_Equals(t *testing.T) {
	tests := []struct {
		name     string
		r1       *VerificationRequest
		r2       *VerificationRequest
		expected bool
	}{
		{
			name:     "both nil",
			r1:       nil,
			r2:       nil,
			expected: true,
		},
		{
			name:     "one nil",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			r2:       nil,
			expected: false,
		},
		{
			name:     "other nil",
			r1:       nil,
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different IDs",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			r2:       &VerificationRequest{ID: "other-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actors",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "other-actor", ControlPlane: true},
			expected: false,
		},
		{
			name:     "different control plane flags",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "equal",
			r1:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			r2:       &VerificationRequest{ID: "fake-id", Actor: "fake-actor", ControlPlane: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.r1.Equals(tt.r2), tt.expected)
		})
	}
}

func TestVerificationRequest_HasID(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.False(t, r.HasID())
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.False(t, r.HasID())
	})

	t.Run("verification request has ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "foo",
		}
		require.True(t, r.HasID())
	})
}

func TestVerificationRequest_ForID(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.False(t, r.ForID("foo"))
	})

	t.Run("verification request has ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "foo",
		}
		require.True(t, r.ForID("foo"))
		require.False(t, r.ForID("bar"))
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.False(t, r.ForID(""))
		require.False(t, r.ForID("foo"))
	})
}

func TestVerificationRequest_String(t *testing.T) {
	t.Run("verification request is nil", func(t *testing.T) {
		var r *VerificationRequest
		require.Empty(t, r.String())
	})

	t.Run("verification request is empty", func(t *testing.T) {
		r := &VerificationRequest{}
		require.Empty(t, r.String())
	})

	t.Run("verification request has empty ID", func(t *testing.T) {
		r := &VerificationRequest{
			ID: "",
		}
		require.Empty(t, r.String())
	})

	t.Run("verification request has data", func(t *testing.T) {
		r := &VerificationRequest{
			ID:           "foo",
			Actor:        "fake-actor",
			ControlPlane: true,
		}
		require.Equal(t, `{"id":"foo","actor":"fake-actor","controlPlane":true}`, r.String())
	})
}
