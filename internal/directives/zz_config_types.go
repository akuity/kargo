// Code generated by quicktype. DO NOT EDIT.

package directives

type CommonDefs interface{}

type CopyConfig struct {
	// InPath is the path to the file or directory to copy.
	InPath string `json:"inPath"`
	// OutPath is the path to the destination file or directory.
	OutPath string `json:"outPath"`
}

type GitCloneConfig struct {
	// The commits, branches, or tags to check out from the repository and the paths where they
	// should be checked out. At least one must be specified.
	Checkout []Checkout `json:"checkout"`
	// Indicates whether to skip TLS verification when cloning the repository. Default is false.
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
	// The URL of a remote Git repository to clone. Required.
	RepoURL string `json:"repoURL"`
}

type Checkout struct {
	// The branch to checkout. Mutually exclusive with 'tag' and 'fromFreight=true'. If none of
	// these is specified, the default branch is checked out.
	Branch string `json:"branch,omitempty"`
	// Indicates whether the ID of a commit to check out may be obtained from Freight. A value
	// of 'true' is mutually exclusive with 'branch' and 'tag'. If none of these is specified,
	// the default branch is checked out.
	FromFreight bool                `json:"fromFreight,omitempty"`
	FromOrigin  *CheckoutFromOrigin `json:"fromOrigin,omitempty"`
	// The path where the repository should be checked out.
	Path string `json:"path"`
	// The tag to checkout. Mutually exclusive with 'branch' and 'fromFreight=true'. If none of
	// these is specified, the default branch is checked out.
	Tag string `json:"tag,omitempty"`
}

type CheckoutFromOrigin struct {
	// The kind of origin. Currently only 'Warehouse' is supported. Required.
	Kind Kind `json:"kind"`
	// The name of the origin. Required.
	Name string `json:"name"`
}

type GitCommitConfig struct {
	// The author of the commit.
	Author *Author `json:"author,omitempty"`
	// The commit message.
	Message string `json:"message,omitempty"`
	// The path to a working directory of a local repository.
	Path string `json:"path"`
}

// The author of the commit.
type Author struct {
	// The email of the author.
	Email string `json:"email,omitempty"`
	// The name of the author.
	Name string `json:"name,omitempty"`
}

type GitPushConfig struct {
	// Indicates whether to push to a new remote branch. A value of 'true' is mutually exclusive
	// with 'targetBranch'. If neither of these is provided, the target branch will be the
	// currently checked out branch.
	GenerateTargetBranch bool `json:"generateTargetBranch,omitempty"`
	// The path to a working directory of a local repository.
	Path string `json:"path"`
	// The target branch to push to. Mutually exclusive with 'generateTargetBranch=true'. If
	// neither of these is provided, the target branch will be the currently checked out branch.
	TargetBranch string `json:"targetBranch,omitempty"`
}

type HelmUpdateChartConfig struct {
	// A list of chart dependencies which should receive updates.
	Charts []Chart `json:"charts"`
	// The path at which the umbrella chart with the dependency can be found.
	Path string `json:"path"`
}

type Chart struct {
	FromOrigin *ChartFromOrigin `json:"fromOrigin,omitempty"`
	// The name of the subchart, as defined in `Chart.yaml`.
	Name string `json:"name"`
	// The repository of the subchart, as defined in `Chart.yaml`. It also supports OCI charts
	// using `oci://`.
	Repository string `json:"repository"`
}

type ChartFromOrigin struct {
	// The kind of origin. Currently only 'Warehouse' is supported. Required.
	Kind Kind `json:"kind"`
	// The name of the origin. Required.
	Name string `json:"name"`
}

// The kind of origin. Currently only 'Warehouse' is supported. Required.
type Kind string

const (
	Warehouse Kind = "Warehouse"
)
