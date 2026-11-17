package commit

import (
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

// CompileFilterExpression compiles the provided Git subscription's filter
// expression against the variables available to it when it is evaluated,
// which depend on the subscription's commit selection strategy. Referencing
// any other variable is an error. It is exported so that admission can reject
// expressions that would fail here.
func CompileFilterExpression(
	sub kargoapi.GitSubscription,
) (*vm.Program, error) {
	var env map[string]any
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategyLexical,
		kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategySemVer:
		env = tagFilterEnv(git.TagMetadata{})
	default:
		env = commitFilterEnv(git.CommitMetadata{})
	}
	return expr.Compile(sub.ExpressionFilter, expr.Env(env))
}

// commitFilterEnv returns the variables available to a filter expression
// evaluated against a commit. It is used both to declare those variables when
// compiling the expression and to supply their values when evaluating it, so
// the two can't drift apart.
func commitFilterEnv(commit git.CommitMetadata) map[string]any {
	return map[string]any{
		"id":         commit.ID,
		"commitDate": commit.CommitDate,
		"author":     commit.Author,
		"committer":  commit.Committer,
		"subject":    commit.Subject,
	}
}

// tagFilterEnv returns the variables available to a filter expression
// evaluated against a tag. It is used both to declare those variables when
// compiling the expression and to supply their values when evaluating it, so
// the two can't drift apart.
func tagFilterEnv(tag git.TagMetadata) map[string]any {
	return map[string]any{
		"tag":         tag.Tag,
		"id":          tag.CommitID,
		"creatorDate": tag.CreatorDate,
		"author":      tag.Author,
		"committer":   tag.Committer,
		"subject":     tag.Subject,
		"tagger":      tag.Tagger,
		"annotation":  tag.Annotation,
	}
}
