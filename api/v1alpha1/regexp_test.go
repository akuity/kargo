package v1alpha1_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/api/v1alpha1/testhelper"
)

// ValidateResourceExpression checks the regular expression identified by marker
// against the provided test cases.
func ValidateResourceExpression(t *testing.T, marker string, testCases map[string]bool) {
	expression, err := findExpression(marker)
	require.NoError(t, err)
	testhelper.ValidateRegularExpression(t, expression, testCases)
}

func TestBranchPattern(t *testing.T) {
	testCases := map[string]bool{
		"":             false,
		"foo/bar":      true,
		"foo.bar":      true,
		"release-0.58": true,
		"/foo":         false,
		"foo/":         false,
		"foo//bar":     true,
		".foo":         false,
		"foo.":         false,
		"foo..bar":     true,
	}

	ValidateResourceExpression(t, "Branch", testCases)
}

func TestDigestPattern(t *testing.T) {
	testCases := map[string]bool{
		"":                        false,
		"sha256:":                 false,
		"sha256:1234567890abcdef": true,
		":1234567890abcdef":       false,
		"sha256:xyz":              false,
		"xyz:1234567890abcdef":    true,
		"sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef": true,
	}

	ValidateResourceExpression(t, "Digest", testCases)
}

func TestDurationPattern(t *testing.T) {
	testCases := map[string]bool{
		"":  false,
		"s": false,
		".": false,
		// simple cases
		"10s": true,
		"20h": true,
		// Decimal cases
		"1.5s": true,
		".5s":  false,
		"0.5s": true,
		"1.5":  false,
		"1.5h": true,
		"1.5m": true,
		"1,4s": false,
		// edge cases
		"1":    false,
		"0":    false,
		"-1":   false,
		"-1s":  false,
		"+33s": false,
	}

	ValidateResourceExpression(t, "Duration", testCases)
}

func TestHelmRepoURLPattern(t *testing.T) {
	testCases := map[string]bool{}

	ValidateResourceExpression(t, "HelmRepoURL", testCases)
}

func TestGitRepoURLPattern(t *testing.T) {
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
		"https://github.com/":         true, //?
		"https://foo:bar@github.com":  false,
		"https://foo:bar@github.com/": true, //?
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
		"ssh://git.example.com/":     true, //?
		"ssh://git@git.example.com":  false,
		"ssh://git@git.example.com/": true, //?
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

	ValidateResourceExpression(t, "GitRepoURL", testCases)
}

func TestImageRepoURLPattern(t *testing.T) {
	testCases := map[string]bool{
		"":  false,
		".": false,
	}

	ValidateResourceExpression(t, "ImageRepoURL", testCases)
}

func TestKubernetesNamePattern(t *testing.T) {
	testCases := map[string]bool{
		"": false,
		// simple cases
		"abc":             true,
		"abc.123":         true,
		"abc-123-abc-123": true,
		// edge cases
		"abc-123-abc-123-": false,
		"-abc-123-abc-123": false,
		// invalid characters
		"abc+123": false,
		"abc 123": false,
		"abc/123": false,
		"abc_123": false,
		"abc:123": false,
	}

	ValidateResourceExpression(t, "KubernetesName", testCases)
}

func TestTagPattern(t *testing.T) {
	testCases := map[string]bool{
		"":  false,
		".": true, // ?
		// simple cases
		"v1.0.0":       true,
		"v1.0.0-alpha": true,
		"latest":       true,
		// invalid characters
		"plus+one":  false,
		"not a tag": false,
	}

	expression, err := findExpression("Tag")
	if err != nil {
		t.Fatal(err)
	}

	for tt, expected := range testCases {
		t.Run(tt, func(t *testing.T) {
			require.Equal(t, expected, expression.MatchString(tt))
		})
	}
}

var patternRE = regexp.MustCompile("// ?\\+kubebuilder:validation:Pattern=`(.*)`")

// findExpression returns the regular expression that is used by the types in
// the current directory, that have the provided marker. A marker is part of the
// directive "// +akuity:test-kubebuilder-pattern=FOO", where FOO is the marker.
// The marker directive must be paired with the kubebuilder regular expression
// pattern validation ("// +kubebuilder:validation:Pattern=`BAR`"). When this
// function is provided "FOO", it will return the regular expression "BAR". It
// is an error if there are no marker directives that match. It is an error if
// several marker directives match unless all of the expressions are identical.
func findExpression(marker string) (*regexp.Regexp, error) {
	fset := token.NewFileSet()

	// Find all types file in the current directory
	pkgs, err := parser.ParseDir(fset, "./", func(fi fs.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), "_types.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	previouslyFoundExpression := ""
	markerRE := regexp.MustCompile("// ?\\+akuity:test-kubebuilder-pattern=" + marker) //nolint:gosimple

	firstLocation := ""

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			cmap := ast.NewCommentMap(fset, f, f.Comments)
			for _, list := range cmap.Comments() {
				foundDirective := false
				commentLocation := ""
				foundExpression := ""

				for _, comment := range list.List {
					if markerRE.MatchString(comment.Text) {
						foundDirective = true
						commentLocation = fset.Position(comment.Pos()).String()
					}
					if patterns := patternRE.FindStringSubmatch(comment.Text); len(patterns) > 1 {
						foundExpression = patterns[1]
					}
				}

				if foundDirective {
					if previouslyFoundExpression == "" {
						previouslyFoundExpression = foundExpression
						firstLocation = commentLocation
					} else if foundExpression == "" {
						return nil, fmt.Errorf("marker at %s had no regular expression", commentLocation)
					} else if previouslyFoundExpression != foundExpression {
						return nil, fmt.Errorf("regexps marked %s are not in sync: %s and %s, %q and %q",
							marker, firstLocation, commentLocation, previouslyFoundExpression, foundExpression)
					}
				}
			}
		}
	}

	if previouslyFoundExpression == "" {
		return nil, fmt.Errorf("no regexp found marked %s", marker)
	}

	return regexp.Compile(previouslyFoundExpression)
}
