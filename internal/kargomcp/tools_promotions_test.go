package kargomcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterRawsByPhase(t *testing.T) {
	t.Parallel()
	raw := func(phase string) json.RawMessage {
		b, _ := json.Marshal(promotionJSON{
			Status: struct {
				Phase      string `json:"phase"`
				Message    string `json:"message"`
				StartedAt  string `json:"startedAt"`
				FinishedAt string `json:"finishedAt"`
			}{Phase: phase},
		})
		return b
	}

	testCases := []struct {
		name   string
		input  []json.RawMessage
		phase  string
		assert func(*testing.T, []json.RawMessage)
	}{
		{
			name:   "empty input",
			input:  nil,
			phase:  "Running",
			assert: func(t *testing.T, got []json.RawMessage) { require.Empty(t, got) },
		},
		{
			name:  "exact phase match",
			input: []json.RawMessage{raw("Running"), raw("Succeeded"), raw("Running")},
			phase: "Running",
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
		{
			name:  "case-insensitive match",
			input: []json.RawMessage{raw("Errored"), raw("Succeeded")},
			phase: "errored",
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 1)
			},
		},
		{
			name:  "no matches returns nil",
			input: []json.RawMessage{raw("Succeeded")},
			phase: "Running",
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Empty(t, got)
			},
		},
		{
			name:  "invalid JSON skipped",
			input: []json.RawMessage{raw("Errored"), []byte("bad"), raw("Errored")},
			phase: "Errored",
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := filterRawsByPhase(tc.input, tc.phase)
			tc.assert(t, got)
		})
	}
}

func TestPromotionToSummary(t *testing.T) {
	t.Parallel()
	p := promotionJSON{}
	p.Metadata.Name = "promo-1"
	p.Spec.Stage = "prod"
	p.Spec.Freight = "abc123"
	p.Status.Phase = "Succeeded"
	p.Status.Message = "done"
	p.Status.StartedAt = "2026-01-01T00:00:00Z"
	p.Status.FinishedAt = "2026-01-01T00:01:00Z"

	s := promotionToSummary(p)
	require.Equal(t, "promo-1", s.Name)
	require.Equal(t, "prod", s.Stage)
	require.Equal(t, "abc123", s.Freight)
	require.Equal(t, "Succeeded", s.Phase)
	require.Equal(t, "done", s.Message)
	require.Equal(t, "2026-01-01T00:00:00Z", s.StartedAt)
	require.Equal(t, "2026-01-01T00:01:00Z", s.FinishedAt)
}
