package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

var _ freightclient.FreightClient = (*Client)(nil)

func (c *Client) ApproveFreight(
	ctx context.Context,
	project, stageName, nameOrAlias string,
) error {
	resp, err := c.api.CoreAPI.ApproveFreight(ctx, project, nameOrAlias).
		Stage(stageName).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

// GetFreight returns, through the API server, the Freight with the given name,
// or nil if there is no such Freight. Freight is matched explicitly by name.
func (c *Client) GetFreight(
	ctx context.Context,
	project string,
	name string,
) (*kargoapi.Freight, error) {
	return c.getFreightReq(ctx, project, name, true, false)
}

func (c *Client) GetFreightByAlias(
	ctx context.Context,
	project string,
	alias string,
) (*kargoapi.Freight, error) {
	return c.getFreightReq(ctx, project, alias, false, true)
}

func (c *Client) GetFreightByNameOrAlias(
	ctx context.Context,
	project string,
	name string,
) (*kargoapi.Freight, error) {
	return c.getFreightReq(ctx, project, name, false, false)
}

func (c *Client) getFreightReq(
	ctx context.Context,
	project string,
	name string,
	nameOnly bool,
	aliasOnly bool,
) (*kargoapi.Freight, error) {
	model, resp, err := c.api.CoreAPI.GetFreight(ctx, project, name).
		NameOnly(nameOnly).
		AliasOnly(aliasOnly).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Freight %q in Project %q: %w", name, project, ResponseError(err, resp),
		)
	}
	freight := &kargoapi.Freight{}
	if err = convert(model, freight); err != nil {
		return nil, fmt.Errorf(
			"error decoding Freight %q in Project %q: %w", name, project, err,
		)
	}
	return freight, nil
}

// CreateFreight creates, through the API server, the given Freight in the
// Project, optionally recording status metadata on it (applied fill-if-missing).
// The operation is idempotent: creating a piece of Freight that already exists is
// not an error. It returns the created (or pre-existing) Freight and whether it
// was genuinely created (true) rather than already present (false).
func (c *Client) CreateFreight(
	ctx context.Context,
	project string,
	freight *kargoapi.Freight,
	metadata map[string]apiextensionsv1.JSON,
) (*kargoapi.Freight, bool, error) {
	genFreight := &kargogen.Freight{}
	if err := convert(freight, genFreight); err != nil {
		return nil, false, fmt.Errorf("error encoding Freight for Project %q: %w", project, err)
	}
	body := kargogen.CreateFreightRequest{Freight: genFreight}
	if len(metadata) > 0 {
		md := map[string]any{}
		if err := convert(metadata, &md); err != nil {
			return nil, false, fmt.Errorf("error encoding Freight metadata for Project %q: %w", project, err)
		}
		body.Metadata = md
	}
	model, resp, err := c.api.CoreAPI.CreateFreight(ctx, project).Body(body).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, false, fmt.Errorf(
			"error creating Freight in Project %q: %w", project, ResponseError(err, resp),
		)
	}
	created := &kargoapi.Freight{}
	if err = convert(model, created); err != nil {
		return nil, false, fmt.Errorf("error decoding created Freight in Project %q: %w", project, err)
	}
	// The API server answers 201 for a genuinely new piece of Freight and 200 when
	// it already existed.
	isNew := resp != nil && resp.StatusCode == http.StatusCreated
	return created, isNew, nil
}

// DeleteFreight deletes a freight through the apiServer
func (c *Client) DeleteFreight(
	ctx context.Context,
	project string,
	freightNameOrAlias string,
) error {
	resp, err := c.api.CoreAPI.DeleteFreight(ctx, project, freightNameOrAlias).Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf(
			"error deleting Freight %q in Project %q: %w", freightNameOrAlias, project, ResponseError(err, resp),
		)
	}
	return nil
}

// PatchFreightAlias sets, through the API server, the alias of the named
// Freight. The API server rejects an alias already used by another piece of
// Freight in the Project.
func (c *Client) PatchFreightAlias(
	ctx context.Context,
	project string,
	freightNameOrAlias string,
	newAlias string,
) error {
	resp, err := c.api.CoreAPI.PatchFreightAlias(ctx, project, freightNameOrAlias).
		NewAlias(newAlias).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf(
			"error setting alias of Freight %q in Project %q to %q: %w",
			freightNameOrAlias, project, newAlias, ResponseError(err, resp),
		)
	}
	return nil
}

// PatchFreightMetadata upserts, through the API server, keys in the status
// metadata of the named Freight. A value already recorded under a key in
// metadata is overwritten; other keys are left as they are.
func (c *Client) PatchFreightMetadata(
	ctx context.Context,
	project string,
	freightName string,
	metadata map[string]apiextensionsv1.JSON,
) error {
	md := map[string]any{}
	if err := convert(metadata, &md); err != nil {
		return fmt.Errorf("error encoding Freight metadata for Project %q: %w", project, err)
	}
	body := kargogen.PatchFreightMetadataRequest{}
	body.Metadata = md
	resp, err := c.api.CoreAPI.PatchFreightMetadata(ctx, project, freightName).
		Body(body).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf(
			"error updating metadata of Freight %q in Project %q: %w",
			freightName, project, ResponseError(err, resp),
		)
	}
	return nil
}

// ListFreightFromOrigin returns, through the API server, the Freight that
// originated from the named Warehouse, narrowed by opts (nil means all of it).
func (c *Client) ListFreightFromOrigin(
	ctx context.Context,
	project string,
	origin kargoapi.FreightOrigin,
	opts *freightclient.ListFreightFromOriginOptions,
) ([]kargoapi.Freight, error) {
	req := c.api.CoreAPI.ListFreightFromOrigin(ctx, project, string(origin.Kind), origin.Name)
	if opts != nil {
		if opts.ApprovedFor != "" {
			req = req.ApprovedFor(opts.ApprovedFor)
		}
		if len(opts.VerifiedIn) > 0 {
			req = req.VerifiedIn(opts.VerifiedIn)
		}
		if opts.CurrentlyIn != "" {
			req = req.CurrentlyIn(opts.CurrentlyIn)
		}
		if opts.AvailabilityStrategy != "" {
			req = req.AvailabilityStrategy(string(opts.AvailabilityStrategy))
		}
		if opts.RequiredSoakTime != nil {
			req = req.RequiredSoakTime(opts.RequiredSoakTime.Duration.String())
		}
	}
	list, resp, err := req.Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Freight from Origin %v in Project %q: %w",
			origin, project, ResponseError(err, resp),
		)
	}
	freight := &kargoapi.FreightList{}
	if err = convert(list, freight); err != nil {
		return nil, fmt.Errorf(
			"error decoding Freight from Origin %v in Project %q: %w", origin, project, err,
		)
	}
	return freight.Items, nil
}

// MarkFreightsCurrentlyInStage records, through the API server, that the named
// Freights, are currently in use by the Stage.
// Any Freights not listed in this request will be unmarked as currentlyIn.
func (c *Client) MarkFreightsCurrentlyInStage(
	ctx context.Context,
	project, stage string,
	freights []string,
) error {
	body := kargogen.MarkFreightsCurrentlyInStageRequest{
		FreightNames: freights,
	}
	resp, err := c.api.CoreAPI.
		MarkFreightsCurrentlyInStage(
			ctx,
			project,
			stage,
		).
		Body(body).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf(
			"error marking Freights %v as currently in Stage %q in Project %q: %w",
			freights, stage, project, ResponseError(err, resp),
		)
	}
	return nil
}

// MarkFreightsVerifiedInStage records, through the API server, that the named
// Freight has been verified in the Stage at verifiedAt. A zero verifiedAt lets
// the API server use the time it handles the request.
func (c *Client) MarkFreightsVerifiedInStage(
	ctx context.Context,
	project, stage string,
	freights []string,
	verifiedAt time.Time,
) error {
	body := kargogen.MarkFreightsVerifiedInStageRequest{
		FreightNames: freights,
	}
	if !verifiedAt.IsZero() {
		at := verifiedAt.UTC().Format(time.RFC3339)
		body.VerifiedAt = &at
	}
	resp, err := c.api.CoreAPI.
		MarkFreightsVerifiedInStage(ctx, project, stage).
		Body(body).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf(
			"error marking Freights %v as verified in Stage %q in Project %q: %w",
			freights, stage, project, ResponseError(err, resp),
		)
	}
	return nil
}

func (c *Client) ClearStageFreights(
	ctx context.Context,
	project, stage string,
) error {
	resp, err := c.api.CoreAPI.
		ClearStageFreights(ctx, project, stage).
		Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf(
			"error clearing Stage %q from Freights status in Project %q: %w",
			stage, project, ResponseError(err, resp),
		)
	}
	return nil
}
