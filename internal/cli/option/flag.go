package option

import (
	"github.com/spf13/pflag"
)

const (
	// FilenameFlag is the flag name for the filename flag.
	FilenameFlag = "filename"
	// FilenameShortFlag is the short flag name for the filename flag.
	FilenameShortFlag = "f"

	// ProjectFlag is the flag name for the project flag.
	ProjectFlag = "project"
	// ProjectShortFlag is the short flag name for the project flag.
	ProjectShortFlag = "p"

	// FreightFlag is the flag name for the freight flag.
	FreightFlag = "freight"

	// StageFlag is the flag name for the stage flag.
	StageFlag = "stage"

	// SubscribersOfFlag is the flag name for the subscribers-of flag.
	SubscribersOfFlag = "subscribers-of"

	// WaitFlag is the flag name for the wait flag.
	WaitFlag = "wait"
)

// Filenames adds the FilenameFlag and FilenameShortFlag to the provided flag set.
func Filenames(fs *pflag.FlagSet, filenames *[]string, usage string) {
	fs.StringSliceVarP(filenames, FilenameFlag, FilenameShortFlag, nil, usage)
}

func InsecureTLS(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.InsecureTLS, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
}

func LocalServer(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.UseLocalServer, "local-server", false, "Use local server")
}

func ClientVersion(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.ClientVersionOnly, "client", false, "If true, shows client version only (no server required)")
}

// Project adds the ProjectFlag and ProjectShortFlag to the provided flag set.
func Project(fs *pflag.FlagSet, project *string, defaultProject, usage string) {
	fs.StringVarP(project, ProjectFlag, ProjectShortFlag, defaultProject, usage)
}

// Stage adds the StageFlag to the provided flag set.
func Stage(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, StageFlag, "", usage)
}

// SubscribersOf adds the SubscribersOfFlag to the provided flag set.
func SubscribersOf(fs *pflag.FlagSet, subscribersOf *string, usage string) {
	fs.StringVar(subscribersOf, SubscribersOfFlag, "", usage)
}

// Freight adds the FreightFlag to the provided flag set.
func Freight(fs *pflag.FlagSet, freight *string, usage string) {
	fs.StringVar(freight, FreightFlag, "", usage)
}

// Wait adds the WaitFlag to the provided flag set.
func Wait(fs *pflag.FlagSet, wait *bool, defaultWait bool, usage string) {
	fs.BoolVar(wait, WaitFlag, defaultWait, usage)
}
