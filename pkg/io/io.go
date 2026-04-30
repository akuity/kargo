package io

import (
	"fmt"
	"io"
)

// BodyTooLargeError is an error that indicates that the size of a request or
// response body exceeded a specified limit.
type BodyTooLargeError struct {
	limit int64
}

func (e *BodyTooLargeError) Error() string {
	return fmt.Sprintf("content exceeds limit of %d bytes", e.limit)
}

// LimitRead reads from the provided io.ReadCloser up to the specified limit.
// If the body exceeds the limit, it returns an error. It returns the read
// bytes or an error if any issues occur during reading.
func LimitRead(r io.ReadCloser, limit int64) ([]byte, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	// Validate that content doesn't exceed the limit
	if err := validateContentSize(r, int64(len(bodyBytes)), limit); err != nil {
		return nil, err
	}

	return bodyBytes, nil
}

// LimitCopy copies from the provided io.ReadCloser to dst up to the specified
// limit. If the source exceeds the limit, it returns an error. It returns the
// number of bytes copied or an error if any issues occur during copying.
func LimitCopy(dst io.Writer, r io.ReadCloser, limit int64) (int64, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit)

	// Copy as far as we are allowed to
	bytesWritten, err := io.Copy(dst, lr)
	if err != nil {
		return bytesWritten, fmt.Errorf("failed to copy from reader: %w", err)
	}

	// Validate that content doesn't exceed the limit
	if err := validateContentSize(r, bytesWritten, limit); err != nil {
		return bytesWritten, err
	}

	return bytesWritten, nil
}

// validateContentSize checks if the content exceeds the specified limit.
// It should be called after reading exactly 'bytesRead' bytes from the reader.
// If bytesRead equals the limit, it attempts to read one more byte to verify
// no additional content exists beyond the limit.
func validateContentSize(r io.ReadCloser, bytesRead int64, limit int64) error {
	if bytesRead == limit {
		// Try to read one more byte to check for additional content
		buf := make([]byte, 1)
		var n int
		var err error
		if n, err = r.Read(buf); err != nil && err != io.EOF {
			return fmt.Errorf("failed to check for additional content: %w", err)
		}
		if n > 0 {
			return &BodyTooLargeError{limit: limit}
		}
	}
	return nil
}
