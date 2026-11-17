package commit

import (
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestCompileFilterExpression(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, *vm.Program, error)
	}{
		{
			name: "invalid syntax",
			sub:  kargoapi.GitSubscription{ExpressionFilter: "(1 + 2"},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.Error(t, err)
				require.Nil(t, p)
			},
		},
		{
			name: "misspelled variable",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				ExpressionFilter:        "authr == 'kent'",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.ErrorContains(t, err, "unknown name authr")
				require.Nil(t, p)
			},
		},
		{
			name: "commit variables with default strategy",
			sub: kargoapi.GitSubscription{
				ExpressionFilter: "!(author contains 'bot') && " +
					"commitDate.After(date('2025-01-01')) && " +
					"id != '' && committer != '' && subject != ''",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.NoError(t, err)
				require.NotNil(t, p)
			},
		},
		{
			name: "commit variables with NewestFromBranch",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				ExpressionFilter:        "commitDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.NoError(t, err)
				require.NotNil(t, p)
			},
		},
		{
			name: "tag variable with NewestFromBranch",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				ExpressionFilter:        "tag == 'v1.0.0'",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.ErrorContains(t, err, "unknown name tag")
				require.Nil(t, p)
			},
		},
		{
			name: "tag variables with SemVer",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				ExpressionFilter: "!(tag contains 'alpha') && " +
					"creatorDate.Year() >= 2024 && id != '' && author != '' && " +
					"committer != '' && subject != '' && tagger != '' && " +
					"annotation != ''",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.NoError(t, err)
				require.NotNil(t, p)
			},
		},
		{
			name: "commit variable with SemVer",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				ExpressionFilter:        "commitDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.ErrorContains(t, err, "unknown name commitDate")
				require.Nil(t, p)
			},
		},
		{
			name: "commit variable with Lexical",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
				ExpressionFilter:        "commitDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.ErrorContains(t, err, "unknown name commitDate")
				require.Nil(t, p)
			},
		},
		{
			name: "commit variable with NewestTag",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
				ExpressionFilter:        "commitDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.ErrorContains(t, err, "unknown name commitDate")
				require.Nil(t, p)
			},
		},
		{
			// Non-boolean results are converted to booleans at evaluation time, so
			// they must still compile.
			name: "non-boolean result",
			sub:  kargoapi.GitSubscription{ExpressionFilter: "1"},
			assertions: func(t *testing.T, p *vm.Program, err error) {
				require.NoError(t, err)
				require.NotNil(t, p)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			p, err := CompileFilterExpression(testCase.sub)
			testCase.assertions(t, p, err)
		})
	}
}

func Test_commitFilterEnv(t *testing.T) {
	t.Parallel()
	now := time.Now()
	env := commitFilterEnv(git.CommitMetadata{
		ID:         "abc123",
		CommitDate: now,
		Author:     "Author <author@example.com>",
		Committer:  "Committer <committer@example.com>",
		Subject:    "feat: add a thing",
	})
	require.Equal(
		t,
		map[string]any{
			"id":         "abc123",
			"commitDate": now,
			"author":     "Author <author@example.com>",
			"committer":  "Committer <committer@example.com>",
			"subject":    "feat: add a thing",
		},
		env,
	)
	// Every declared variable must be usable in a compiled expression.
	_, err := expr.Compile(
		"id != '' && commitDate.IsZero() == false && author != '' && "+
			"committer != '' && subject != ''",
		expr.Env(env),
	)
	require.NoError(t, err)
}

func Test_tagFilterEnv(t *testing.T) {
	t.Parallel()
	now := time.Now()
	env := tagFilterEnv(git.TagMetadata{
		Tag:         "v1.0.0",
		CommitID:    "abc123",
		CreatorDate: now,
		Author:      "Author <author@example.com>",
		Committer:   "Committer <committer@example.com>",
		Subject:     "feat: add a thing",
		Tagger:      "Tagger <tagger@example.com>",
		Annotation:  "Release v1.0.0",
	})
	require.Equal(
		t,
		map[string]any{
			"tag":         "v1.0.0",
			"id":          "abc123",
			"creatorDate": now,
			"author":      "Author <author@example.com>",
			"committer":   "Committer <committer@example.com>",
			"subject":     "feat: add a thing",
			"tagger":      "Tagger <tagger@example.com>",
			"annotation":  "Release v1.0.0",
		},
		env,
	)
}
