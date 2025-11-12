package builtin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	intio "github.com/akuity/kargo/pkg/io"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindOCIDownload = "oci-download"

	maxOCIArtifactSize = 100 << 20
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindOCIDownload,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newOCIDownloader,
		},
	)
}

// ociDownloader is an implementation of the promotion.StepRunner interface
// that downloads OCI artifacts from a registry.
type ociDownloader struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newOCIDownloader returns an implementation of the promotion.StepRunner
// interface that downloads OCI artifacts from a registry. It uses the provided
// credentials database to authenticate with the registry.
func newOCIDownloader(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &ociDownloader{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindOCIDownload),
	}
}

// Run implements the promotion.StepRunner interface.
func (d *ociDownloader) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := d.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return d.run(ctx, stepCtx, cfg)
}

// convert validates the ociDownloader configuration against a JSON schema
// and converts it into a builtin.OCIDownloadConfig struct.
func (d *ociDownloader) convert(cfg promotion.Config) (builtin.OCIDownloadConfig, error) {
	return validateAndConvert[builtin.OCIDownloadConfig](d.schemaLoader, cfg, stepKindOCIDownload)
}

// run executes the ociDownloader step with the provided configuration.
func (d *ociDownloader) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIDownloadConfig,
) (promotion.StepResult, error) {
	absOutPath, err := d.prepareOutputPath(stepCtx.WorkDir, cfg.OutPath, cfg.AllowOverwrite)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	img, err := d.resolveImage(ctx, stepCtx, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	if err = d.extractLayerToFile(img, cfg.MediaType, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// prepareOutputPath validates and prepares the output path for the artifact.
func (d *ociDownloader) prepareOutputPath(workDir, outPath string, allowOverwrite bool) (string, error) {
	absOutPath, err := securejoin.SecureJoin(workDir, outPath)
	if err != nil {
		return "", fmt.Errorf("failed to join path %q: %w", outPath, err)
	}

	if err = d.checkFileOverwrite(absOutPath, outPath, allowOverwrite); err != nil {
		return "", err
	}

	destDir := filepath.Dir(absOutPath)
	if err = os.MkdirAll(destDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	return absOutPath, nil
}

// checkFileOverwrite validates file overwrite conditions.
func (d *ociDownloader) checkFileOverwrite(absOutPath, outPath string, allowOverwrite bool) error {
	if !allowOverwrite {
		if _, err := os.Stat(absOutPath); err == nil || !os.IsNotExist(err) {
			if err != nil {
				return fmt.Errorf("error checking destination file: %w", err)
			}
			return &promotion.TerminalError{
				Err: fmt.Errorf("file already exists at %s and overwrite is not allowed", outPath),
			}
		}
	}
	return nil
}

// resolveImage resolves the OCI image/artifact from the registry.
func (d *ociDownloader) resolveImage(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIDownloadConfig,
) (v1.Image, error) {
	ref, credType, err := d.parseImageReference(cfg.ImageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference %q: %w", cfg.ImageRef, err)
	}

	remoteOpts, err := d.buildRemoteOptions(ctx, stepCtx, cfg, ref, credType)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve image %q: %w", cfg.ImageRef, err)
	}

	return img, nil
}

// parseImageReference parses the image reference and determines credential type.
func (d *ociDownloader) parseImageReference(imageRef string) (name.Reference, credentials.Type, error) {
	credType := credentials.TypeImage

	// To support Helm OCI repositories, we check if the image reference
	// starts with "oci://". If it does, we treat it as a Helm repository
	// and set the credential type accordingly.
	if strings.HasPrefix(imageRef, "oci://") {
		// Remove the "oci://" prefix if present, as the parser expects a
		// standard image reference format.
		imageRef = strings.TrimPrefix(imageRef, "oci://")
		credType = credentials.TypeHelm
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image reference %q: %w", imageRef, err)
	}

	return ref, credType, nil
}

// buildRemoteOptions constructs the remote options for the registry.
func (d *ociDownloader) buildRemoteOptions(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIDownloadConfig,
	ref name.Reference,
	credType credentials.Type,
) ([]remote.Option, error) {
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(d.buildHTTPTransport(cfg)),
	}

	// Configure authentication
	if authOpt, err := d.getAuthOption(ctx, stepCtx, ref, credType); err != nil {
		return nil, err
	} else if authOpt != nil {
		remoteOpts = append(remoteOpts, authOpt)
	}

	return remoteOpts, nil
}

// buildHTTPTransport creates a new HTTP transport with TLS settings based on
// the configuration.
func (d *ociDownloader) buildHTTPTransport(cfg builtin.OCIDownloadConfig) *http.Transport {
	httpTransport := cleanhttp.DefaultTransport()
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	return httpTransport
}

// getAuthOption retrieves and configures authentication for the registry.
func (d *ociDownloader) getAuthOption(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	ref name.Reference,
	credType credentials.Type,
) (remote.Option, error) {
	repoURL := ref.Context().String()

	// NB: Some credential database implementations expect the URL to be
	// prefixed with "oci://".
	if credType == credentials.TypeHelm {
		repoURL = "oci://" + repoURL
	}

	creds, err := d.credsDB.Get(ctx, stepCtx.Project, credType, repoURL)
	if err != nil {
		return nil, fmt.Errorf("error obtaining credentials for image repo %q: %w", repoURL, err)
	}

	if creds != nil && (creds.Username != "" || creds.Password != "") {
		return remote.WithAuth(&authn.Basic{
			Username: creds.Username,
			Password: creds.Password,
		}), nil
	}

	return nil, nil
}

// extractLayerToFile extracts the target layer from the image to the specified
// file.
func (d *ociDownloader) extractLayerToFile(img v1.Image, mediaType, absOutPath string) error {
	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	targetLayer, err := d.findTargetLayer(img, manifest, mediaType)
	if err != nil {
		return fmt.Errorf("failed to find target layer: %w", err)
	}

	return d.writeLayerToFile(targetLayer, absOutPath)
}

// findTargetLayer finds the appropriate layer based on media type preference.
func (d *ociDownloader) findTargetLayer(
	img v1.Image,
	manifest *v1.Manifest,
	targetMediaType string,
) (v1.Layer, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to get image layers: %w", err)
	}

	if len(layers) == 0 {
		return nil, errors.New("image has no layers")
	}

	// If a specific media type is requested, find the first matching layer
	if targetMediaType != "" {
		for i, layerDesc := range manifest.Layers {
			if string(layerDesc.MediaType) == targetMediaType {
				if i >= len(layers) {
					return nil, fmt.Errorf("layer index %d out of range", i)
				}
				return layers[i], nil
			}
		}
		return nil, fmt.Errorf("no layer found with media type %q", targetMediaType)
	}

	// If no specific media type requested, return the first layer
	return layers[0], nil
}

// writeLayerToFile writes the layer content to a file using atomic operations.
func (d *ociDownloader) writeLayerToFile(layer v1.Layer, absOutPath string) error {
	tempFile, tempPath, err := d.createTempFile(absOutPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err = d.copyLayerToFile(layer, tempFile); err != nil {
		return err
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = fs.SimpleAtomicMove(tempPath, absOutPath); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the same directory as the target.
func (d *ociDownloader) createTempFile(absOutPath string) (*os.File, string, error) {
	destDir := filepath.Dir(absOutPath)
	baseFile := filepath.Base(absOutPath)

	tempFile, err := os.CreateTemp(destDir, baseFile+".tmp")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	if err = tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, "", fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	return tempFile, tempFile.Name(), nil
}

// copyLayerToFile copies layer content to the file with size limits.
func (d *ociDownloader) copyLayerToFile(layer v1.Layer, tempFile *os.File) error {
	size, err := layer.Size()
	if err != nil {
		return fmt.Errorf("failed to get layer size: %w", err)
	}
	if size > maxOCIArtifactSize {
		return &promotion.TerminalError{
			Err: fmt.Errorf("layer size %d exceeds maximum allowed size of %d bytes", size, maxOCIArtifactSize),
		}
	}

	layerReader, err := layer.Compressed()
	if err != nil {
		return fmt.Errorf("failed to get layer content: %w", err)
	}

	if _, err = intio.LimitCopy(tempFile, layerReader, maxOCIArtifactSize); err != nil {
		if errors.Is(err, &intio.BodyTooLargeError{}) {
			return &promotion.TerminalError{
				Err: fmt.Errorf("failed to copy layer content: %w", err),
			}
		}
		return fmt.Errorf("failed to copy layer content: %w", err)
	}

	return nil
}
