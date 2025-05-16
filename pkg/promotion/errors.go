package promotion

import "errors"

// TerminalError wraps another error to indicate to the step execution engine
// that the step that produced the error should not be retried.
type TerminalError struct {
	Err error
}

// Error implements the error interface.
func (e *TerminalError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// IsTerminal returns true if the error is a terminal error or wraps one and
// false otherwise.
func IsTerminal(err error) bool {
	te := &TerminalError{}
	return errors.As(err, &te)
}
