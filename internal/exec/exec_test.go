package exec

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExec(t *testing.T) {
	testCases := []struct {
		name       string
		cmd        *exec.Cmd
		assertions func(res []byte, err error)
	}{
		{
			name: "error",
			// This command should fail, but ALSO produce some output
			cmd: exec.Command("expr", "100", "/", "0"),
			assertions: func(res []byte, err error) {
				require.Error(t, err)
				exitErr, ok := err.(*ExitError)
				require.True(t, ok)
				// Path to expr will be different on Mac and Linux
				require.True(t, strings.HasSuffix(exitErr.Command, "expr 100 / 0"))
				require.Equal(t, "expr: division by zero\n", string(exitErr.Output))
				require.NotEmpty(t, exitErr.ExitCode)
				require.Contains(t, err.Error(), "expr 100 / 0")
				require.Contains(t, err.Error(), "expr: division by zero")
			},
		},
		{
			name: "success",
			cmd:  exec.Command("echo", "foobar"),
			assertions: func(res []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, "foobar\n", string(res))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(Exec(testCase.cmd))
		})
	}
}
