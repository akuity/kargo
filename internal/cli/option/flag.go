package option

import (
	"github.com/spf13/pflag"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	// AliasFlag is the flag name for the alias flag.
	AliasFlag = "alias"
	// AliasShortFlag is the short flag name for the alias flag.
	AliasShortFlag = "a"

	// AsKubernetesResourcesFlag is the flag name for the as-kubernetes-resources
	// flag.
	AsKubernetesResourcesFlag = "as-kubernetes-resources"
	// AsKubernetesResourcesShortFlag is the short flag name for the
	// as-kubernetes-resources flag.
	AsKubernetesResourcesShortFlag = "k"

	// Claim is a flag name for the claim flag
	ClaimFlag = "claim"

	// EmailFlag is the flag name for the email flag.
	EmailFlag = "email"

	// FilenameFlag is the flag name for the filename flag.
	FilenameFlag = "filename"
	// FilenameShortFlag is the short flag name for the filename flag.
	FilenameShortFlag = "f"

	// FreightFlag is the flag name for the freight flag.
	FreightFlag = "freight"

	// FreightAliasFlag is the flag name for the freight-alias flag.
	FreightAliasFlag = "freight-alias"

	// GitFlag is the flag name for the git flag.
	GitFlag = string(credentials.TypeGit)

	// GroupFlag is the flag name for the group flag.
	GroupFlag = "group"

	// HelmFlag is the flag name for the helm flag.
	HelmFlag = string(credentials.TypeHelm)

	// ImageFlag is the flag name for the image flag.
	ImageFlag = string(credentials.TypeImage)

	// InsecureTLSFlag is the flag name for the insecure-tls flag.
	InsecureTLSFlag = "insecure-skip-tls-verify"

	// InteractivePasswordFlag is the flag name for the interactive-password flag.
	InteractivePasswordFlag = "interactive-password"

	// NameFlag is the flag name for the name flag.
	NameFlag = "name"

	// DescriptionFlag is the flag name for the description flag.
	DescriptionFlag = "description"

	// NewAliasFlag is the flag name for the new-alias flag.
	NewAliasFlag = "new-alias"

	// NoHeadersFlag is the flag name for the no-headers flag.
	NoHeadersFlag = "no-headers"

	// OldAliasFlag is the flag name for the old-alias flag.
	OldAliasFlag = "old-alias"

	// OriginFlag is the flag name for the origin flag.
	OriginFlag = "origin"

	// PasswordFlag is the flag name for the password flag.
	PasswordFlag = "password"

	// ProjectFlag is the flag name for the project flag.
	ProjectFlag = "project"
	// ProjectShortFlag is the short flag name for the project flag.
	ProjectShortFlag = "p"

	// RecursiveFlag is the flag name for the recursive flag.
	RecursiveFlag = "recursive"
	// RecursiveShortFlag is the short flag name for the recursive flag.
	RecursiveShortFlag = "R"

	// RegexFlag is the flag name for the regex flag.
	RegexFlag = "regex"

	// RepoURLFlag is the flag name for the repo-url flag.
	RepoURLFlag = "repo-url"

	// ResourceNameFlag is the flag name for the resource-name flag.
	ResourceNameFlag = "resource-name"

	// ResourceTypeFlag is the flag name for the resource-type flag.
	ResourceTypeFlag = "resource-type"

	// RoleFlag is the flag name for the role flag.
	RoleFlag = "role"

	// StageFlag is the flag name for the stage flag.
	StageFlag = "stage"

	// SubFlag is the flag name for the sub flag.
	SubFlag = "sub"

	// DownstreamFromFlag is the flag name for the downstream-from flag.
	DownstreamFromFlag = "downstream-from"

	// TypeFlag is the flag name for the type flag.
	TypeFlag = "type"

	// UsernameFlag is the flag name for the username flag.
	UsernameFlag = "username"

	// VerbFlag is the flag name for the verb flag.
	VerbFlag = "verb"

	// WaitFlag is the flag name for the wait flag.
	WaitFlag = "wait"
)

// Claims adds a multi-value ClaimFlag to the provided flag set.
func Claims(fs *pflag.FlagSet, claims *[]string, usage string) {
	fs.StringSliceVar(claims, ClaimFlag, nil, usage)
}

// Alias adds the AliasFlag to the provided flag set.
func Alias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, AliasFlag, "", usage)
}

// Aliases adds a multi-value AliasFlag to the provided flag set.
func Aliases(fs *pflag.FlagSet, stage *[]string, usage string) {
	fs.StringArrayVar(stage, AliasFlag, nil, usage)
}

// AsKubernetesResources adds the AsKubernetesResourcesFlag and
// AsKubernetesResourcesShortFlag to the provided flag set.
func AsKubernetesResources(fs *pflag.FlagSet, asKubernetesResources *bool, usage string) {
	fs.BoolVarP(
		asKubernetesResources,
		AsKubernetesResourcesFlag,
		AsKubernetesResourcesShortFlag,
		false,
		usage,
	)
}

// Description adds the DescriptionFlag to the provided flag set.
func Description(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, DescriptionFlag, "", usage)
}

// Emails adds a multi-value EmailFlag to the provided flag set.
func Emails(fs *pflag.FlagSet, emails *[]string, usage string) {
	fs.StringSliceVar(emails, EmailFlag, nil, usage)
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
	fs.BoolVar(git, GitFlag, false, usage)
}

// Groups adds a multi-value GroupFlag to the provided flag set.
func Groups(fs *pflag.FlagSet, groups *[]string, usage string) {
	fs.StringSliceVar(groups, GroupFlag, nil, usage)
}

// Helm adds the HelmFlag to the provided flag set.
func Helm(fs *pflag.FlagSet, helm *bool, usage string) {
	fs.BoolVar(helm, HelmFlag, false, usage)
}

// Image adds the ImageFlag to the provided flag set.
func Image(fs *pflag.FlagSet, image *bool, usage string) {
	fs.BoolVar(image, ImageFlag, false, usage)
}

// InsecureTLS adds the InsecureTLSFlag to the provided flag set.
func InsecureTLS(fs *pflag.FlagSet, insecure *bool) {
	fs.BoolVar(insecure, InsecureTLSFlag, false, "Skip TLS certificate verification")
}

// InteractivePassword adds the InteractivePasswordFlag to the provided flag set.
func InteractivePassword(fs *pflag.FlagSet, changePasswordInteractively *bool, usage string) {
	fs.BoolVar(changePasswordInteractively, InteractivePasswordFlag, false, usage)
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

// NoHeaders adds the NoHeadersFlag to the provided flag set.
func NoHeaders(fs *pflag.FlagSet, noHeaders *bool) {
	fs.BoolVar(
		noHeaders,
		NoHeadersFlag,
		false,
		"When using the default output format, don't print headers (default print headers).",
	)
}

// OldAlias adds the OldAliasFlag to the provided flag set.
func OldAlias(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, OldAliasFlag, "", usage)
}

// Origins adds the OriginsFlag to the provided flag set.
func Origins(fs *pflag.FlagSet, origin *[]string, usage string) {
	fs.StringArrayVar(origin, OriginFlag, nil, usage)
}

// Password adds the PasswordFlag to the provided flag set.
func Password(fs *pflag.FlagSet, password *string, usage string) {
	fs.StringVar(password, PasswordFlag, "", usage)
}

// Project adds the ProjectFlag and ProjectShortFlag to the provided flag set.
func Project(fs *pflag.FlagSet, project *string, defaultProject, usage string) {
	fs.StringVarP(project, ProjectFlag, ProjectShortFlag, defaultProject, usage)
}

// Recursive adds the RecursiveFlag and RecursiveShortFlag to the provided flag
// set.
func Recursive(fs *pflag.FlagSet, recursive *bool) {
	fs.BoolVarP(recursive, RecursiveFlag, RecursiveShortFlag, false,
		"Process the directory used in -f, --filename recursively. Useful when "+
			"you want to manage related manifests organized within the same directory.",
	)
}

// RepoURL adds the RepoURLFlag to the provided flag set.
func RepoURL(fs *pflag.FlagSet, repoURL *string, usage string) {
	fs.StringVar(repoURL, RepoURLFlag, "", usage)
}

// Regex adds the RegexFlag to the provided flag set.
func Regex(fs *pflag.FlagSet, regex *bool, usage string) {
	fs.BoolVar(regex, RegexFlag, false, usage)
}

// ResourceName adds the ResourceNameFlag to the provided flag set.
func ResourceName(fs *pflag.FlagSet, resourceName *string, usage string) {
	fs.StringVar(resourceName, ResourceNameFlag, "", usage)
}

// ResourceType adds the ResourceTypeFlag to the provided flag set.
func ResourceType(fs *pflag.FlagSet, repoType *string, usage string) {
	fs.StringVar(repoType, ResourceTypeFlag, "", usage)
}

// Role adds the RoleFlag to the provided flag set.
func Role(fs *pflag.FlagSet, role *string, usage string) {
	fs.StringVar(role, RoleFlag, "", usage)
}

// Stage adds the StageFlag to the provided flag set.
func Stage(fs *pflag.FlagSet, stage *string, usage string) {
	fs.StringVar(stage, StageFlag, "", usage)
}

// Subs adds a multi-value SubFlag to the provided flag set.
func Subs(fs *pflag.FlagSet, subs *[]string, usage string) {
	fs.StringSliceVar(subs, SubFlag, nil, usage)
}

// DownstreamFrom adds the DownstreamFromFlag to the provided flag set.
func DownstreamFrom(fs *pflag.FlagSet, downstreamFrom *string, usage string) {
	fs.StringVar(downstreamFrom, DownstreamFromFlag, "", usage)
}

// Type adds the TypeFlag to the provided flag set.
func Type(fs *pflag.FlagSet, repoType *string, usage string) {
	fs.StringVar(repoType, TypeFlag, "", usage)
}

// Username adds the UsernameFlag to the provided flag set.
func Username(fs *pflag.FlagSet, username *string, usage string) {
	fs.StringVar(username, UsernameFlag, "", usage)
}

// Verbs adds a multi-value VerbFlag to the provided flag set.
func Verbs(fs *pflag.FlagSet, verbs *[]string, usage string) {
	fs.StringSliceVar(verbs, VerbFlag, nil, usage)
}

// Wait adds the WaitFlag to the provided flag set.
func Wait(fs *pflag.FlagSet, wait *bool, defaultWait bool, usage string) {
	fs.BoolVar(wait, WaitFlag, defaultWait, usage)
}
