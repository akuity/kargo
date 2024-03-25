package image

import (
	"context"
	"crypto/tls"
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
	"github.com/hashicorp/go-cleanhttp"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
	"golang.org/x/sync/semaphore"

	"github.com/akuity/kargo/internal/logging"
)

const (
	// maxMetadataConcurrency is the maximum number of concurrent goroutines that
	// can be used to fetch metadata. Per Go's documentation, goroutines are very
	// cheap (practical to spawn tens of thousands), AND we have a rate limiter in
	// place for each registry, so there's no reason for this number not to be
	// pretty large.
	maxMetadataConcurrency = 1000

	unknown = "unknown"
)

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

// repositoryClient is a client for retrieving information from a specific image
// container repository.
type repositoryClient struct {
	registry *registry
	image    string
	repo     distribution.Repository

	// The following behaviors are overridable for testing purposes:

	getImageByTagFn func(
		context.Context,
		string,
		*platformConstraint,
	) (*Image, error)

	getImageByDigestFn func(
		context.Context,
		digest.Digest,
		*platformConstraint,
	) (*Image, error)

	getManifestByTagFn func(
		context.Context,
		string,
	) (distribution.Manifest, error)

	getManifestByDigestFn func(
		context.Context,
		digest.Digest,
	) (distribution.Manifest, error)

	extractImageFromManifestFn func(
		context.Context,
		distribution.Manifest,
		*platformConstraint,
	) (*Image, error)

	extractImageFromV1ManifestFn func(
		*schema1.SignedManifest, // nolint: staticcheck
		*platformConstraint,
	) (*Image, error)

	extractImageFromV2ManifestFn func(
		context.Context,
		*schema2.DeserializedManifest,
		*platformConstraint,
	) (*Image, error)

	extractImageFromOCIManifestFn func(
		context.Context,
		*ocischema.DeserializedManifest,
		*platformConstraint,
	) (*Image, error)

	extractImageFromCollectionFn func(
		context.Context,
		distribution.Manifest,
		*platformConstraint,
	) (*Image, error)

	getBlobFn func(context.Context, digest.Digest) ([]byte, error)
}

// newRepositoryClient parses the provided repository URL to infer registry
// information and image name. This information is used to initialize and
// return a new repository client.
func newRepositoryClient(
	repoURL string,
	insecureSkipTLSVerify bool,
	creds *Credentials,
) (*repositoryClient, error) {
	repoRef, err := reference.ParseNormalizedNamed(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing image repo URL %s: %w", repoURL, err)
	}
	registryURL := reference.Domain(repoRef)
	reg := getRegistry(registryURL)
	image := reg.normalizeImageName(reference.Path(repoRef))
	apiAddress := strings.TrimSuffix(reg.apiAddress, "/")

	httpTransport := cleanhttp.DefaultTransport()
	if insecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: insecureSkipTLSVerify, // nolint: gosec
		}
	}

	challengeManager, err := getChallengeManager(
		apiAddress,
		&rateLimitedRoundTripper{
			limiter:              reg.rateLimiter,
			internalRoundTripper: httpTransport,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting challenge manager for %s: %w", apiAddress, err)
	}

	if creds == nil {
		creds = &Credentials{}
	}

	rlt := &rateLimitedRoundTripper{
		limiter: reg.rateLimiter,
		internalRoundTripper: transport.NewTransport(
			httpTransport,
			auth.NewAuthorizer(
				challengeManager,
				auth.NewTokenHandler(
					httpTransport,
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
		return nil, fmt.Errorf("error getting reference for image %q: %w", image, err)
	}
	repo, err := client.NewRepository(imageRef, apiAddress, rlt)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating internal repository for image %q in registry %s: %w",
			image,
			apiAddress,
			err,
		)
	}

	r := &repositoryClient{
		registry: reg,
		image:    image,
		repo:     repo,
	}

	r.getImageByTagFn = r.getImageByTag
	r.getImageByDigestFn = r.getImageByDigest
	r.getManifestByTagFn = r.getManifestByTag
	r.getManifestByDigestFn = r.getManifestByDigest
	r.extractImageFromManifestFn = r.extractImageFromManifest
	r.extractImageFromV1ManifestFn = r.extractImageFromV1Manifest
	r.extractImageFromV2ManifestFn = r.extractImageFromV2Manifest
	r.extractImageFromOCIManifestFn = r.extractImageFromOCIManifest
	r.extractImageFromCollectionFn = r.extractImageFromCollection
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
		return nil, fmt.Errorf("error requesting %s: %w", apiAddress, err)
	}
	defer resp.Body.Close()
	// Consider only HTTP 200 and 401 to be valid responses
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf(
			"GET %s returned an HTTP %d status code; this address may not "+
				"be a valid v2 Registry endpoint",
			apiAddress,
			resp.StatusCode,
		)
	}
	challengeManager := challenge.NewSimpleManager()
	if err = challengeManager.AddResponse(resp); err != nil {
		err = fmt.Errorf("error configuring challenge manager: %w", err)
	}
	return challengeManager, err
}

// getTags retrieves a list of all tags from the repository.
func (r *repositoryClient) getTags(ctx context.Context) ([]string, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("retrieving tags for image")
	tagSvc := r.repo.Tags(ctx)
	tags, err := tagSvc.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving tags from repository: %w", err)
	}
	return tags, nil
}

// getImageByTag retrieves an Image by tag. This function uses no cache since
// tags can be mutable.
func (r *repositoryClient) getImageByTag(
	ctx context.Context,
	tag string,
	platform *platformConstraint,
) (*Image, error) {
	manifest, err := r.getManifestByTagFn(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("error retrieving manifest for tag %s: %w", tag, err)
	}
	image, err := r.extractImageFromManifestFn(ctx, manifest, platform)
	if err != nil {
		return nil, fmt.Errorf("error extracting image from manifest for tag %q: %w", tag, err)
	}
	if image != nil {
		image.Tag = tag
	}
	return image, nil
}

// getImageByDigest retrieves an Image for a given digest. This function uses a
// cache since information retrieved by digest will never change.
func (r *repositoryClient) getImageByDigest(
	ctx context.Context,
	d digest.Digest,
	platform *platformConstraint,
) (*Image, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("retrieving image for manifest", "digest", d)

	if entry, exists := r.registry.imageCache.Get(d.String()); exists {
		image := entry.(Image) // nolint: forcetypeassert
		return &image, nil
	}

	logger.V(2).Info("image for manifest NOT found in cache", "digest", d)

	manifest, err := r.getManifestByDigestFn(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("error retrieving manifest %s: %w", d, err)
	}
	image, err := r.extractImageFromManifestFn(ctx, manifest, platform)
	if err != nil {
		return nil, fmt.Errorf("error extracting image from manifest %s: %w", d, err)
	}

	if image != nil {
		// Cache the image
		r.registry.imageCache.Set(d.String(), *image, cache.DefaultExpiration)
		logger.V(2).Info("cached image for manifest", "digest", d)
	}

	return image, nil
}

// getManifestByTag retrieves a manifest for a given tag.
func (r *repositoryClient) getManifestByTag(
	ctx context.Context,
	tag string,
) (distribution.Manifest, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("retrieving manifest from repository", "tag", tag)
	manifestSvc, err := r.repo.Manifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting manifest service: %w", err)
	}
	manifest, err := manifestSvc.Get(
		ctx,
		digest.FromString(tag),
		distribution.WithTag(tag),
		distribution.WithManifestMediaTypes(knownMediaTypes),
	)
	if err != nil {
		return nil, fmt.Errorf("error retrieving manifest for tag %q: %w", tag, err)
	}
	return manifest, nil
}

// getManifestByDigest retrieves a manifest for a given digest.
func (r *repositoryClient) getManifestByDigest(
	ctx context.Context,
	d digest.Digest,
) (distribution.Manifest, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("retrieving manifest from repository", "digest", d.String())
	manifestSvc, err := r.repo.Manifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting manifest service: %w", err)
	}
	manifest, err := manifestSvc.Get(
		ctx,
		d,
		distribution.WithManifestMediaTypes(knownMediaTypes),
	)
	if err != nil {
		return nil, fmt.Errorf("error retrieving manifest for digest %q: %w", d, err)
	}
	return manifest, nil
}

// extractImageFromManifest extracts an Image from a given manifest. V1
// (legacy), V2, and OCI manifests are supported as well as manifest lists and
// indices (e.g. for multi-arch images).
func (r *repositoryClient) extractImageFromManifest(
	ctx context.Context,
	manifest distribution.Manifest,
	platform *platformConstraint,
) (*Image, error) {
	switch m := manifest.(type) {
	case *schema1.SignedManifest: //nolint: staticcheck
		return r.extractImageFromV1ManifestFn(m, platform)
	case *schema2.DeserializedManifest:
		return r.extractImageFromV2ManifestFn(ctx, m, platform)
	case *ocischema.DeserializedManifest:
		return r.extractImageFromOCIManifestFn(ctx, m, platform)
	case *manifestlist.DeserializedManifestList, *ocischema.DeserializedImageIndex:
		return r.extractImageFromCollectionFn(ctx, manifest, platform)
	default:
		return nil, fmt.Errorf("invalid manifest type %T", manifest)
	}
}

// manifestInfo is a struct used for unmarshaling manifest information.
type manifestInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"architecture"`
	Variant string `json:"variant"`
	Created string `json:"created"`
}

// extractImageFromV1Manifest extracts an Image from a given V1 manifest. It is
// valid for this function to return nil if the manifest does not match the
// specified platform, if any.
func (r *repositoryClient) extractImageFromV1Manifest(
	manifest *schema1.SignedManifest, // nolint: staticcheck
	platform *platformConstraint,
) (*Image, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload() // nolint: staticcheck
	if err != nil {
		return nil, fmt.Errorf("error extracting payload from V1 manifest: %w", err)
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(context.Background())
	logger.V(2).Info("extracting image from V1 manifest", "digest", digest)

	if len(manifest.History) == 0 {
		return nil, fmt.Errorf("no history information found in V1 manifest %s", digest)
	}

	var info manifestInfo
	if err = json.Unmarshal(
		[]byte(manifest.History[0].V1Compatibility),
		&info,
	); err != nil {
		return nil, fmt.Errorf("error unmarshaling V1 manifest %s: %w", digest, err)
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing createdAt timestamp from V1 manifest %s: %w",
			digest,
			err,
		)
	}

	return &Image{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractImageFromV2Manifest extracts an Image from a given V2 manifest. It is
// valid for this function to return nil if the manifest does not match the
// specified platform, if any.
func (r *repositoryClient) extractImageFromV2Manifest(
	ctx context.Context,
	manifest *schema2.DeserializedManifest,
	platform *platformConstraint,
) (*Image, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload()
	if err != nil {
		return nil, fmt.Errorf("error extracting payload from V2 manifest: %w", err)
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("extracting image from V2 manifest", "digest", digest)

	// This referenced config object has platform information and creation
	// timestamp
	blob, err := r.getBlobFn(ctx, manifest.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf(
			"error fetching blob %s referenced by V2 manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}
	var info manifestInfo
	if err = json.Unmarshal(blob, &info); err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling blob %s referenced by V2 manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing createdAt timestamp from blob %s referenced by V2 manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}

	return &Image{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractImageFromOCIManifest extracts an Image from a given OCI manifest. It
// is valid for this function to return nil if the manifest does not match the
// specified platform, if any.
func (r *repositoryClient) extractImageFromOCIManifest(
	ctx context.Context,
	manifest *ocischema.DeserializedManifest,
	platform *platformConstraint,
) (*Image, error) {
	// We need this to calculate the digest
	_, manifestBytes, err := manifest.Payload()
	if err != nil {
		return nil, fmt.Errorf("error extracting payload from OCI manifest: %w", err)
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info("extracting image from OCI manifest", "digest", digest)

	// This referenced config object has platform information and creation
	// timestamp
	blob, err := r.getBlobFn(ctx, manifest.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf(
			"error fetching blob %s referenced by OCI manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}
	var info manifestInfo
	if err = json.Unmarshal(blob, &info); err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling blob %s referenced by OCI manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}

	if info.OS == unknown || info.OS == "" || info.Arch == unknown || info.Arch == "" {
		// This doesn't look like an image. It might be an attestation or something
		// else. It's definitely not what we're looking for.
		return nil, nil
	}

	if platform != nil &&
		!platform.matches(info.OS, info.Arch, info.Variant) {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, info.Created)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing createdAt timestamp from blob %s referenced by OCI manifest %s: %w",
			manifest.Config.Digest,
			digest,
			err,
		)
	}

	return &Image{
		Digest:    digest,
		CreatedAt: &createdAt,
	}, nil
}

// extractImageFromCollection extracts an Image from a V2 manifest list or OCI
// index. It is valid for this function to return nil if no manifest in the list
// or index matches the specified platform, if any. This function assumes it is
// only ever invoked with a manifest list or index.
func (r *repositoryClient) extractImageFromCollection(
	ctx context.Context,
	collection distribution.Manifest,
	platform *platformConstraint,
) (*Image, error) {
	// We need this to calculate the digest. Note that this is the digest of the
	// list or index.
	_, manifestBytes, err := collection.Payload()
	if err != nil {
		return nil, fmt.Errorf("error getting collection payload: %w", err)
	}
	digest := digest.FromBytes(manifestBytes)

	logger := logging.LoggerFromContext(ctx)
	logger.V(2).Info(
		"extracting image from V2 manifest list or OCI index",
		"digest", digest,
	)

	refs := make([]distribution.Descriptor, 0, len(collection.References()))
	for _, ref := range collection.References() {
		if ref.Platform == nil ||
			ref.Platform.OS == unknown || ref.Platform.OS == "" ||
			ref.Platform.Architecture == unknown || ref.Platform.Architecture == "" {
			// This reference doesn't look like a reference to an image. It might
			// be an attestation or something else. Skip it.
			continue
		}
		refs = append(refs, ref)
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf(
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
			return nil, fmt.Errorf(
				"expected only one reference to match platform %q, but found %d",
				platform.String(),
				len(matchedRefs),
			)
		}
		ref := matchedRefs[0]
		image, err := r.getImageByDigestFn(ctx, ref.Digest, platform)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image from manifest %s: %w",
				ref.Digest,
				err,
			)
		}
		if image == nil {
			// This really shouldn't happen.
			return nil, fmt.Errorf(
				"expected manifest for digest %v to match platform %q, but it did not",
				ref.Digest,
				platform.String(),
			)
		}
		image.Digest = digest
		return image, nil
	}

	// If we get to here there was no platform constraint.

	// Manifest lists and indices don't have a createdAt timestamp, and we had no
	// platform constraint, so we'll follow ALL the references to find the most
	// recently pushed manifest's createdAt timestamp.
	var createdAt *time.Time
	for _, ref := range refs {
		image, err := r.getImageByDigestFn(ctx, ref.Digest, platform)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image from manifest %s: %w",
				ref.Digest,
				err,
			)
		}
		if image == nil {
			// This really shouldn't happen.
			return nil, fmt.Errorf(
				"found no image for manifest %s",
				ref.Digest,
			)
		}
		if createdAt == nil || image.CreatedAt.After(*createdAt) {
			createdAt = image.CreatedAt
		}
	}

	return &Image{
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
	logger.V(2).Info("retrieving blob for digest", "digest", digest.String())
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
