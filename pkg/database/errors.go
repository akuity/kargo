package database

import "errors"

var (
	// ErrNotFound is returned when the row asked for by name does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when a row with the same name already
	// exists in the same Project.
	ErrAlreadyExists = errors.New("already exists")
	// ErrConflict is returned when a write carries a precondition, such as an
	// ID or a version, that the current row does not satisfy.
	ErrConflict = errors.New("conflict")
	// ErrInvalid is returned when the database rejects a value, for instance
	// a label whose value is not a string or a parameter containing a NUL.
	ErrInvalid = errors.New("invalid")
	// ErrNotMirrored is returned when a write depends on the mirrored row of
	// a Kubernetes object that the mirror has not recorded yet. Retrying
	// once the mirror catches up succeeds.
	ErrNotMirrored = errors.New("not mirrored yet")
)
