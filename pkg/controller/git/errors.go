package git

import (
	"errors"
)

// ErrRebaseUnsafe is returned when the PushIntegrationPolicyRebaseOrFail
// policy is in effect and the signature-trust decision matrix determines that
// rebasing is not safe.
var ErrRebaseUnsafe = errors.New(
	"rebase is unsafe and policy prohibits merge fallback",
)

// IsRebaseUnsafe returns true if the error is a rebase-unsafe error or wraps
// one and false otherwise.
func IsRebaseUnsafe(err error) bool {
	return errors.Is(err, ErrRebaseUnsafe)
}

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
