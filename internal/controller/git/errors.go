package git

import (
	"errors"
)

// ErrMergeConflict is returned when a merge conflict occurs.
var ErrMergeConflict = errors.New("merge conflict")

// IsMergeConflict returns true if the error is a merge conflict or wraps one
// and false otherwise.
func IsMergeConflict(err error) bool {
	return errors.Is(err, ErrMergeConflict)
}

// ErrNonFastForward is returned when a push is rejected because it is not a
// fast-forward or needs to be fetched first.
var ErrNonFastForward = errors.New("non-fast-forward")

// IsNonFastForward returns true if the error is a non-fast-forward or wraps one
// and false otherwise.
func IsNonFastForward(err error) bool {
	return errors.Is(err, ErrNonFastForward)
}
