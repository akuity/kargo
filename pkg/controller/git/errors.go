package git

import (
	"errors"
)

// ErrRebaseUnsafe is returned when the PushIntegrationPolicyRebaseOrFail
// policy is in effect and rebasing is not safe.
var ErrRebaseUnsafe = errors.New(
	"rebase is unsafe and policy prohibits merge fallback",
)

// IsRebaseUnsafe returns true if the error is a rebase-unsafe error or wraps
// one and false otherwise.
func IsRebaseUnsafe(err error) bool {
	return errors.Is(err, ErrRebaseUnsafe)
}

// ErrShallowHistoryIncomplete is returned when a shallow clone cannot be
// deepened enough to integrate remote changes safely.
var ErrShallowHistoryIncomplete = errors.New(
	"shallow clone history is insufficient for integrating remote changes; " +
		"clone with a greater depth or with no depth limit",
)

// IsShallowHistoryIncomplete returns true if the error is a
// shallow-history-incomplete error or wraps one and false otherwise.
func IsShallowHistoryIncomplete(err error) bool {
	return errors.Is(err, ErrShallowHistoryIncomplete)
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
