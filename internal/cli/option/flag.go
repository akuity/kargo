package option

import (
	"github.com/spf13/pflag"
)

const (
	// AliasFlag is the flag name for the alias flag.
	AliasFlag = "alias"
	// AliasShortFlag is the short flag name for the alias flag.
	AliasShortFlag = "a"

	// FilenameFlag is the flag name for the filename flag.
	FilenameFlag = "filename"
	// FilenameShortFlag is the short flag name for the filename flag.
	FilenameShortFlag = "f"

	// FreightFlag is the flag name for the freight flag.
	FreightFlag = "freight"

	// FreightAliasFlag is the flag name for the freight-alias flag.
	FreightAliasFlag = "freight-alias"

	// NameFlag is the flag name for the name flag.
	NameFlag = "name"

	// ProjectFlag is the flag name for the project flag.
	ProjectFlag = "project"
	// ProjectShortFlag is the short flag name for the project flag.
	ProjectShortFlag = "p"

	// NewAliasFlag is the flag name for the new-alias flag.
	NewAliasFlag = "new-alias"

	// OldAliasFlag is the flag name for the old-alias flag.
	OldAliasFlag = "old-alias"

	// StageFlag is the flag name for the stage flag.
	StageFlag = "stage"

	// SubscribersOfFlag is the flag name for the subscribers-of flag.
	SubscribersOfFlag = "subscribers-of"

	// WaitFlag is the flag name for the wait flag.
	WaitFlag = "wait"
)

// Alias adds the AliasFlag to the provided flag set.
func Alias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, AliasFlag, "", usage)
}

// Filenames adds the FilenameFlag and FilenameShortFlag to the provided flag set.
func Filenames(fs *pflag.FlagSet, filenames *[]string, usage string) {
	fs.StringSliceVarP(filenames, FilenameFlag, FilenameShortFlag, nil, usage)
}

// Freight adds the FreightFlag to the provided flag set.
func Freight(fs *pflag.FlagSet, freight *string, usage string) {
	fs.StringVar(freight, FreightFlag, "", usage)
}

// FreightAlias adds the FreightAliasFlag to the provided flag set.
func FreightAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, FreightAliasFlag, "", usage)
}

func InsecureTLS(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.InsecureTLS, "insecure-skip-tls-verify", false, "Skip TLS certificate verification")
}

func LocalServer(fs *pflag.FlagSet, opt *Option) {
	fs.BoolVar(&opt.UseLocalServer, "local-server", false, "Use local server")
}

// Name adds the NameFlag to the provided flag set.
func Name(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, NameFlag, "", usage)
}

// NewAlias adds the NewAliasFlag to the provided flag set.
func NewAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, NewAliasFlag, "", usage)
}

// OldAlias adds the OldAliasFlag to the provided flag set.
func OldAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, OldAliasFlag, "", usage)
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

// Wait adds the WaitFlag to the provided flag set.
func Wait(fs *pflag.FlagSet, wait *bool, defaultWait bool, usage string) {
	fs.BoolVar(wait, WaitFlag, defaultWait, usage)
}
