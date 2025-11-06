package subscription

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/image"
	"github.com/akuity/kargo/pkg/logging"
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
