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
