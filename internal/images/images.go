package images

import (
	"log"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	argoLog "github.com/argoproj-labs/argocd-image-updater/pkg/log"
	"github.com/argoproj-labs/argocd-image-updater/pkg/options"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/pkg/errors"
)

type ImageUpdateStrategy string

const (
	ImageUpdateStrategySemVer ImageUpdateStrategy = "SemVer"
	ImageUpdateStrategyLatest ImageUpdateStrategy = "Latest"
	ImageUpdateStrategyName   ImageUpdateStrategy = "Name"
	ImageUpdateStrategyDigest ImageUpdateStrategy = "Digest"
)

func init() {
	err := argoLog.SetLogLevel("ERROR")
	if err != nil {
		log.Fatal(err)
	}
}

func GetLatestTag(
	repoURL string,
	updateStrategy ImageUpdateStrategy,
	semverConstraint string,
	allowTags string,
	ignoreTags []string,
	platform string,
	creds *Credentials,
) (string, error) {
	img := image.NewFromIdentifier(repoURL)
	vc := &image.VersionConstraint{
		Constraint: semverConstraint,
		Strategy:   img.ParseUpdateStrategy(string(updateStrategy)),
	}
	if allowTags != "" {
		vc.MatchFunc, vc.MatchArgs = img.ParseMatchfunc(allowTags)
	}
	vc.IgnoreList = ignoreTags
	vc.Options = options.NewManifestOptions()
	if platform != "" {
		os, arch, variant, err := image.ParsePlatform(platform)
		if err != nil {
			return "", errors.Wrapf(
				err,
				"error parsing platform %q for image %q",
				platform,
				repoURL,
			)
		}
		vc.Options = vc.Options.WithPlatform(os, arch, variant)
	}
	vc.Options = vc.Options.WithMetadata(vc.Strategy.NeedsMetadata())

	rep, err := registry.GetRegistryEndpoint(img.RegistryURL)
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error getting container registry endpoint for image %q",
			repoURL,
		)
	}

	if creds == nil {
		creds = &Credentials{}
	}
	regClient, err := registry.NewClient(rep, creds.Username, creds.Password)
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error creating registry client for image %q",
			repoURL,
		)
	}

	tags, err := rep.GetTags(img, regClient, vc)
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error fetching tags for image %q",
			repoURL,
		)
	}

	tag, err := getNewestVersionFromTags(img, vc, tags)
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error finding newest tag for %q",
			repoURL,
		)
	}
	if tag == "" {
		return "", errors.Errorf(
			"found no suitable version of image %q",
			repoURL,
		)
	}

	return tag, nil
}
