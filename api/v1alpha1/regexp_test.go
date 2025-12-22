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

	testingPkg "github.com/akuity/kargo/api/testing"
)

// ValidateResourceExpression checks the regular expression identified by marker
// against the provided test cases.
func ValidateResourceExpression(t *testing.T, marker string, testCases map[string]bool) {
	expression, err := findExpression(marker)
	require.NoError(t, err)
	testingPkg.ValidateRegularExpression(t, expression, testCases)
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
