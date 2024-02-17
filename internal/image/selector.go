package image

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
)

// SelectionStrategy represents a strategy for selecting a single image from a
// container image repository.
type SelectionStrategy string

const (
	// SelectionStrategyDigest represents an image selection strategy that is
	// useful for finding the digest of a container image that is currently
	// referenced by a mutable tag, e.g. latest. This strategy requires the use of
	// a constraint that must exactly match the name of a, presumably, mutable
	// tag.
	SelectionStrategyDigest SelectionStrategy = "Digest"
	// SelectionStrategyLexical represents an image selection strategy that is
	// useful for finding the the image referenced by the tag that is lexically
	// last among those matched by a regular expression and not explicitly
	// ignored. This strategy is useful for finding the images referenced by the
	// latest in a series of tag that are suffixed with a predictably formatted
	// timestamp.
	SelectionStrategyLexical SelectionStrategy = "Lexical"
	// SelectionStrategyNewestBuild represents an image selection strategy that is
	// useful for finding the image that was most recently pushed to the image
	// repository. This is the least efficient strategy because it can require the
	// retrieval of many manifests from the image repository. It is best to use
	// this strategy with caution and constrain the eligible tags as much as
	// possible using a regular expression.
	SelectionStrategyNewestBuild SelectionStrategy = "NewestBuild"
	// SelectionStrategySemVer represents an image selection strategy that is
	// useful for finding the images referenced by the tag is the highest among
	// tags from the repository that are valid semantic versions. An optional
	// constraint can limit the eligible range of semantic versions.
	SelectionStrategySemVer SelectionStrategy = "SemVer"
)

// Selector is an interface for selecting a single image from a container image
// repository.
type Selector interface {
	// Select selects a single image from a container image repository.
	Select(context.Context) (*Image, error)
}

// SelectorOptions represents options for creating a Selector.
type SelectorOptions struct {
	// Constraint holds a selection strategy-specific value for constraining image
	// selection.
	Constraint string
	// AllowRegex is an optional regular expression that can be used to constrain
	// image selection based on eligible tags.
	AllowRegex string
	// Ignore is an optional list of tags that should explicitly be ignored when
	// selecting an image.
	Ignore []string
	// Platform is an optional platform constraint. If specified, the selected
	// image must match the platform constraint or Selector implementations will
	// return nil a image.
	Platform string
	// Creds holds optional credentials for authenticating to the image
	// repository.
	Creds *Credentials
}

// NewSelector returns some implementation of the Selector interface that
// selects a single image from a container image repository based on a selection
// strategy and a set of optional constraints.
func NewSelector(
	repoURL string,
	strategy SelectionStrategy,
	opts *SelectorOptions,
) (Selector, error) {
	if opts == nil {
		opts = &SelectorOptions{}
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
	case SelectionStrategyDigest:
		return newDigestSelector(repoClient, opts.Constraint, platform)
	case SelectionStrategyLexical:
		return newLexicalSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			platform,
		), nil
	case SelectionStrategyNewestBuild:
		return newNewestBuildSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			platform,
		), nil
	case SelectionStrategySemVer, "":
		return newSemVerSelector(
			repoClient,
			allowRegex,
			opts.Ignore,
			opts.Constraint,
			platform,
		)
	default:
		return nil, errors.Errorf("invalid image selection strategy %q", strategy)
	}
}

// allowsTag returns true if the given tag matches the given regular expression
// or if the regular expression is nil. It returns false otherwise.
func allowsTag(tag string, allowRegex *regexp.Regexp) bool {
	if allowRegex == nil {
		return true
	}
	return allowRegex.MatchString(tag)
}

// ignoresTag returns true if the given tag is in the given list of ignored
// tags. It returns false otherwise.
func ignoresTag(tag string, ignore []string) bool {
	for _, i := range ignore {
		if i == tag {
			return true
		}
	}
	return false
}
