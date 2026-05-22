package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

type testErrorResponse struct {
	payload *models.ErrorResponse
}

func (t *testErrorResponse) Error() string {
	return "generated response"
}

func (t *testErrorResponse) GetPayload() *models.ErrorResponse {
	return t.payload
}

func TestFormatAPIError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "uses response payload message",
			err: &testErrorResponse{
				payload: &models.ErrorResponse{
					Error: "auto-promotion hold changed; reload and try again",
				},
			},
			expected: "promote: auto-promotion hold changed; reload and try again",
		},
		{
			name:     "falls back to wrapped error",
			err:      errors.New("transport failed"),
			expected: "promote: transport failed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := FormatAPIError("promote", testCase.err)
			require.EqualError(t, err, testCase.expected)
		})
	}
}
