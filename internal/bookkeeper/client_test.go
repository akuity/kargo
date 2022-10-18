package bookkeeper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalToError(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		assertions func(t *testing.T, err error)
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			assertions: func(t *testing.T, err error) {
				require.IsType(t, &ErrBadRequest{}, err)
			},
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			assertions: func(t *testing.T, err error) {
				require.IsType(t, &ErrNotFound{}, err)
			},
		},
		{
			name:       "conflict",
			statusCode: http.StatusConflict,
			assertions: func(t *testing.T, err error) {
				require.IsType(t, &ErrConflict{}, err)
			},
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			assertions: func(t *testing.T, err error) {
				require.IsType(t, &ErrInternalServer{}, err)
			},
		},
		{
			name:       "other error",
			statusCode: http.StatusBadGateway,
			assertions: func(t *testing.T, err error) {
				require.Equal(
					t,
					fmt.Sprintf(
						"received %d from Bookkeeper server",
						http.StatusBadGateway,
					),
					err.Error(),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res := &http.Response{
				StatusCode: testCase.statusCode,
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
			}
			defer res.Body.Close()
			client := &client{}
			err := client.unmarshalToError(res)
			testCase.assertions(t, err)
		})
	}
}
