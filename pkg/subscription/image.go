package subscription

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/image"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/validation"
)

// CacheByTagPolicy represents a policy regarding caching of container image
// metadata using image tags as keys.
type CacheByTagPolicy string

const (
	// CacheByTagPolicyForbid indicates that caching by tag is forbidden. This
	// is silently enforced. Any container image subscription that opts into
	// caching by tag will be treated as if it does not.
	CacheByTagPolicyForbid CacheByTagPolicy = "Forbid"
	// CacheByTagPolicyAllow indicates that caching by tag is allowed. Container
	// image subscriptions may opt into caching by tag.
	CacheByTagPolicyAllow CacheByTagPolicy = "Allow"
	// CacheByTagPolicyRequire indicates that caching by tag is required.
	// Container image subscriptions must explicitly opt into caching by tag or
	// their artifact discovery processes will fail. Requiring the explicit opt-in
	// is tantamount to acknowledging the cache by tag behavior to minimize the
	// potential for developers to be taken by surprise. This option sacrifices
	// some small degree of usability for safety.
	CacheByTagPolicyRequire CacheByTagPolicy = "Require"
	// CacheByTagPolicyForce indicates that caching by tag is forced. This is
	// silently enforced. Any container image subscription that does not opt into
	// caching by tag will be treated as if it does.
	CacheByTagPolicyForce CacheByTagPolicy = "Force"
)

func init() {
	DefaultSubscriberRegistry.MustRegister(SubscriberRegistration{
		Predicate: func(
			_ context.Context,
			sub kargoapi.RepoSubscription,
		) (bool, error) {
			return sub.Image != nil, nil
		},
		Value:    newImageSubscriber,
		Metadata: SubscriberMetadata{Schema: mustLoadSchema("image")},
	})
}

// imageSubscriber is an implementation of the Subscriber interface that
// discovers container images from a container image repository.
type imageSubscriber struct {
	credentialsDB    credentials.Database
	cacheByTagPolicy CacheByTagPolicy

	// newSelectorFn constructs the image Selector for a subscription. It is a
	// field so tests can substitute a fake Selector for the real one.
	newSelectorFn func(
		ctx context.Context,
		sub kargoapi.ImageSubscription,
		creds *image.Credentials,
	) (image.Selector, error)
}

// newImageSubscriber returns an implementation of the Subscriber interface that
// discovers container images from a container image repository.
func newImageSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &imageSubscriber{
		credentialsDB: credentialsDB,
		cacheByTagPolicy: CacheByTagPolicy(
			os.GetEnv(
				"CACHE_BY_TAG_POLICY",
				string(CacheByTagPolicyAllow),
			),
		),
		newSelectorFn: image.NewSelector,
	}, nil
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

// ValidateSubscription implements Subscriber. The subscription's schema
// covers everything else.
func (i *imageSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	sub := s.Image
	var errs field.ErrorList

	// Validate constraint as semver if using SemVer strategy
	if sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategySemVer ||
		sub.ImageSelectionStrategy == "" {
		if sub.Constraint != "" {
			if err := validation.SemverConstraint(
				f.Child("constraint"),
				sub.Constraint,
			); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Validate cache settings
	if sub.ImageSelectionStrategy != kargoapi.ImageSelectionStrategyDigest &&
		!sub.CacheByTag &&
		i.cacheByTagPolicy == CacheByTagPolicyRequire {
		errs = append(
			errs,
			field.Invalid(
				f.Child("cacheByTag"),
				sub.CacheByTag,
				"caching image metadata by tag is required by controller "+
					"configuration; enable with caution as this feature is safe only "+
					"for subscriptions not involving \"mutable\" tags",
			),
		)
	}

	return errs
}

// DiscoverArtifacts implements Subscriber.
func (i *imageSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
	_ any,
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

	switch i.cacheByTagPolicy {
	case CacheByTagPolicyForbid:
		imgSub.CacheByTag = false
	case CacheByTagPolicyAllow:
		// Leave as is
	case CacheByTagPolicyRequire:
		if !imgSub.CacheByTag {
			return nil, fmt.Errorf(
				"caching image metadata by tag is required by controller " +
					"configuration; enable with caution as this feature is safe only " +
					"for subscriptions not involving \"mutable\" tags",
			)
		}
	case CacheByTagPolicyForce:
		imgSub.CacheByTag = true
	}

	selector, err := i.newSelectorFn(ctx, *imgSub, regCreds)
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
		RepoURL:          imgSub.RepoURL,
		Platform:         imgSub.Platform,
		References:       images,
		SubscriptionName: sub.Name,
	}, nil
}
