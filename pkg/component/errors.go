package component

import "errors"

// NamedRegistrationNotFoundError is an error returned when a registration
// cannot be found in a name-based registry.
type NamedRegistrationNotFoundError struct {
	Name string
}

func (e NamedRegistrationNotFoundError) Error() string {
	return "registration with name " + e.Name + " not found"
}

// RegistrationNotFoundError is an error returned when no matching registration
// can be found in a registry.
type RegistrationNotFoundError struct{}

func (e RegistrationNotFoundError) Error() string {
	return "no matching registration found"
}

// IsNotFoundError returns true if the error is a NamedRegistrationNotFound or
// RegistrationNotFound error.
func IsNotFoundError(err error) bool {
	return errors.As(err, &RegistrationNotFoundError{}) ||
		errors.As(err, &NamedRegistrationNotFoundError{})
}
