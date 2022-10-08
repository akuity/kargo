package cli

import (
	"flag"

	"github.com/spf13/pflag"
)

const (
	flagImage        = "image"
	flagInsecure     = "insecure"
	flagOutput       = "output"
	flagOutputJSON   = "json"
	flagOutputYAML   = "yaml"
	flagRepo         = "repo"
	flagRepoPassword = "repo-password"
	flagRepoUsername = "repo-username"
	flagServer       = "server"
	flagTargetBranch = "target-branch"
)

var flagSetOutput *pflag.FlagSet

func init() {
	flagSetOutput = pflag.NewFlagSet(
		"output",
		pflag.ErrorHandling(flag.ExitOnError),
	)
	flagSetOutput.StringP(
		flagOutput,
		"o",
		"",
		"specify a format for command output (json or yaml)",
	)
}
