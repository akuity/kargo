package acr

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

const (
	// projectResourcePrefix precedes a Kargo Project's name in the name of the
	// Azure managed identity Kargo acts as on that Project's behalf.
	projectResourcePrefix = "kargo-project-"
)

// errNoProjectIdentity indicates that no usable Azure identity exists for a
// Project. It signals that the caller should fall back to the controller's own
// identity rather than treat the condition as a failure.
var errNoProjectIdentity = errors.New("no Project-specific Azure identity")

// projectIdentityName returns the name of the Azure managed identity Kargo
// acts as on behalf of the given Project.
func projectIdentityName(project string) string {
	return projectResourcePrefix + project
}

// userAssignedIdentityGetter is the subset of the Azure Resource Manager
// managed identity API that this package uses.
type userAssignedIdentityGetter interface {
	Get(
		ctx context.Context,
		resourceGroupName string,
		resourceName string,
		options *armmsi.UserAssignedIdentitiesClientGetOptions,
	) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
}

// resolveClientID resolves a Kargo Project to the client ID of the Azure
// managed identity Kargo acts as on that Project's behalf. Azure addresses
// identities by GUID, and a GUID cannot be derived from a Project's name the
// way an AWS role ARN or a GCP service account email can, so a lookup stands in
// for the naming convention those providers rely on.
//
// Nothing is retained between calls. A resolved client ID would be correct
// until the identity behind it is deleted and recreated, after which a retained
// one would send every later exchange for that Project to an identity that no
// longer exists, silently falling back to the controller's own identity until
// the process restarts. Resolving each time costs one request on a path that
// already makes several and runs only when a token is actually needed.

func (p *WorkloadIdentityProvider) resolveClientID(
	ctx context.Context,
	project string,
) (string, error) {
	res, err := p.identities.Get(
		ctx,
		p.resourceGroup,
		projectIdentityName(project),
		nil,
	)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) &&
			respErr.StatusCode == http.StatusNotFound {
			return "", errNoProjectIdentity
		}
		return "", fmt.Errorf(
			"error getting managed identity %q in resource group %q: %w",
			projectIdentityName(project), p.resourceGroup, err,
		)
	}
	if res.Properties == nil || res.Properties.ClientID == nil {
		return "", errNoProjectIdentity
	}
	return *res.Properties.ClientID, nil
}
