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

func (e *BodyTooLargeError) Is(target error) bool {
	_, ok := target.(*BodyTooLargeError)
	return ok
}

// LimitRead reads from the provided io.ReadCloser up to the specified limit.
// If the body exceeds the limit, it returns an error. If the body is exactly
// the limit, it checks for additional content and returns an error if any
// additional content is found. It returns the read bytes or an error if any
// issues occur during reading.
func LimitRead(r io.ReadCloser, limit int64) ([]byte, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	// If we read exactly the limit, the body might be larger
	if int64(len(bodyBytes)) == limit {
		// Try to read one more byte
		buf := make([]byte, 1)
		var n int
		if n, err = r.Read(buf); err != nil && err != io.EOF {
			return nil, fmt.Errorf(
				"failed to check for additional content: %w",
				err,
			)
		}
		if n > 0 {
			return nil, &BodyTooLargeError{limit: limit}
		}
	}
	return bodyBytes, nil
}
