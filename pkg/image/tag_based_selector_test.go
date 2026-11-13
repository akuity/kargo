package image

import (
	"regexp"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewTagBasedSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, *tagBasedSelector, error)
	}{
		{
			name: "error building base selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			name: "error compiling AllowTagsRegexes",
			sub: kargoapi.ImageSubscription{
				RepoURL:          "example/image",
				AllowTagsRegexes: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling IgnoreTagsRegexes",
			sub: kargoapi.ImageSubscription{
				RepoURL:           "example/image",
				IgnoreTagsRegexes: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling ignore tags regex")
			},
		},
		{
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL:           "example/image",
				AllowTagsRegexes:  []string{`^v2\.`},
				IgnoreTagsRegexes: []string{`^v1\.0\..*`},
			},
			assertions: func(t *testing.T, s *tagBasedSelector, err error) {
				require.NoError(t, err)
				require.NotNil(t, s.baseSelector)
				require.Len(t, s.allowTagsRegexes, 1)
				require.Equal(t, `^v2\.`, s.allowTagsRegexes[0].String())
				require.Len(t, s.ignoreTagsRegexes, 1)
				require.Equal(t, `^v1\.0\..*`, s.ignoreTagsRegexes[0].String())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newTagBasedSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_tagBasedSelector_MatchesTag(t *testing.T) {
	testCases := []struct {
		name        string
		selector    *tagBasedSelector
		tag         string
		shouldMatch bool
	}{
		{
			name:        "no regex specified",
			selector:    &tagBasedSelector{},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name: "regex matches",
			selector: &tagBasedSelector{
				allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name: "regex does not match",
			selector: &tagBasedSelector{
				allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "123",
			shouldMatch: false,
		},
		{
			name: "ignored",
			selector: &tagBasedSelector{
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name: "not ignored",
			selector: &tagBasedSelector{
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "123",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &tagBasedSelector{
				allowTagsRegexes:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.shouldMatch,
				testCase.selector.MatchesTag(testCase.tag),
			)
		})
	}
}

func Test_tagBasedSelector_filterTags(t *testing.T) {
	filtered := (&tagBasedSelector{
		allowTagsRegexes:  []*regexp.Regexp{regexp.MustCompile(`v1\.`)},
		ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
	}).filterTags([]string{
		"v1.0.0", // Allowed, but ignored
		"v1.1.0", // Allowed
		"v2.0.0", // Not allowed
	})
	require.Equal(t, []string{"v1.1.0"}, filtered)
}

func Test_tagBasedSelector_getImagesByTags(t *testing.T) {
	approved := map[string]string{"approved": "true"}
	images := []image{
		{Tag: "v3.0.0"},
		{Tag: "v2.0.0", Annotations: approved},
		{Tag: "v1.0.0", Annotations: approved},
	}
	testCases := []struct {
		name       string
		expression string
		assertions func(t *testing.T, fetched []string, imgs []image, err error)
	}{
		{
			name: "no filter expression",
			assertions: func(t *testing.T, fetched []string, imgs []image, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"v3.0.0"}, fetched)
				require.Equal(t, images[:1], imgs)
			},
		},
		{
			name:       "filter expression skips images until limit is reached",
			expression: `annotations["approved"] == "true"`,
			assertions: func(t *testing.T, fetched []string, imgs []image, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"v3.0.0", "v2.0.0"}, fetched)
				require.Equal(t, images[1:2], imgs)
			},
		},
		{
			name:       "error evaluating filter expression",
			expression: `"foo"`,
			assertions: func(t *testing.T, _ []string, _ []image, err error) {
				require.ErrorContains(t, err, "error evaluating filter expression")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newTagBasedSelector(
				kargoapi.ImageSubscription{
					RepoURL:          "example/image",
					ExpressionFilter: testCase.expression,
					DiscoveryLimit:   1,
				},
				nil,
			)
			require.NoError(t, err)
			withFakeImages(s.repoClient, images...)
			var fetched []string
			getFn := s.repoClient.remoteGetFn
			s.repoClient.remoteGetFn = func(
				ref name.Reference,
				opts ...remote.Option,
			) (*remote.Descriptor, error) {
				fetched = append(fetched, ref.Identifier())
				return getFn(ref, opts...)
			}
			imgs, err := s.getImagesByTags(
				t.Context(),
				[]string{"v3.0.0", "v2.0.0", "v1.0.0"},
			)
			testCase.assertions(t, fetched, imgs, err)
		})
	}
}
