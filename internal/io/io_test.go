package io

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitRead(t *testing.T) {
	const maxBytes = 2 << 20 // 2MB

	tests := []struct {
		name       string
		reader     io.ReadCloser
		limit      int64
		assertions func(t *testing.T, data []byte, err error)
	}{
		{
			name:   "small content within limit",
			reader: io.NopCloser(strings.NewReader("hello there")),
			limit:  maxBytes,
			assertions: func(t *testing.T, data []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, "hello there", string(data))
			},
		},
		{
			name:   "empty content",
			reader: io.NopCloser(strings.NewReader("")),
			limit:  maxBytes,
			assertions: func(t *testing.T, data []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, "", string(data))
			},
		},
		{
			name:   "content exactly at limit",
			reader: io.NopCloser(strings.NewReader("hello")),
			limit:  5,
			assertions: func(t *testing.T, data []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, "hello", string(data))
			},
		},
		{
			name:   "content exceeds limit by one byte",
			reader: io.NopCloser(strings.NewReader("hello!")),
			limit:  5,
			assertions: func(t *testing.T, _ []byte, err error) {
				require.ErrorContains(t, err, "content exceeds limit of 5 bytes")
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name: "content exceeds limit significantly",
			reader: func() io.ReadCloser {
				b := make([]byte, maxBytes+1000)
				for i := range b {
					b[i] = 'a'
				}
				return io.NopCloser(bytes.NewBuffer(b))
			}(),
			limit: maxBytes,
			assertions: func(t *testing.T, _ []byte, err error) {
				require.ErrorContains(t, err, "content exceeds limit")
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name:   "zero limit with non-empty content",
			reader: io.NopCloser(strings.NewReader("a")),
			limit:  0,
			assertions: func(t *testing.T, _ []byte, err error) {
				require.ErrorContains(t, err, "content exceeds limit of 0 bytes")
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name:   "zero limit with empty content",
			reader: io.NopCloser(strings.NewReader("")),
			limit:  0,
			assertions: func(t *testing.T, data []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, "", string(data))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := LimitRead(test.reader, test.limit)
			test.assertions(t, data, err)
		})
	}
}

func TestLimitCopy(t *testing.T) {
	const maxBytes = 2 << 20 // 2MB

	tests := []struct {
		name       string
		reader     io.ReadCloser
		limit      int64
		assertions func(t *testing.T, buf *bytes.Buffer, bytesWritten int64, err error)
	}{
		{
			name:   "small content within limit",
			reader: io.NopCloser(strings.NewReader("hello there")),
			limit:  maxBytes,
			assertions: func(t *testing.T, buf *bytes.Buffer, bytesWritten int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(11), bytesWritten)
				require.Equal(t, "hello there", buf.String())
			},
		},
		{
			name:   "empty content",
			reader: io.NopCloser(strings.NewReader("")),
			limit:  maxBytes,
			assertions: func(t *testing.T, buf *bytes.Buffer, bytesWritten int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), bytesWritten)
				require.Equal(t, "", buf.String())
			},
		},
		{
			name:   "content exactly at limit",
			reader: io.NopCloser(strings.NewReader("hello")),
			limit:  5,
			assertions: func(t *testing.T, buf *bytes.Buffer, bytesWritten int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(5), bytesWritten)
				require.Equal(t, "hello", buf.String())
			},
		},
		{
			name:   "content exceeds limit by one byte",
			reader: io.NopCloser(strings.NewReader("hello!")),
			limit:  5,
			assertions: func(t *testing.T, _ *bytes.Buffer, bytesWritten int64, err error) {
				require.ErrorContains(t, err, "content exceeds limit of 5 bytes")
				require.Equal(t, int64(5), bytesWritten)
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name: "content exceeds limit significantly",
			reader: func() io.ReadCloser {
				b := make([]byte, maxBytes+1000)
				for i := range b {
					b[i] = 'a'
				}
				return io.NopCloser(bytes.NewBuffer(b))
			}(),
			limit: maxBytes,
			assertions: func(t *testing.T, _ *bytes.Buffer, bytesWritten int64, err error) {
				require.Error(t, err)
				require.Equal(t, int64(maxBytes), bytesWritten)
				require.ErrorContains(t, err, "content exceeds limit")
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name:   "zero limit with non-empty content",
			reader: io.NopCloser(strings.NewReader("a")),
			limit:  0,
			assertions: func(t *testing.T, _ *bytes.Buffer, bytesWritten int64, err error) {
				require.ErrorContains(t, err, "content exceeds limit of 0 bytes")
				require.Equal(t, int64(0), bytesWritten)
				var bodyTooLargeErr *BodyTooLargeError
				require.ErrorAs(t, err, &bodyTooLargeErr)
			},
		},
		{
			name:   "zero limit with empty content",
			reader: io.NopCloser(strings.NewReader("")),
			limit:  0,
			assertions: func(t *testing.T, buf *bytes.Buffer, bytesWritten int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), bytesWritten)
				require.Equal(t, "", buf.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bytesWritten, err := LimitCopy(&buf, tt.reader, tt.limit)
			tt.assertions(t, &buf, bytesWritten, err)
		})
	}
}

func TestLimitReadErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		reader     io.ReadCloser
		limit      int64
		assertions func(t *testing.T, reader io.ReadCloser, data []byte, err error)
	}{
		{
			name:   "read error propagation",
			reader: &failingReader{err: io.ErrUnexpectedEOF},
			limit:  1024,
			assertions: func(t *testing.T, reader io.ReadCloser, _ []byte, err error) {
				require.ErrorContains(t, err, "failed to read from reader")
				failingReader := reader.(*failingReader) // nolint:forcetypeassert
				require.ErrorIs(t, err, failingReader.err)
			},
		},
		{
			name:   "reader closes on success",
			reader: &trackingCloser{Reader: strings.NewReader("hello")},
			limit:  1024,
			assertions: func(t *testing.T, reader io.ReadCloser, _ []byte, err error) {
				require.NoError(t, err)
				tracker := reader.(*trackingCloser) // nolint:forcetypeassert
				require.True(t, tracker.closed, "reader should be closed")
			},
		},
		{
			name:   "reader closes on error",
			reader: &trackingCloser{Reader: strings.NewReader("hello!")},
			limit:  5,
			assertions: func(t *testing.T, reader io.ReadCloser, _ []byte, err error) {
				require.Error(t, err)
				tracker := reader.(*trackingCloser) // nolint:forcetypeassert
				require.True(t, tracker.closed, "reader should be closed")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := LimitRead(test.reader, test.limit)
			test.assertions(t, test.reader, data, err)
		})
	}
}

func TestLimitCopyErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		reader     io.ReadCloser
		limit      int64
		assertions func(t *testing.T, reader io.ReadCloser, err error)
	}{
		{
			name:   "copy error propagation",
			reader: &failingReader{err: io.ErrUnexpectedEOF},
			limit:  1024,
			assertions: func(t *testing.T, reader io.ReadCloser, err error) {
				require.ErrorContains(t, err, "failed to copy from reader")
				failingReader := reader.(*failingReader) // nolint:forcetypeassert
				require.ErrorIs(t, err, failingReader.err)
			},
		},
		{
			name:   "reader closes on success",
			reader: &trackingCloser{Reader: strings.NewReader("hello")},
			limit:  1024,
			assertions: func(t *testing.T, reader io.ReadCloser, err error) {
				require.NoError(t, err)
				tracker := reader.(*trackingCloser) // nolint:forcetypeassert
				require.True(t, tracker.closed, "reader should be closed")
			},
		},
		{
			name:   "reader closes on error",
			reader: &trackingCloser{Reader: strings.NewReader("hello!")},
			limit:  5,
			assertions: func(t *testing.T, reader io.ReadCloser, err error) {
				require.Error(t, err)
				tracker := reader.(*trackingCloser) // nolint:forcetypeassert
				require.True(t, tracker.closed, "reader should be closed")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := LimitCopy(&buf, test.reader, test.limit)
			test.assertions(t, test.reader, err)
		})
	}
}

func TestBodyTooLargeError(t *testing.T) {
	tests := []struct {
		name       string
		limit      int64
		assertions func(t *testing.T, err error)
	}{
		{
			name:  "1KB limit",
			limit: 1024,
			assertions: func(t *testing.T, err error) {
				require.Equal(t, "content exceeds limit of 1024 bytes", err.Error())
			},
		},
		{
			name:  "zero limit",
			limit: 0,
			assertions: func(t *testing.T, err error) {
				require.Equal(t, "content exceeds limit of 0 bytes", err.Error())
			},
		},
		{
			name:  "large limit",
			limit: 1048576,
			assertions: func(t *testing.T, err error) {
				require.Equal(t, "content exceeds limit of 1048576 bytes", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BodyTooLargeError{limit: tt.limit}
			tt.assertions(t, err)
		})
	}
}

type failingReader struct {
	err error
}

func (f *failingReader) Read(_ []byte) (int, error) {
	return 0, f.err
}

func (f *failingReader) Close() error {
	return nil
}

type trackingCloser struct {
	io.Reader
	closed bool
}

func (t *trackingCloser) Close() error {
	t.closed = true
	return nil
}
