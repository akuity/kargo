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

	// GitFlag is the flag name for the git flag.
	GitFlag = "git"

	// HelmFlag is the flag name for the helm flag.
	HelmFlag = "helm"

	// ImageFlag is the flag name for the image flag.
	ImageFlag = "image"

	// NameFlag is the flag name for the name flag.
	NameFlag = "name"

	// NewAliasFlag is the flag name for the new-alias flag.
	NewAliasFlag = "new-alias"

	// OldAliasFlag is the flag name for the old-alias flag.
	OldAliasFlag = "old-alias"

	// PasswordFlag is the flag name for the password flag.
	PasswordFlag = "password"

	// ProjectFlag is the flag name for the project flag.
	ProjectFlag = "project"
	// ProjectShortFlag is the short flag name for the project flag.
	ProjectShortFlag = "p"

	// RepoURLFlag is the flag name for the repo-url flag.
	RepoURLFlag = "repo-url"

	// RepoURLPatternFlag is the flag name for the repo-url-pattern flag.
	RepoURLPatternFlag = "repo-url-pattern"

	// StageFlag is the flag name for the stage flag.
	StageFlag = "stage"

	// SubscribersOfFlag is the flag name for the subscribers-of flag.
	SubscribersOfFlag = "subscribers-of"

	// UsernameFlag is the flag name for the username flag.
	UsernameFlag = "username"

	// WaitFlag is the flag name for the wait flag.
	WaitFlag = "wait"
)

// Alias adds the AliasFlag to the provided flag set.
func Alias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, AliasFlag, "", usage)
}

// Aliases adds a multi-value AliasFlag to the provided flag set.
func Aliases(fs *pflag.FlagSet, stage *[]string, usage string) {
	fs.StringArrayVar(stage, AliasFlag, nil, usage)
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

// Git adds the GitFlag to the provided flag set.
func Git(fs *pflag.FlagSet, git *bool, usage string) {
	fs.BoolVar(git, "git", false, usage)
}

// Helm adds the HelmFlag to the provided flag set.
func Helm(fs *pflag.FlagSet, helm *bool, usage string) {
	fs.BoolVar(helm, "helm", false, usage)
}

// Image adds the ImageFlag to the provided flag set.
func Image(fs *pflag.FlagSet, image *bool, usage string) {
	fs.BoolVar(image, "image", false, usage)
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

// Names adds a multi-value NameFlag to the provided flag set.
func Names(fs *pflag.FlagSet, stage *[]string, usage string) {
	fs.StringArrayVar(stage, NameFlag, nil, usage)
}

// NewAlias adds the NewAliasFlag to the provided flag set.
func NewAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, NewAliasFlag, "", usage)
}

// OldAlias adds the OldAliasFlag to the provided flag set.
func OldAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, OldAliasFlag, "", usage)
}

func Password(fs *pflag.FlagSet, password *string, usage string) {
	fs.StringVar(password, PasswordFlag, "", usage)
}

// Project adds the ProjectFlag and ProjectShortFlag to the provided flag set.
func Project(fs *pflag.FlagSet, project *string, defaultProject, usage string) {
	fs.StringVarP(project, ProjectFlag, ProjectShortFlag, defaultProject, usage)
}

// RepoURL adds the RepoURLFlag to the provided flag set.
func RepoURL(fs *pflag.FlagSet, repoURL *string, usage string) {
	fs.StringVar(repoURL, RepoURLFlag, "", usage)
}

// RepoURLPattern adds the RepoURLPatternFlag to the provided flag set.
func RepoURLPattern(fs *pflag.FlagSet, repoURLPattern *string, usage string) {
	fs.StringVar(repoURLPattern, RepoURLPatternFlag, "", usage)
}

// Stage adds the StageFlag to the provided flag set.
func Stage(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, StageFlag, "", usage)
}

// SubscribersOf adds the SubscribersOfFlag to the provided flag set.
func SubscribersOf(fs *pflag.FlagSet, subscribersOf *string, usage string) {
	fs.StringVar(subscribersOf, SubscribersOfFlag, "", usage)
}

// Username adds the UsernameFlag to the provided flag set.
func Username(fs *pflag.FlagSet, username *string, usage string) {
	fs.StringVar(username, UsernameFlag, "", usage)
}

// Wait adds the WaitFlag to the provided flag set.
func Wait(fs *pflag.FlagSet, wait *bool, defaultWait bool, usage string) {
	fs.BoolVar(wait, WaitFlag, defaultWait, usage)
}
