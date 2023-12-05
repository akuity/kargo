package image

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/manifestlist"
	"github.com/distribution/distribution/v3/manifest/ocischema"
	"github.com/distribution/distribution/v3/manifest/schema1" //nolint: staticcheck
	"github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/distribution/distribution/v3/reference"
	"github.com/distribution/distribution/v3/registry/client"
	"github.com/distribution/distribution/v3/registry/client/auth"
	"github.com/distribution/distribution/v3/registry/client/auth/challenge"
	"github.com/distribution/distribution/v3/registry/client/transport"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"go.uber.org/ratelimit"
	"golang.org/x/sync/semaphore"

	"github.com/akuity/kargo/internal/logging"
)

// maxMetadataConcurrency is the maximum number of concurrent goroutines that
// can be used to fetch metadata. Per Go's documentation, goroutines are very
// cheap (practical to spawn tens of thousands), AND we have a rate limiter in
// place for each registry, so there's no reason for this number not to be
// pretty large.
const maxMetadataConcurrency = 1000

var metaSem = semaphore.NewWeighted(maxMetadataConcurrency)

// knownMediaTypes is the list of supported media types.
var knownMediaTypes = []string{
	// V!
	schema1.MediaTypeSignedManifest, //nolint: staticcheck
	// V2
	schema2.SchemaVersion.MediaType,
	manifestlist.SchemaVersion.MediaType,
	// OCI
	ocischema.SchemaVersion.MediaType,
	ociv1.MediaTypeImageIndex,
}

// repositoryClient is a client for retrieving tag information from a specific
// image container repository.
type repositoryClient struct {
	registry *registry
	image    string
	repo     distribution.Repository

	// The following behaviors are overridable for testing purposes:

	getTagByNameFn func(
		context.Context,
		string,
		*platformConstraint,
	) (*Tag, error)

	getTagByDigestFn func(
		context.Context,
		digest.Digest,
		*platformConstraint,
	) (*Tag, error)

	getManifestByTagNameFn func(
		context.Context,
		string,
	) (distribution.Manifest, error)

	getManifestByDigestFn func(
		context.Context,
		digest.Digest,
	) (distribution.Manifest, error)

	extractTagFromManifestFn func(
		context.Context,
		distribution.Manifest,
		*platformConstraint,
	) (*Tag, error)

	extractTagFromV1ManifestFn func(
		*schema1.SignedManifest, // nolint: staticcheck
		*platformConstraint,
	) (*Tag, error)

	extractTagFromV2ManifestFn func(
		context.Context,
		*schema2.DeserializedManifest,
		*platformConstraint,
	) (*Tag, error)

	extractTagFromOCIManifestFn func(
		context.Context,
		*ocischema.DeserializedManifest,
		*platformConstraint,
	) (*Tag, error)

	extractTagFromCollectionFn func(
		context.Context,
		distribution.Manifest,
		*platformConstraint,
	) (*Tag, error)

	getBlobFn func(context.Context, digest.Digest) ([]byte, error)
}

// newRepositoryClient parses the provided repository URL to infer registry
// information and image name. This information is used to initialize and
// return a new repository client.
func newRepositoryClient(
	repoURL string,
	creds *Credentials,
) (*repositoryClient, error) {
	repoRef, err := reference.ParseNormalizedNamed(repoURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing image repo URL %s", repoURL)
	}
	registryURL := reference.Domain(repoRef)
	reg := getRegistry(registryURL)
	image := reg.normalizeImageName(reference.Path(repoRef))
	apiAddress := strings.TrimSuffix(reg.apiAddress, "/")

	challengeManager, err := getChallengeManager(
		apiAddress,
		&rateLimitedRoundTripper{
			limiter:              reg.rateLimiter,
			internalRoundTripper: http.DefaultTransport,
		},
	)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error getting challenge manager for %s", apiAddress)
	}

	if creds == nil {
		creds = &Credentials{}
	}

	rlt := &rateLimitedRoundTripper{
		limiter: reg.rateLimiter,
		internalRoundTripper: transport.NewTransport(
			http.DefaultTransport,
			auth.NewAuthorizer(
				challengeManager,
				auth.NewTokenHandler(
					http.DefaultTransport,
					creds,
					image,
					"pull",
				),
				auth.NewBasicHandler(creds),
			),
		),
	}

	imageRef, err := reference.WithName(image)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting reference for image %q", image)
	}
	repo, err := client.NewRepository(imageRef, apiAddress, rlt)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error creating internal repository for image %q in registry %s",
			image,
			apiAddress,
		)
	}

	r := &repositoryClient{
		registry: reg,
		image:    image,
		repo:     repo,
	}

	r.getTagByNameFn = r.getTagByName
	r.getTagByDigestFn = r.getTagByDigest
	r.getManifestByTagNameFn = r.getManifestByTagName
	r.getManifestByDigestFn = r.getManifestByDigest
	r.extractTagFromManifestFn = r.extractTagFromManifest
	r.extractTagFromV1ManifestFn = r.extractTagFromV1Manifest
	r.extractTagFromV2ManifestFn = r.extractTagFromV2Manifest
	r.extractTagFromOCIManifestFn = r.extractTagFromOCIManifest
	r.extractTagFromCollectionFn = r.extractTagFromCollection
	r.getBlobFn = r.getBlob

	return r, nil
}

// getChallengeManager makes an initial request to a registry's API v2 endpoint.
// The response is used to configure a challenge manager, which is returned.
//
// Defining it this way makes it easy to override for testing purposes.
var getChallengeManager = func(
	apiAddress string,
	roundTripper http.RoundTripper,
) (challenge.Manager, error) {
	httpClient := &http.Client{
		Transport: roundTripper,
	}
	apiAddress = fmt.Sprintf("%s/v2/", apiAddress)
	resp, err := httpClient.Get(apiAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting %s", apiAddress)
	}
	defer resp.Body.Close()
	// Consider only HTTP 200 and 401 to be valid responses
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusUnauthorized {
		return nil, errors.Errorf(
			"GET %s returned an HTTP %d status code; this address may not "+
				"be a valid v2 Registry endpoint",
			apiAddress,
			resp.StatusCode,
		)
	}
	challengeManager := challenge.NewSimpleManager()
	err = challengeManager.AddResponse(resp)
	return challengeManager,
		errors.Wrap(err, "error configuring challenge manager")
}

// getTagNames retrieves a list of all tag names from the repository.
func (r *repositoryClient) getTagNames(ctx context.Context) ([]string, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Trace("retrieving tags for image")
	tagSvc := r.repo.Tags(ctx)
	tTags, err := tagSvc.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving tags from repository")
	}
	return tTags, nil
}

// getTagByName retrieves a tag by name. This function uses no cache since tags
// can be mutable.
func (r *repositoryClient) getTagByName(
	ctx context.Context,
	tagName string,
	platform *platformConstraint,
) (*Tag, error) {
	manifest, err := r.getManifestByTagNameFn(ctx, tagName)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error retrieving manifest %s", tagName)
	}
	tag, err := r.extractTagFromManifestFn(ctx, manifest, platform)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error extracting tag from manifest %q",
			tagName,
		)
	}
	if tag != nil {
		tag.Name = tagName
	}
	return tag, nil
}

// getTagByDigest retrieves a tag for a given digest. This function uses a cache
// since information retrieved by digest will never change.
func (r *repositoryClient) getTagByDigest(
	ctx context.Context,
	d digest.Digest,
	platform *platformConstraint,
) (*Tag, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("retrieving tag for manifest %s", d)

	if entry, exists := r.registry.tagCache.Get(d.String()); exists {
		tag := entry.(Tag) // nolint: forcetypeassert
		return &tag, nil
	}

	logger.Tracef("tag for manifest %s NOT found in cache", d)

	manifest, err := r.getManifestByDigestFn(ctx, d)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error retrieving manifest %s", d)
	}
	tag, err := r.extractTagFromManifestFn(ctx, manifest, platform)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error extracting tag from manifest %s",
			d,
		)
	}

	if tag != nil {
		// Cache the tag
		r.registry.tagCache.Set(d.String(), *tag, cache.DefaultExpiration)
		logger.Tracef("cached tag for manifest %s", d)
	}

	return tag, nil
}

// getManifestByTag retrieves a manifest for a given tag.
func (r *repositoryClient) getManifestByTagName(
	ctx context.Context,
	tagName string,
) (distribution.Manifest, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("retrieving manifest for tag %q from repository", tagName)
	manifestSvc, err := r.repo.Manifests(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting manifest service")
	}
	manifest, err := manifestSvc.Get(
		ctx,
		digest.FromString(tagName),
		distribution.WithTag(tagName),
		distribution.WithManifestMediaTypes(knownMediaTypes),
	)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error retrieving manifest for tag %q", tagName)
	}
	return manifest, nil
}

// getManifestByDigest retrieves a manifest for a given digest.
func (r *repositoryClient) getManifestByDigest(
	ctx context.Context,
	d digest.Digest,
) (distribution.Manifest, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("retrieving manifest for digest %q from repository", d.String())
	manifestSvc, err := r.repo.Manifests(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting manifest service")
	}
	manifest, err := manifestSvc.Get(
		ctx,
		d,
		distribution.WithManifestMediaTypes(knownMediaTypes),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving manifest for digest %q", d)
	}
	return manifest, nil
}

// extractTagFromManifest extracts tag information from a given manifest. V1
// (legacy), V2, and OCI manifests are supported as well as manifest lists and
// indices (e.g. for multi-arch images).
func (r *repositoryClient) extractTagFromManifest(
	ctx context.Context,
	manifest distribution.Manifest,
	platform *platformConstraint,
) (*Tag, error) {
	switch m := manifest.(type) {
	case *schema1.SignedManifest: //nolint: staticcheck
		return r.extractTagFromV1ManifestFn(m, platform)
	case *schema2.DeserializedManifest:
		return r.extractTagFromV2ManifestFn(ctx, m, platform)
	case *ocischema.DeserializedManifest:
		return r.extractTagFromOCIManifestFn(ctx, m, platform)
	case *manifestlist.DeserializedManifestList, *ocischema.DeserializedImageIndex:
		return r.extractTagFromCollectionFn(ctx, manifest, platform)
	default:
		return nil, errors.Errorf("invalid manifest type %T", manifest)
	}
}

// manifestInfo is a struct used for unmarshaling manifest information.
type manifestInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"architecture"`
	Variant string `json:"variant"`
	Created string `json:"created"`
}

// extractTagFromV1Manifest extracts tag information from a given V1 manifest.
// It is valid for this function to return nil if the manifest does not match
// the specified platform, if any.
func (r *repositoryClient) extractTagFromV1Manifest(
	manifest *schema1.SignedManifest, // nolint: staticcheck
	platform *platformConstraint,
) (*Tag, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload() // nolint: staticcheck
	if err != nil {
		return nil, errors.Wrap(err, "error extracting payload from V1 manifest")
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(context.Background())
	logger.Tracef("extracting tag from V1 manifest %s", digest)

	if len(manifest.History) == 0 {
		return nil,
			errors.Errorf("no history information found in V1 manifest %s", digest)
	}

	var info manifestInfo
	if err = json.Unmarshal(
		[]byte(manifest.History[0].V1Compatibility),
		&info,
	); err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling V1 manifest %s", digest)
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error parsing createdAt timestamp from V1 manifest %s",
			digest,
		)
	}

	return &Tag{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractTagFromV2Manifest extracts tag information from a given V2 manifest.
// It is valid for this function to return nil if the manifest does not match
// the specified platform, if any.
func (r *repositoryClient) extractTagFromV2Manifest(
	ctx context.Context,
	manifest *schema2.DeserializedManifest,
	platform *platformConstraint,
) (*Tag, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload()
	if err != nil {
		return nil,
			errors.Wrap(err, "error extracting payload from V2 manifest")
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("extracting tag from V2 manifest %s", digest)

	// This referenced config object has platform information and creation
	// timestamp
	blob, err := r.getBlobFn(ctx, manifest.Config.Digest)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error fetching blob %s referenced by V2 manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}
	var info manifestInfo
	if err = json.Unmarshal(blob, &info); err != nil {
		return nil, errors.Wrapf(
			err,
			"error unmarshaling blob %s referenced by V2 manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error parsing createdAt timestamp from blob %s referenced by V2 "+
				"manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}

	return &Tag{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractTagFromOCIManifest extracts tag information from a given OCI manifest.
// It is valid for this function to return nil if the manifest does not match
// the specified platform, if any.
func (r *repositoryClient) extractTagFromOCIManifest(
	ctx context.Context,
	manifest *ocischema.DeserializedManifest,
	platform *platformConstraint,
) (*Tag, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload()
	if err != nil {
		return nil, errors.Wrap(err, "error extracting payload from OCI manifest")
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("extracting tag from OCI manifest %s", digest)

	// This referenced config object has platform information and creation
	// timestamp
	blob, err := r.getBlobFn(ctx, manifest.Config.Digest)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error fetching blob %s referenced by OCI manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}
	var info manifestInfo
	if err = json.Unmarshal(blob, &info); err != nil {
		return nil, errors.Wrapf(
			err,
			"error unmarshaling blob %s referenced by OCI manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error parsing createdAt timestamp from blob %s referenced by OCI "+
				"manifest %s",
			manifest.Config.Digest,
			digest,
		)
	}

	return &Tag{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractTagFromCollection extracts tag information from a V2 manifest list or
// OCI index. It is valid for this function to return nil if no manifest in the
// list or index matches the specified platform, if any. This function assumes
// it is only ever invoked with a manifest list or index.
func (r *repositoryClient) extractTagFromCollection(
	ctx context.Context,
	collection distribution.Manifest,
	platform *platformConstraint,
) (*Tag, error) {
	// We need this to calculate the digest. Note that this is the digest of the
	// list or index.
	_, manifestBytes, err := collection.Payload()
	if err != nil {
		return nil, errors.Wrap(err, "error getting collection payload")
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.Tracef(
		"extracting tag from V2 manifest list or OCI index %s",
		digest,
	)

	refs := collection.References()
	if len(refs) == 0 {
		return nil, errors.Errorf(
			"empty V2 manifest list or OCI index %s is not supported",
			digest,
		)
	}

	// If there's a platform constraint, find the ref that matches it and
	// that's the information we're really after.
	if platform != nil {
		var matchedRefs []distribution.Descriptor
		// Filter out references that don't match the platform
		for _, ref := range refs {
			if platform != nil && !platform.matches(
				ref.Platform.OS,
				ref.Platform.Architecture,
				ref.Platform.Variant,
			) {
				continue
			}
			matchedRefs = append(matchedRefs, ref)
		}
		if len(matchedRefs) == 0 {
			// No refs matched the platform
			return nil, nil
		}
		if len(matchedRefs) > 1 {
			// This really shouldn't happen.
			return nil, errors.Errorf(
				"expected only one reference to match platform %q, but found %d",
				platform.String(),
				len(matchedRefs),
			)
		}
		ref := matchedRefs[0]
		tag, err := r.getTagByDigestFn(ctx, ref.Digest, platform)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting tag from manifest %s",
				ref.Digest,
			)
		}
		if tag == nil {
			// This really shouldn't happen.
			return nil, errors.Errorf(
				"expected manifest for digest %v to match platform %q, but it did not",
				ref.Digest,
				platform.String(),
			)
		}
		tag.Digest = digest
		return tag, nil
	}

	// If we get to here there was no platform constraint.

	// Manifest lists and indices don't have a createdAt timestamp, and we had no
	// platform constraint, so we'll follow ALL the references to find the most
	// recently pushed manifest's createdAt timestamp.
	var createdAt *time.Time
	for _, ref := range refs {
		tag, err := r.getTagByDigestFn(ctx, ref.Digest, platform)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting tag from manifest %s",
				ref.Digest,
			)
		}
		if tag == nil {
			// This really shouldn't happen.
			return nil, errors.Errorf(
				"found no tag for manifest %s",
				ref.Digest,
			)
		}
		if createdAt == nil || tag.CreatedAt.After(*createdAt) {
			createdAt = tag.CreatedAt
		}
	}

	return &Tag{
		Digest:    digest,
		CreatedAt: createdAt,
	}, nil
}

// getBlob retrieves a blob from the repository.
func (r *repositoryClient) getBlob(
	ctx context.Context,
	digest digest.Digest,
) ([]byte, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Tracef("retrieving blob for digest %q", digest.String())
	return r.repo.Blobs(ctx).Get(ctx, digest)
}

// rateLimitedRoundTripper is a rate limited implementation of
// http.RoundTripper.
type rateLimitedRoundTripper struct {
	limiter              ratelimit.Limiter
	internalRoundTripper http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (r *rateLimitedRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	r.limiter.Take()
	return r.internalRoundTripper.RoundTrip(req)
}
