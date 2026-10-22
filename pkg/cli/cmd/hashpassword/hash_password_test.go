package hashpassword

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	libpassword "github.com/akuity/kargo/pkg/password"
)

func TestNewCommand(t *testing.T) {
	const testPassword = "correct horse battery staple"
	testCases := []struct {
		name   string
		args   []string
		stdin  string
		assert func(t *testing.T, out *bytes.Buffer, err error)
	}{
		{
			name: "--password and --password-stdin both set",
			args: []string{"--password", testPassword, "--password-stdin"},
			assert: func(t *testing.T, _ *bytes.Buffer, err error) {
				require.ErrorContains(t, err, "none of the others can be")
			},
		},
		{
			name: "--password",
			args: []string{"--password", testPassword},
			assert: func(t *testing.T, out *bytes.Buffer, err error) {
				require.NoError(t, err)
				requireHashOf(t, testPassword, out)
			},
		},
		{
			name:  "--password-stdin",
			args:  []string{"--password-stdin"},
			stdin: testPassword + "\n",
			assert: func(t *testing.T, out *bytes.Buffer, err error) {
				require.NoError(t, err)
				requireHashOf(t, testPassword, out)
			},
		},
		{
			name: "--password-stdin with empty stdin",
			args: []string{"--password-stdin"},
			assert: func(t *testing.T, _ *bytes.Buffer, err error) {
				require.ErrorContains(t, err, "must not be empty")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			streams, in, out, _ := genericiooptions.NewTestIOStreams()
			in.WriteString(testCase.stdin)
			cmd := NewCommand(streams)
			cmd.SetArgs(testCase.args)
			err := cmd.Execute()
			testCase.assert(t, out, err)
		})
	}
}

// requireHashOf requires out to contain a single line holding a hash of
// password.
func requireHashOf(t *testing.T, password string, out *bytes.Buffer) {
	t.Helper()
	hash, ok := strings.CutSuffix(out.String(), "\n")
	require.True(t, ok)
	require.NotContains(t, hash, "\n")
	ok, err := libpassword.Verify(hash, password)
	require.NoError(t, err)
	require.True(t, ok)
}
