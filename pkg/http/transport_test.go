package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIdleTimeoutTransport(t *testing.T) {
	const chunk = "data"
	testCases := []struct {
		name               string
		timeout            time.Duration
		handler            http.HandlerFunc
		cancelAfterHeaders bool
		assert             func(*testing.T, []byte, error)
	}{
		{
			name:    "body stalls before first byte",
			timeout: 50 * time.Millisecond,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = http.NewResponseController(w).Flush()
				<-r.Context().Done()
			},
			assert: func(t *testing.T, data []byte, err error) {
				var idleErr idleTimeoutError
				require.ErrorAs(t, err, &idleErr)
				require.ErrorContains(t, err, "no response data received for 50ms")
				require.Empty(t, data)
			},
		},
		{
			name:    "body stalls after first byte",
			timeout: 50 * time.Millisecond,
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(chunk))
				_ = http.NewResponseController(w).Flush()
				<-r.Context().Done()
			},
			assert: func(t *testing.T, data []byte, err error) {
				var idleErr idleTimeoutError
				require.ErrorAs(t, err, &idleErr)
				require.Equal(t, chunk, string(data))
			},
		},
		{
			name:    "body progresses past timeout in total",
			timeout: 500 * time.Millisecond,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				for range 60 {
					_, _ = w.Write([]byte(chunk))
					_ = http.NewResponseController(w).Flush()
					time.Sleep(10 * time.Millisecond)
				}
			},
			assert: func(t *testing.T, data []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, strings.Repeat(chunk, 60), string(data))
			},
		},
		{
			name:    "caller cancels",
			timeout: time.Minute,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = http.NewResponseController(w).Flush()
				<-r.Context().Done()
			},
			cancelAfterHeaders: true,
			assert: func(t *testing.T, _ []byte, err error) {
				require.ErrorIs(t, err, context.Canceled)
				var idleErr idleTimeoutError
				require.False(t, errors.As(err, &idleErr))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(testCase.handler)
			t.Cleanup(srv.Close)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			client := &http.Client{
				Transport: NewIdleTimeoutTransport(
					http.DefaultTransport,
					testCase.timeout,
				),
			}
			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			if testCase.cancelAfterHeaders {
				cancel()
			}
			data, err := io.ReadAll(res.Body)
			testCase.assert(t, data, err)
		})
	}
}

func TestIdleTimeoutTransport_FakeBase(t *testing.T) {
	const timeout = 20 * time.Millisecond
	testCases := []struct {
		name   string
		body   io.ReadCloser
		assert func(*testing.T, *http.Response, context.Context)
	}{
		{
			name: "nil body passes through",
			assert: func(t *testing.T, res *http.Response, reqCtx context.Context) {
				require.Nil(t, res.Body)
				require.ErrorIs(t, reqCtx.Err(), context.Canceled)
			},
		},
		{
			name: "NoBody passes through",
			body: http.NoBody,
			assert: func(t *testing.T, res *http.Response, reqCtx context.Context) {
				require.Equal(t, http.NoBody, res.Body)
				require.ErrorIs(t, reqCtx.Err(), context.Canceled)
			},
		},
		{
			name: "read to EOF cancels request context",
			body: io.NopCloser(strings.NewReader("data")),
			assert: func(t *testing.T, res *http.Response, reqCtx context.Context) {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				require.Equal(t, "data", string(data))
				require.ErrorIs(t, reqCtx.Err(), context.Canceled)
				require.ErrorIs(t, context.Cause(reqCtx), context.Canceled)
			},
		},
		{
			name: "caller pauses past timeout before EOF",
			body: io.NopCloser(strings.NewReader("data")),
			assert: func(t *testing.T, res *http.Response, _ context.Context) {
				buf := make([]byte, 4)
				n, err := res.Body.Read(buf)
				require.NoError(t, err)
				require.Equal(t, 4, n)
				time.Sleep(2 * timeout)
				n, err = res.Body.Read(buf)
				require.Zero(t, n)
				require.Same(t, io.EOF, err)
			},
		},
		{
			name: "Close stops the timer and cancels the request",
			body: io.NopCloser(strings.NewReader("data")),
			assert: func(t *testing.T, res *http.Response, reqCtx context.Context) {
				require.NoError(t, res.Body.Close())
				require.ErrorIs(t, reqCtx.Err(), context.Canceled)
				time.Sleep(2 * timeout)
				require.ErrorIs(t, context.Cause(reqCtx), context.Canceled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var reqCtx context.Context
			transport := NewIdleTimeoutTransport(
				roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					reqCtx = req.Context()
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       testCase.body,
					}, nil
				}),
				timeout,
			)
			req, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"http://example.com",
				nil,
			)
			require.NoError(t, err)
			res, err := transport.RoundTrip(req)
			require.NoError(t, err)
			if res.Body != nil {
				defer res.Body.Close()
			}
			testCase.assert(t, res, reqCtx)
		})
	}
}

func TestIdleTimeoutTransport_RoundTripError(t *testing.T) {
	t.Parallel()
	errTest := errors.New("round trip failed")
	var reqCtx context.Context
	transport := NewIdleTimeoutTransport(
		roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			reqCtx = req.Context()
			return nil, errTest
		}),
		time.Minute,
	)
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://example.com",
		nil,
	)
	require.NoError(t, err)
	res, err := transport.RoundTrip(req) // nolint: bodyclose
	require.ErrorIs(t, err, errTest)
	require.Nil(t, res)
	require.ErrorIs(t, reqCtx.Err(), context.Canceled)
}

func TestIdleTimeoutTransport_WrappedRoundTripper(t *testing.T) {
	t.Parallel()
	next := &http.Transport{}
	transport := NewIdleTimeoutTransport(next, time.Minute)
	wrapper, ok := transport.(interface {
		WrappedRoundTripper() http.RoundTripper
	})
	require.True(t, ok)
	require.Same(t, next, wrapper.WrappedRoundTripper())
}
