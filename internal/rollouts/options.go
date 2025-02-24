package rollouts

import (
	"maps"

	"github.com/expr-lang/expr"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ulidLength is the length of a ulid.ULID string.
	ulidLength = 26
	// maxNameSuffixLength is the maximum length of the name suffix for an
	// AnalysisRun. It assumes that the suffix contains e.g. a SHA and can
	// be truncated to a smaller length than the maxNamePrefixLength.
	maxNameSuffixLength = 7
	// maxNamePrefixLength is the maximum length of the name prefix for an
	// AnalysisRun. It takes into account the maximum length of the name
	// field (253 characters), and the additional characters that will be
	// appended to the name (ULID, SHA, and period separators).
	maxNamePrefixLength = 253 - (1 + ulidLength) - (1 + maxNameSuffixLength)
)

// AnalysisRunOption is an option for configuring the build of an AnalysisRun.
type AnalysisRunOption interface {
	ApplyToAnalysisRun(*AnalysisRunOptions)
}

// AnalysisRunOptions holds the options for building an AnalysisRun.
type AnalysisRunOptions struct {
	NamePrefix       string
	NameSuffix       string
	ExtraLabels      map[string]string
	ExtraAnnotations map[string]string
	Owners           []Owner
	ExpressionConfig *ArgumentEvaluationConfig
}

// Owner represents a reference to an owner object.
type Owner struct {
	APIVersion    string
	Kind          string
	Reference     types.NamespacedName
	BlockDeletion bool
}

// ArgumentEvaluationConfig holds the configuration for the evaluation of
// expressions in the AnalysisRun arguments.
type ArgumentEvaluationConfig struct {
	// Env is a (nested) map of variables that can be used in the expressions.
	// The keys are the variable names, and the values are the variable values.
	// When the value itself is a map, it is considered a nested variable and
	// can be accessed using dot notation. e.g. `${{ foo.bar }}`.
	Env map[string]any
	// Options are the options for the expression evaluation. It can be used to
	// configure the behavior of the expression evaluation and the functions
	// available.
	Options []expr.Option
}

// Apply applies the given options to the AnalysisRunOptions.
func (o *AnalysisRunOptions) Apply(opts ...AnalysisRunOption) {
	for _, opt := range opts {
		opt.ApplyToAnalysisRun(o)
	}
}

// WithNamePrefix sets the name prefix for the AnalysisRun. If it is longer
// than maxNamePrefixLength, it will be truncated.
type WithNamePrefix string

func (o WithNamePrefix) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	prefix := o
	if len(prefix) > maxNamePrefixLength {
		prefix = prefix[0:maxNamePrefixLength]
	}
	opts.NamePrefix = string(prefix)
}

// WithNameSuffix sets the name suffix for the AnalysisRun. If it is longer
// than maxNameSuffixLength, it will be truncated.
type WithNameSuffix string

func (o WithNameSuffix) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	suffix := o
	if len(suffix) > maxNameSuffixLength {
		suffix = suffix[0:maxNameSuffixLength]
	}
	opts.NameSuffix = string(suffix)
}

// WithExtraLabels sets the extra labels for the AnalysisRun. It can be passed
// multiple times to add more labels.
type WithExtraLabels map[string]string

func (o WithExtraLabels) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	if opts.ExtraLabels != nil {
		maps.Copy(opts.ExtraLabels, o)
		return
	}
	opts.ExtraLabels = o
}

// WithExtraAnnotations sets the extra labels for the AnalysisRun. It can be passed
// multiple times to add more annotations.
type WithExtraAnnotations map[string]string

func (o WithExtraAnnotations) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	if opts.ExtraAnnotations != nil {
		maps.Copy(opts.ExtraAnnotations, o)
		return
	}
	opts.ExtraAnnotations = o
}

// WithOwner sets the owner for the AnalysisRun. It can be passed multiple times
// to add more owners.
type WithOwner Owner

func (o WithOwner) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	opts.Owners = append(opts.Owners, Owner(o))
}

// WithArgumentEvaluationConfig sets the argument evaluation configuration for
// the AnalysisRun. By default, the environment is empty, and only the builtin
// functions are available.
type WithArgumentEvaluationConfig ArgumentEvaluationConfig

func (o WithArgumentEvaluationConfig) ApplyToAnalysisRun(opts *AnalysisRunOptions) {
	opts.ExpressionConfig = (*ArgumentEvaluationConfig)(&o)
}
