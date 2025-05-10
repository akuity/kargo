package v1alpha1_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitSubscription_RepoURL(t *testing.T) {
	// most test cases are taken from internal/git/git_test.go
	testCases := map[string]bool{
		"":             false,
		":":            false,
		"/etc/passwd":  false,
		"//etc/passwd": false,
		"https:":       false,

		"https://not a url":                      false,
		"http://github.com/example/repo?foo=bar": true,
		"ssh://not a url":                        false,
		"ssh://github.com/example/repo?foo=bar":  true,
		"not even remotely a url":                false,
		// URLs of the form http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
		"https://github.com":          false,
		"https://github.com/":         true,
		"https://foo:bar@github.com":  false,
		"https://foo:bar@github.com/": true,
		// Variable features of the following URLs:
		//   1000. Outbound proxy or not
		//   0100. Port number or not
		//   0010. .git suffix or not
		//   0001. Trailing slash or not
		"https://github.com/example/repo":                  true,
		"https://github.com/example/repo/":                 true,
		"https://github.com/example/repo.git":              true,
		"https://github.com/example/repo.git/":             true,
		"https://localhost:8443/example/repo":              true,
		"https://localhost:8443/example/repo/":             true,
		"https://localhost:8443/example/repo.git":          true,
		"https://localhost:8443/example/repo.git/":         true,
		"https://foo:bar@github.com/example/repo":          true,
		"https://foo:bar@github.com/example/repo/":         true,
		"https://foo:bar@github.com/example/repo.git":      true,
		"https://foo:bar@github.com/example/repo.git/":     true,
		"https://foo:bar@localhost:8443/example/repo":      true,
		"https://foo:bar@localhost:8443/example/repo/":     true,
		"https://foo:bar@localhost:8443/example/repo.git":  true,
		"https://foo:bar@localhost:8443/example/repo.git/": true,
		// URLS of the form ssh://[user@]host.xz[:port][/path/to/repo[.git][/]]
		"ssh://git.example.com":      false,
		"ssh://git.example.com/":     true, //??
		"ssh://git@git.example.com":  false,
		"ssh://git@git.example.com/": true, //??
		// Variable features of the following URLs:
		//   1000. Username or not
		//   0100. Port number or not
		//   0010. .git suffix or not
		//   0001. Trailing slash or not
		"ssh://github.com/example/repo":              true,
		"ssh://github.com/example/repo/":             true,
		"ssh://github.com/example/repo.git":          true,
		"ssh://github.com/example/repo.git/":         true,
		"ssh://localhost:2222/example/repo":          true,
		"ssh://localhost:2222/example/repo/":         true,
		"ssh://localhost:2222/example/repo.git":      true,
		"ssh://localhost:2222/example/repo.git/":     true,
		"ssh://git@github.com/example/repo":          true,
		"ssh://git@github.com/example/repo/":         true,
		"ssh://git@github.com/example/repo.git":      true,
		"ssh://git@github.com/example/repo.git/":     true,
		"ssh://git@localhost:2222/example/repo":      true,
		"ssh://git@localhost:2222/example/repo/":     true,
		"ssh://git@localhost:2222/example/repo.git":  true,
		"ssh://git@localhost:2222/example/repo.git/": true,
		// SCP-style URLs of the form [user@]host.xz[:path/to/repo[.git][/]]
		"git.example.com":     false,
		"git@git.example.com": false,
		// Variable features of the following URLs:
		//   100. Username or not
		//   010. .git suffix or not
		//   001. Trailing slash or not
		"github.com:example/repo":          false, //?
		"github.com:example/repo/":         false, //?
		"github.com:example/repo.git":      false, //?
		"github.com:example/repo.git/":     false, //?
		"git@github.com:example/repo":      true,
		"git@github.com:example/repo/":     true,
		"git@github.com:example/repo.git":  true,
		"git@github.com:example/repo.git/": true,
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "warehouse_types.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	patternRE := regexp.MustCompile("// ?\\+kubebuilder:validation:Pattern=`(.*)`")
	markerRE := regexp.MustCompile("// ?\\+akuity:IsGitRepoURL") //nolint:gosimple

	expression := ""

	cmap := ast.NewCommentMap(fset, f, f.Comments)
	for _, list := range cmap.Comments() {
		foundExpression := ""
		foundMarker := false
		for _, comment := range list.List {
			if markerRE.MatchString(comment.Text) {
				foundMarker = true
			}
			if patterns := patternRE.FindStringSubmatch(comment.Text); len(patterns) > 1 {
				foundExpression = patterns[1]
			}
		}

		if foundMarker {
			if expression == "" {
				expression = foundExpression
			} else if expression != foundExpression {
				t.Fatalf("multiple RepoURL regexes found: %q and %q", expression, foundExpression)
			}
		}
	}

	require.NotEmpty(t, expression, "could not find akuity marker in source file")

	re := regexp.MustCompile(expression)

	for url, valid := range testCases {
		t.Run(url, func(t *testing.T) {
			require.Equal(t, valid, re.MatchString(url))
		})
	}
}
