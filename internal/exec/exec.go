package exec

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

// ExitError is an error type that is produced by the Exec() function when a
// command returns a non-zero exit code.
type ExitError struct {
	// Command is the command that caused the error.
	Command string
	// Output is the combined output (stdout and stderr) produced when Command was
	// executed.
	Output []byte
	// ExitCode is the exit code that was returned when Command was executed.
	ExitCode int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf(
		"error executing cmd [%s]: %s",
		e.Command,
		string(e.Output),
	)
}

// Exec is a custom replacement for cmd.CombinedOutput(). It executes the
// provided command and returns the command's combined output (stdout + stderr)
// and an error. When the command completes successfully, with a non-zero exit
// code, the error is nil. If the command's exit code is non-zero, the error is
// of type ExitError. Other, unanticipated errors are wrapped and returned
// as-is. The primary benefit to calling Exec() over calling
// cmd.CombinedOutput() directly is that errors will automatically include
// command output, which is likely to contain important information about the
// cause of the error.
func Exec(cmd *exec.Cmd) ([]byte, error) {
	res, err := cmd.CombinedOutput()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return res, &ExitError{
			Command:  cmd.String(),
			Output:   res,
			ExitCode: exitErr.ExitCode(),
		}
	}
	return res,
		errors.Wrapf(err, "error executing cmd [%s]: %s", cmd.String(), string(res))
}
