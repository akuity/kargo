package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKargoSubject(t *testing.T) {
	require.Equal(t, "akuity.kargo.my-topic", NewKargoSubject("my-topic"))
}

func TestNewEventsSubjectPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		kind     string
		expected string
	}{
		{
			name:     "with kind",
			kind:     "PromotionRequest",
			expected: "akuity.kargo.events.promotionrequest",
		},
		{
			name:     "without kind",
			expected: "akuity.kargo.events.GLOBAL",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, NewEventsSubjectPrefix(testCase.kind))
		})
	}
}
