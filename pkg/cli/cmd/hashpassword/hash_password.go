package hashpassword

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	libpassword "github.com/akuity/kargo/pkg/password"
)

const passwordStdinFlag = "password-stdin"

type hashPasswordOptions struct {
	genericiooptions.IOStreams

	Password      string
	PasswordStdin bool
}

func NewCommand(streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &hashPasswordOptions{IOStreams: streams}

	cmd := &cobra.Command{
		Use:   "hash-password [--password PASSWORD | --password-stdin]",
		Short: "Hash a password for the Kargo admin account",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Hash a password entered at a prompt
kargo hash-password

# Hash a password specified by flag
kargo hash-password --password 'my-password'

# Hash a password read from stdin
echo 'my-password' | kargo hash-password --password-stdin
`),
		RunE: func(*cobra.Command, []string) error {
			return cmdOpts.run()
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the hash-password options to the provided
// command.
func (o *hashPasswordOptions) addFlags(cmd *cobra.Command) {
	option.Password(cmd.Flags(), &o.Password,
		"The password to hash. If neither this nor --password-stdin is set, the password is prompted for.")
	cmd.Flags().BoolVar(&o.PasswordStdin, passwordStdinFlag, false,
		"Read the password to hash from stdin.")

	cmd.MarkFlagsMutuallyExclusive(option.PasswordFlag, passwordStdinFlag)
}

// run hashes the password and prints the hash.
func (o *hashPasswordOptions) run() error {
	password, err := o.getPassword()
	if err != nil {
		return err
	}
	if password == "" {
		return errors.New("password must not be empty")
	}
	hash, err := libpassword.Hash(password)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}
	_, _ = fmt.Fprintln(o.Out, hash)
	return nil
}

// getPassword returns the password from the --password flag, from the first
// line of stdin, or from a prompt, in that order of precedence.
func (o *hashPasswordOptions) getPassword() (string, error) {
	switch {
	case o.Password != "":
		return o.Password, nil
	case o.PasswordStdin:
		scanner := bufio.NewScanner(o.In)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading password from stdin: %w", err)
		}
		return scanner.Text(), nil
	default:
		return promptForPassword()
	}
}

// promptForPassword prompts for the password twice and returns it if both
// entries match. Prompts are written to stderr so that stdout carries only the
// hash, even when it is captured.
func promptForPassword() (string, error) {
	stdio := survey.WithStdio(os.Stdin, os.Stderr, os.Stderr)
	var password, confirmation string
	if err := survey.AskOne(
		&survey.Password{Message: "Password"},
		&password,
		stdio,
	); err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}
	if err := survey.AskOne(
		&survey.Password{Message: "Confirm password"},
		&confirmation,
		stdio,
	); err != nil {
		return "", fmt.Errorf("error reading password confirmation: %w", err)
	}
	if password != confirmation {
		return "", errors.New("passwords do not match")
	}
	return password, nil
}
