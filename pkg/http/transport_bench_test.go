package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// BenchmarkIdleTimeoutTransport measures the wrapper's per-request and
// per-Read cost against the bare transport, with an in-memory base so network
// noise doesn't mask it. Sizes span a manifest and a large chart index.
func BenchmarkIdleTimeoutTransport(b *testing.B) {
	transports := []struct {
		name      string
		transport func(http.RoundTripper) http.RoundTripper
	}{
		{
			name:      "base",
			transport: func(rt http.RoundTripper) http.RoundTripper { return rt },
		},
		{
			name: "idle-timeout",
			transport: func(rt http.RoundTripper) http.RoundTripper {
				return NewIdleTimeoutTransport(rt, time.Minute)
			},
		},
	}
	for _, size := range []int{1 << 10, 1 << 20} {
		payload := bytes.Repeat([]byte("x"), size)
		for _, tr := range transports {
			b.Run(fmt.Sprintf("%s/%dKiB", tr.name, size>>10), func(b *testing.B) {
				rt := tr.transport(roundTripperFunc(
					func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							// No io.WriterTo, like a real response body.
							Body: struct {
								io.Reader
								io.Closer
							}{bytes.NewReader(payload), io.NopCloser(nil)},
						}, nil
					},
				))
				req, err := http.NewRequestWithContext(
					b.Context(),
					http.MethodGet,
					"http://example.com",
					nil,
				)
				if err != nil {
					b.Fatal(err)
				}
				buf := make([]byte, 4<<10)
				b.SetBytes(int64(size))
				b.ReportAllocs()
				for b.Loop() {
					res, err := rt.RoundTrip(req)
					if err != nil {
						b.Fatal(err)
					}
					for {
						if _, err = res.Body.Read(buf); err != nil {
							break
						}
					}
					if !errors.Is(err, io.EOF) {
						b.Fatal(err)
					}
					if err = res.Body.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
