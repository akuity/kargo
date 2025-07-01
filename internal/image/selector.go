package image

import (
	"context"
	"fmt"
	"regexp"
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
	// useful for finding the image referenced by the tag that is lexically
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

// Selector is an interface for selecting images from a container image
// repository.
type Selector interface {
	// Select selects a single image from a container image repository.
	Select(context.Context) ([]Image, error)
}

// SelectorOptions represents options for creating a Selector.
type SelectorOptions struct {
	// StrictSemvers, when set to true, will cause applicable selectors to only
	// count tags as valid semantic versions if they contain ALL of the major,
	// minor, and patch version components.
	StrictSemvers bool
	// Constraint specifies selector-specific constraints on image selection.
	Constraint string
	// AllowRegex is an optional regular expression that can be used to constrain
	// image selection based on eligible tags.
	AllowRegex string
	allowRegex *regexp.Regexp
	// Ignore is an optional list of tags that should explicitly be ignored when
	// selecting an image.
	Ignore []string
	// Platform is an optional platform constraint. If specified, the selected
	// image must match the platform constraint or Selector implementations will
	// return nil.
	Platform string
	platform *platformConstraint
	// Creds holds optional credentials for authenticating to the image
	// repository.
	Creds *Credentials
	// InsecureSkipTLSVerify is an optional flag, that if set to true, will
	// disable verification of the image repository's TLS certificate.
	InsecureSkipTLSVerify bool
	// DiscoveryLimit is an optional limit on the number of images that can be
	// discovered by the Selector. The limit is applied after filtering images
	// based on the AllowRegex and Ignore fields. If the limit is zero, all
	// discovered images will be returned.
	DiscoveryLimit int
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

	if opts.AllowRegex != "" {
		var err error
		if opts.allowRegex, err = regexp.Compile(opts.AllowRegex); err != nil {
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				opts.AllowRegex,
				err,
			)
		}
	}

	if opts.Platform != "" {
		var err error
		if opts.platform, err = parsePlatformConstraint(opts.Platform); err != nil {
			return nil, fmt.Errorf("error parsing platform constraint %q: %w", opts.Platform, err)
		}
	}

	repoClient, err := newRepositoryClient(repoURL, opts.InsecureSkipTLSVerify, opts.Creds)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating repository client for image %q: %w",
			repoURL,
			err,
		)
	}

	switch strategy {
	case SelectionStrategyDigest:
		return newDigestSelector(repoClient, *opts)
	case SelectionStrategyLexical:
		return newLexicalSelector(repoClient, *opts), nil
	case SelectionStrategyNewestBuild:
		return newNewestBuildSelector(repoClient, *opts), nil
	case SelectionStrategySemVer, "":
		return newSemVerSelector(repoClient, *opts)
	default:
		return nil, fmt.Errorf("invalid image selection strategy %q", strategy)
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
