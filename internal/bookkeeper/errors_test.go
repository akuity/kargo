package bookkeeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrBadRequest(t *testing.T) {
	testCases := []struct {
		name       string
		err        *ErrBadRequest
		assertions func(t *testing.T, err *ErrBadRequest)
	}{
		{
			name: "without details",
			err: &ErrBadRequest{
				Reason: "i don't have to answer to you",
			},
			assertions: func(t *testing.T, err *ErrBadRequest) {
				require.Contains(t, err.Error(), err.Reason)
				for _, detail := range err.Details {
					require.NotContains(t, err.Error(), detail)
				}
			},
		},
		{
			name: "with details",
			err: &ErrBadRequest{
				Reason:  "i don't have to answer to you",
				Details: []string{"the", "devil", "is", "in", "the", "details"},
			},
			assertions: func(t *testing.T, err *ErrBadRequest) {
				require.Contains(t, err.Error(), err.Reason)
				for _, detail := range err.Details {
					require.Contains(t, err.Error(), detail)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, testCase.err)
		})
	}
}

func TestErrNotFound(t *testing.T) {
	err := &ErrNotFound{
		Type: "User",
		ID:   "tony@starkindustries.com",
	}
	require.Contains(t, err.Error(), "not found")
	require.Contains(t, err.Error(), err.Type)
	require.Contains(t, err.Error(), err.ID)
}

func TestErrConflict(t *testing.T) {
	err := &ErrConflict{
		Type:   "User",
		ID:     "tony@starkindustries.com",
		Reason: "i don't have to answer to you",
	}
	require.Contains(t, err.Error(), err.Reason)
}

func TestErrInternalServer(t *testing.T) {
	err := &ErrInternalServer{}
	require.Contains(t, err.Error(), "internal server error")
}

func TestErrNotSupported(t *testing.T) {
	err := &ErrNotSupported{
		Details: "i don't have to answer to you",
	}
	require.Contains(t, err.Error(), err.Details)
}
