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

var (
	flagSetOutput *pflag.FlagSet
	flagSetRender *pflag.FlagSet
	flagSetServer *pflag.FlagSet
)

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

	flagSetRender = pflag.NewFlagSet(
		"render",
		pflag.ErrorHandling(flag.ExitOnError),
	)
	flagSetRender.StringP(
		flagRepo,
		"r",
		"",
		"the URL of a remote gitops repo",
	)
	flagSetRender.StringP(
		flagRepoUsername,
		"u",
		"",
		"username for reading from and writing to the remote gitops repo (can "+
			"also be set using the BOOKKEEPER_REPO_USERNAME environment variable)",
	)
	flagSetRender.StringP(
		flagRepoPassword,
		"p",
		"",
		"password or token for reading from and writing to the remote gitops "+
			"repo (can also be set using the BOOKKEEPER_REPO_PASSWORD environment "+
			"variable)",
	)
	flagSetRender.StringP(
		flagTargetBranch,
		"t",
		"",
		"the environment-specific branch to write fully-rendered configuration to",
	)

	flagSetServer = pflag.NewFlagSet(
		"server",
		pflag.ErrorHandling(flag.ExitOnError),
	)
	flagSetServer.BoolP(
		flagInsecure,
		"k",
		false,
		"tolerate certificate errors for HTTPS connections",
	)
	flagSetServer.StringP(
		flagServer,
		"s",
		"",
		"specify the address of the Bookkeeper server (can also be set using "+
			"the BOOKKEEPER_SERVER environment variable)",
	)
}
