package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/credentials"
)

// parseOCIReference parses an OCI image or chart reference and determines the
// credential type. If the reference starts with "oci://", it is treated as a
// Helm OCI repository and the credential type is set to TypeHelm.
func parseOCIReference(imageRef string) (name.Reference, credentials.Type, error) {
	credType := credentials.TypeImage

	// To support Helm OCI repositories, we check if the image reference
	// starts with "oci://". If it does, we treat it as a Helm repository
	// and set the credential type accordingly.
	if trimmed, ok := strings.CutPrefix(imageRef, "oci://"); ok {
		imageRef = trimmed
		credType = credentials.TypeHelm
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image reference %q: %w", imageRef, err)
	}

	return ref, credType, nil
}

// buildOCIRemoteOptions constructs the remote options for interacting with an
// OCI registry, including context, HTTP transport, and authentication.
func buildOCIRemoteOptions(
	ctx context.Context,
	credsDB credentials.Database,
	project string,
	ref name.Reference,
	credType credentials.Type,
	insecureSkipTLSVerify bool,
) ([]remote.Option, error) {
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(ociHTTPTransport(insecureSkipTLSVerify)),
	}

	if authOpt, err := ociAuthOption(ctx, credsDB, project, ref, credType); err != nil {
		return nil, err
	} else if authOpt != nil {
		remoteOpts = append(remoteOpts, authOpt)
	}

	return remoteOpts, nil
}

// ociHTTPTransport creates a new HTTP transport with TLS settings based on the
// insecureSkipTLSVerify flag.
func ociHTTPTransport(insecureSkipTLSVerify bool) *http.Transport {
	httpTransport := cleanhttp.DefaultTransport()
	if insecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	return httpTransport
}

// ociAuthOption retrieves and configures authentication for an OCI registry.
func ociAuthOption(
	ctx context.Context,
	credsDB credentials.Database,
	project string,
	ref name.Reference,
	credType credentials.Type,
) (remote.Option, error) {
	repoURL := ref.Context().String()

	// NB: Some credential database implementations expect the URL to be
	// prefixed with "oci://".
	if credType == credentials.TypeHelm {
		repoURL = "oci://" + repoURL
	}

	creds, err := credsDB.Get(ctx, project, credType, repoURL)
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
