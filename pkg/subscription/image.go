package subscription

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/image"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/validation"
)

func init() {
	DefaultSubscriberRegistry.MustRegister(SubscriberRegistration{
		Predicate: func(
			_ context.Context,
			sub kargoapi.RepoSubscription,
		) (bool, error) {
			return sub.Image != nil, nil
		},
		Value: newImageSubscriber,
	})
}

// imageSubscriber is an implementation of the Subscriber interface that
// discovers container images from a container image repository.
type imageSubscriber struct {
	credentialsDB credentials.Database
}

// newImageSubscriber returns an implementation of the Subscriber interface that
// discovers container images from a container image repository.
func newImageSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &imageSubscriber{credentialsDB: credentialsDB}, nil
}

// ApplySubscriptionDefaults implements Subscriber.
func (i *imageSubscriber) ApplySubscriptionDefaults(
	_ context.Context,
	sub *kargoapi.RepoSubscription,
) error {
	if sub == nil || sub.Image == nil {
		return nil
	}
	if sub.Image.ImageSelectionStrategy == "" {
		sub.Image.ImageSelectionStrategy = kargoapi.ImageSelectionStrategySemVer
	}
	if sub.Image.StrictSemvers == nil {
		sub.Image.StrictSemvers = ptr.To(true)
	}
	if sub.Image.DiscoveryLimit == 0 {
		sub.Image.DiscoveryLimit = 20
	}
	return nil
}

var (
	imageRepoURLRegex = regexp.MustCompile(`^(\w+([\.-]\w+)*(:[\d]+)?/)?(\w+([\.-]\w+)*)(/\w+([\.-]\w+)*)*$`)

	validImageSelectionStrategies = []kargoapi.ImageSelectionStrategy{
		kargoapi.ImageSelectionStrategyDigest,
		kargoapi.ImageSelectionStrategyLexical,
		kargoapi.ImageSelectionStrategyNewestBuild,
		kargoapi.ImageSelectionStrategySemVer,
	}
)

// ValidateSubscription implements Subscriber.
func (i *imageSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	// TODO(krancour): Longer term, we might want to start doing this with JSON
	// schema.

	sub := s.Image
	var errs field.ErrorList

	// Validate RepoURL: MinLength=1, Pattern (Image repo URL regex)
	if err := validation.MinLength(f.Child("repoURL"), sub.RepoURL, 1); err != nil {
		errs = append(errs, err)
	}
	if !imageRepoURLRegex.MatchString(sub.RepoURL) {
		errs = append(errs, field.Invalid(
			f.Child("repoURL"),
			sub.RepoURL,
			"must be a valid image repository URL",
		))
	}

	// Validate ImageSelectionStrategy
	if sub.ImageSelectionStrategy != "" {
		if err := validateImageSelectionStrategy(
			f.Child("imageSelectionStrategy"),
			sub.ImageSelectionStrategy,
		); err != nil {
			errs = append(errs, err)
		}
	}

	// If imageSelectionStrategy is Digest, constraint must be set
	if sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategyDigest && sub.Constraint == "" {
		errs = append(errs, field.Invalid(
			f.Child("constraint"),
			sub.Constraint,
			"must be set when imageSelectionStrategy is Digest",
		))
	}

	// Validate constraint as semver if using SemVer strategy
	if sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategySemVer || sub.ImageSelectionStrategy == "" {
		if sub.Constraint != "" {
			if err := validation.SemverConstraint(
				f.Child("constraint"),
				sub.Constraint,
			); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Validate Platform
	if sub.Platform != "" {
		if !image.ValidatePlatformConstraint(sub.Platform) {
			errs = append(errs, field.Invalid(
				f.Child("platform"),
				sub.Platform,
				"must be in format <os>/<arch>",
			))
		}
	}

	// Validate DiscoveryLimit: Minimum=1, Maximum=100
	if sub.DiscoveryLimit < 1 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be >= 1",
		))
	} else if sub.DiscoveryLimit > 100 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be <= 100",
		))
	}

	return errs
}

func validateImageSelectionStrategy(
	f *field.Path,
	strategy kargoapi.ImageSelectionStrategy,
) *field.Error {
	if !slices.Contains(validImageSelectionStrategies, strategy) {
		return field.NotSupported(f, strategy, []string{
			string(kargoapi.ImageSelectionStrategyDigest),
			string(kargoapi.ImageSelectionStrategyLexical),
			string(kargoapi.ImageSelectionStrategyNewestBuild),
			string(kargoapi.ImageSelectionStrategySemVer),
		})
	}
	return nil
}

// DiscoverArtifacts implements Subscriber.
func (i *imageSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
) (any, error) {
	imgSub := sub.Image

	if imgSub == nil {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx).WithValues("repo", imgSub.RepoURL)

	// Obtain credentials for the image repository.
	creds, err := i.credentialsDB.Get(
		ctx,
		project,
		credentials.TypeImage,
		imgSub.RepoURL,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining credentials for image repo %q: %w",
			imgSub.RepoURL,
			err,
		)
	}
	var regCreds *image.Credentials
	if creds != nil {
		regCreds = &image.Credentials{
			Username: creds.Username,
			Password: creds.Password,
		}
		logger.Debug("obtained credentials for image repo")
	} else {
		logger.Debug("found no credentials for image repo")
	}

	selector, err := image.NewSelector(ctx, *imgSub, regCreds)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining selector for image %q: %w",
			imgSub.RepoURL, err,
		)
	}
	images, err := selector.Select(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error discovering newest applicable images %q: %w",
			imgSub.RepoURL, err,
		)
	}
	if len(images) == 0 {
		logger.Debug("discovered no images")
	} else {
		logger.Debug("discovered images", "count", len(images))
	}

	return kargoapi.ImageDiscoveryResult{
		RepoURL:    imgSub.RepoURL,
		Platform:   imgSub.Platform,
		References: images,
	}, nil
}
