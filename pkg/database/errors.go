package database

import "errors"

var (
	// ErrNotFound is returned when the row asked for by name does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when a row with the same name already
	// exists in the same Project.
	ErrAlreadyExists = errors.New("already exists")
	// ErrConflict is returned when a write carries a precondition, such as a
	// resource version or a UID, that the current row does not satisfy.
	ErrConflict = errors.New("conflict")
	// ErrProjectNotMirrored is returned when a write names a Project that has
	// no row yet. Projects reach the database from Kubernetes with some lag,
	// so a caller that has just confirmed the Project exists should retry.
	ErrProjectNotMirrored = errors.New("project is not in the database yet")
	// ErrInvalid is returned when the database rejects a value, for instance
	// a label whose value is not a string or a parameter containing a NUL.
	ErrInvalid = errors.New("invalid")
)
