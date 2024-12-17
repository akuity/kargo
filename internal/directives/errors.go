package directives

import "errors"

// terminalError wraps another error to indicate to the step execution engine
// that the step that produced the error should not be retried.
type terminalError struct {
	err error
}

// Error implements the error interface.
func (e *terminalError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// isTerminal returns true if the error is a terminal error or wraps one and
// false otherwise.
func isTerminal(err error) bool {
	te := &terminalError{}
	return errors.As(err, &te)
}
