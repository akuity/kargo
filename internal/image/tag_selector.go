package image

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
)

// TagSelectionStrategy represents a strategy for selecting a single tag from a
// container image repository.
type TagSelectionStrategy string

const (
	// TagSelectionStrategyDigest represents a tag selection strategy that is
	// useful for finding the digest of a container image that is currently
	// referenced by a mutable tag, e.g. latest. This strategy requires the use of
	// a constraint that must exactly match the name of a, presumably, mutable
	// tag.
	TagSelectionStrategyDigest TagSelectionStrategy = "Digest"
	// TagSelectionStrategyLexical represents a tag selection strategy that
	// is useful for finding the tag that is lexically last among those matched
	// by a regular expression and not explicitly ignored. This strategy is useful
	// for finding the latest in a series of tag names that are suffixed with a
	// predictably formatted timestamp.
	TagSelectionStrategyLexical TagSelectionStrategy = "Lexical"
	// TagSelectionStrategyNewestBuild represents a tag selection strategy that is
	// useful for finding the tag that was most recently pushed to the image
	// repository. This is the east efficient strategy because it can require the
	// retrieval of many manifests from the image repository. It is best to use
	// this strategy with caution and constrain the eligible tags as much as
	// possible using a regular expression.
	TagSelectionStrategyNewestBuild TagSelectionStrategy = "NewestBuild"
	// TagSelectionStrategySemVer represents a tag selection strategy that is
	// useful for finding the tag is the highest among tags from the repository
	// that are valid semantic versions. An optional constraint can limit the
	// eligible range of semantic versions.
	TagSelectionStrategySemVer TagSelectionStrategy = "SemVer"
)

// TagSelector is an interface for selecting a single tag from a container image
// repository.
type TagSelector interface {
	// SelectTag selects a single tag from a container image repository.
	SelectTag(context.Context) (*Tag, error)
}

// TagSelectorOptions represents options for creating a TagSelector.
type TagSelectorOptions struct {
	// Constraint holds a selection strategy-specific value for constraining the
	// eligible tags.
	Constraint string
	// AllowRegex is an optional regular expression that can be used to constrain
	// the eligible tags.
	AllowRegex string
	// Ignore is an optional list of tag names that should explicitly be ignored.
	Ignore []string
	// Platform is an optional platform constraint. If specified, it does not
	// impact what tags are eligible for selection, but the selected tag must
	// match the platform constraint or TagSelector implementations will return
	// nil a tag.
	Platform string
	// Creds holds optional credentials for authenticating to the image
	// repository.
	Creds *Credentials
}

// NewTagSelector returns some implementation of the TagSelector interface that
// selects a single tag from a container image repository based on a tag
// selection strategy and a set of optional constraints.
func NewTagSelector(
	repoURL string,
	strategy TagSelectionStrategy,
	opts *TagSelectorOptions,
) (TagSelector, error) {
	if opts == nil {
		opts = &TagSelectorOptions{}
	}

	var allowRegex *regexp.Regexp
	if opts.AllowRegex != "" {
		var err error
		if allowRegex, err = regexp.Compile(opts.AllowRegex); err != nil {
			return nil, errors.Wrapf(
				err,
				"error compiling regular expression %q",
				opts.AllowRegex,
			)
		}
	}

	var platform *platformConstraint
	if opts.Platform != "" {
		p, err := parsePlatformConstraint(opts.Platform)
		if err != nil {
			return nil,
				errors.Wrapf(err, "error parsing platform constraint %q", opts.Platform)
		}
		platform = &p
	}

	repoClient, err := newRepositoryClient(repoURL, opts.Creds)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error creating repository client for image %q",
			repoURL,
		)
	}

	switch strategy {
	case TagSelectionStrategyDigest:
		return newDigestTagSelector(repoClient, opts.Constraint, platform)
	case TagSelectionStrategyLexical:
		return newLexicalTagSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			platform,
		), nil
	case TagSelectionStrategyNewestBuild:
		return newNewestBuildTagSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			platform,
		), nil
	case TagSelectionStrategySemVer, "":
		return newSemVerTagSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			opts.Constraint,
			platform,
		)
	default:
		return nil, errors.Errorf("invalid tag selection strategy %q", strategy)
	}
}

// allows returns true if the given tag name matches the given regular
// expression or if the regular expression is nil. It returns false otherwise.
func allows(tagName string, allowRegex *regexp.Regexp) bool {
	if allowRegex == nil {
		return true
	}
	return allowRegex.MatchString(tagName)
}

// ignores returns true if the given tag name is in the given list of ignored
// tag names. It returns false otherwise.
func ignores(tagName string, ignore []string) bool {
	for _, i := range ignore {
		if i == tagName {
			return true
		}
	}
	return false
}
