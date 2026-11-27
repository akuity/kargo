package storage

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient"
	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

var _ freightclient.FreightClient = (*Client)(nil)

var freightGR = kargoapi.GroupVersion.WithResource("freights").GroupResource()

func (storage *Client) CreateFreight(
	ctx context.Context,
	freight *kargoapi.Freight,
	metadata map[string]apiextensionsv1.JSON,
) (*kargoapi.Freight, bool, error) {
	created := true
	if err := storage.client.Create(ctx, freight); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, false, fmt.Errorf("create freight: %w", err)
		}
		created = false
	}

	// Read back the created (or pre-existing) Freight so metadata is applied to its
	// current status and the response reflects the stored resource.
	current := &kargoapi.Freight{}
	key := client.ObjectKey{Namespace: freight.Namespace, Name: freight.Name}
	if err := storage.client.Get(ctx, key, current); err != nil {
		return nil, false, fmt.Errorf("get freight: %w", err)
	}

	if len(metadata) > 0 {
		newStatus := *current.Status.DeepCopy()
		changed := false
		for k, v := range metadata {
			// Fill-if-missing: never overwrite metadata already recorded.
			if _, exists := newStatus.Metadata[k]; exists {
				continue
			}
			if newStatus.Metadata == nil {
				newStatus.Metadata = map[string]apiextensionsv1.JSON{}
			}
			newStatus.Metadata[k] = v
			changed = true
		}
		if changed {
			if err := storage.patchFreightStatus(ctx, storage.client, current, newStatus); err != nil {
				return nil, false, fmt.Errorf("patch freight status: %w", err)
			}
			current.Status = newStatus
		}
	}

	return current, created, nil
}

// DeleteFreight deletes existing freight.
// If freight does not exist, it returns apierrors.NotFound error - caller should handle idempotency, if necessary
func (storage *Client) DeleteFreight(ctx context.Context, project string, nameOrAlias string) error {
	freight, err := storage.GetFreightByNameOrAlias(ctx, project, nameOrAlias)
	if err != nil {
		return err
	}
	if freight == nil {
		return apierrors.NewNotFound(freightGR, nameOrAlias)
	}

	return storage.client.Delete(ctx, freight)
}

func (storage *Client) GetFreight(ctx context.Context, project string, name string) (*kargoapi.Freight, error) {
	freight := &kargoapi.Freight{}
	err := storage.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		freight,
	)
	if err != nil && apierrors.IsNotFound(err) {
		return nil, nil
	}
	return freight, err
}

func (storage *Client) GetFreightByAlias(
	ctx context.Context,
	project string, alias string,
) (*kargoapi.Freight, error) {
	list := &kargoapi.FreightList{}
	if err := storage.client.List(
		ctx,
		list,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.LabelKeyAlias: alias},
	); err != nil {
		return nil, err
	}
	switch len(list.Items) {
	case 0:
		return nil, nil
	case 1:
		return &list.Items[0], nil
	default:
		names := make([]string, len(list.Items))
		for i, freight := range list.Items {
			names[i] = freight.Name
		}
		return nil, apierrors.NewConflict(freightGR, alias, fmt.Errorf(
			"alias %q is shared by multiple pieces of Freight in project %q (%s); "+
				"refer to the Freight by name or give one of them a new alias",
			alias, project, strings.Join(names, ", "),
		))
	}
}

func (storage *Client) GetFreightByNameOrAlias(
	ctx context.Context,
	project, nameOrAlias string,
) (*kargoapi.Freight, error) {
	freight, err := storage.GetFreight(ctx, project, nameOrAlias)
	if err != nil {
		return nil, err
	}
	if freight == nil {
		return storage.GetFreightByAlias(ctx, project, nameOrAlias)
	}
	return freight, nil
}

func (storage *Client) PatchFreightAlias(
	ctx context.Context,
	project string,
	nameOrAlias string, newAlias string,
) error {
	freight, err := storage.GetFreightByNameOrAlias(ctx, project, nameOrAlias)
	if err != nil {
		return err
	}
	if freight == nil {
		return apierrors.NewNotFound(freightGR, nameOrAlias)
	}

	// Make sure this alias isn't already used by some other piece of Freight
	freightList := kargoapi.FreightList{}
	if err := storage.client.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.LabelKeyAlias: newAlias},
	); err != nil {
		return err
	}
	if len(freightList.Items) > 1 ||
		(len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
		return apierrors.NewConflict(freightGR, nameOrAlias, fmt.Errorf(
			"alias %q already used by another piece of Freight in namespace %q",
			newAlias,
			project,
		))
	}

	// Proceed with the update using patch
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{%q:%q}},"alias":%q}`,
			kargoapi.LabelKeyAlias,
			newAlias,
			newAlias,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	return storage.client.Patch(ctx, freight, patch)
}

func (storage *Client) PatchFreightMetadata(
	ctx context.Context,
	project string, freightName string,
	metadata map[string]apiextensionsv1.JSON,
) error {
	freight, err := storage.GetFreight(ctx, project, freightName)
	if err != nil {
		return err
	}
	if freight == nil {
		return apierrors.NewNotFound(freightGR, freightName)
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.Metadata == nil {
		newStatus.Metadata = make(map[string]apiextensionsv1.JSON, len(metadata))
	}
	for k, v := range metadata {
		newStatus.Metadata[k] = v
	}

	if err := storage.patchFreightStatus(ctx, storage.client, freight, newStatus); err != nil {
		return fmt.Errorf("patch freight status: %w", err)
	}
	return nil
}

func (storage *Client) ApproveFreight(ctx context.Context,
	project, stageName, nameOrAlias string) error {
	freight, err := storage.GetFreightByNameOrAlias(ctx, project, nameOrAlias)
	if err != nil {
		return err
	}

	if freight == nil {
		return apierrors.NewNotFound(freightGR, nameOrAlias)
	}

	stage := &kargoapi.Stage{}
	if err := storage.client.Get(
		ctx,
		client.ObjectKey{Name: stageName, Namespace: project},
		stage,
	); err != nil {
		return err
	}

	// Approval bypasses verification and soak requirements, but not origin
	// membership: an approval for a Stage that doesn't request Freight from the
	// Freight's origin could never be acted upon and is certainly a mistake.
	if !stage.RequestsFreightFromOrigin(freight.Origin) {
		return apierrors.NewBadRequest(fmt.Sprintf(
			"Stage %q does not request Freight from origin %q",
			stageName, freight.Origin.String(),
		))
	}

	if freight.IsApprovedFor(stageName) {
		return nil
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.ApprovedFor == nil {
		newStatus.ApprovedFor = make(map[string]kargoapi.ApprovedStage)
	}
	newStatus.AddApprovedStage(stageName, time.Now())

	if err := storage.patchFreightStatus(
		ctx, storage.client, freight, newStatus,
	); err != nil {
		return fmt.Errorf("patch freight status: %w", err)
	}

	var actor string
	eventMsg := fmt.Sprintf("Freight approved for Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		eventMsg += fmt.Sprintf(" by %q", actor)
	}

	if storage.eventSender != nil {
		evt, err := event.NewFreightApproved(eventMsg, actor, stageName, freight)
		if err == nil {
			err = storage.eventSender.Send(ctx, evt)
		}
		if err != nil {
			logging.LoggerFromContext(ctx).Error(err,
				"error sending Freight approved event")
		}
	}
	return nil
}

func (storage *Client) ListFreightFromOrigin(
	ctx context.Context,
	project string, origin kargoapi.FreightOrigin,
	opts *freightclient.ListFreightFromOriginOptions,
) ([]kargoapi.Freight, error) {
	if opts == nil {
		opts = &freightclient.ListFreightFromOriginOptions{}
	}

	// Common selectors we'll use a few times in the code that follows:

	var originSelector fields.Selector
	if origin.Kind != kargoapi.FreightOriginKindWarehouse {
		// TODO: handle other origins when implemented
		return nil, fmt.Errorf("unsupported origin kind %q", origin.Kind)
	}
	originSelector = fields.OneTermEqualSelector("warehouse", origin.Name)

	// Selects Freight currently used by a specific Stage
	var currentlyInSelector fields.Selector
	if opts.CurrentlyIn != "" {
		currentlyInSelector = fields.OneTermEqualSelector("currentlyIn", opts.CurrentlyIn)
	}

	// Build a list of fields.Selectors we'll use to make multiple queries whose
	// results we will merge and de-dupe.
	fieldSelectors := make([]fields.Selector, 0)
	if opts.ApprovedFor == "" && len(opts.VerifiedIn) == 0 {
		// Just list all Freight resources that originated from the Origin
		sels := make([]fields.Selector, 1, 2)
		sels[0] = originSelector
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	}
	if opts.ApprovedFor != "" {
		// List all Freight resources that are approved for the specified Stage
		sels := make([]fields.Selector, 2, 3)
		sels[0] = originSelector
		sels[1] = fields.OneTermEqualSelector("approvedFor", opts.ApprovedFor)
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	}

	// Construct selectors for listing Freight using the configured AvailabilityStrategy
	// semantics.
	switch opts.AvailabilityStrategy {
	case kargoapi.FreightAvailabilityStrategyAll:
		// Query for Freight that is verified in ALL the VerifiedIn stages.
		sels := make([]fields.Selector, 1, len(opts.VerifiedIn)+2)
		sels[0] = originSelector
		for _, stage := range opts.VerifiedIn {
			sels = append(
				sels,
				fields.OneTermEqualSelector("verifiedIn", stage),
			)
		}
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	case kargoapi.FreightAvailabilityStrategyOneOf, "":
		// Query for Freight that is verified in ANY of the VerifiedIn stages.
		for _, stage := range opts.VerifiedIn {
			sels := make([]fields.Selector, 2, 3)
			sels[0] = originSelector
			sels[1] = fields.OneTermEqualSelector("verifiedIn", stage)
			if currentlyInSelector != nil {
				sels = append(sels, currentlyInSelector)
			}
			fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
		}
	default:
		return nil, fmt.Errorf(
			"unsupported AvailabilityStrategy: %s",
			opts.AvailabilityStrategy,
		)
	}

	var freight []kargoapi.Freight
	for _, fs := range fieldSelectors {
		res := &kargoapi.FreightList{}
		if err := storage.client.List(
			ctx,
			res,
			&client.ListOptions{
				Namespace:     project,
				FieldSelector: fs,
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight for Origin %v in namespace %q: %w",
				origin,
				project,
				err,
			)
		}
		freight = append(freight, res.Items...)
	}

	// Sort and de-dupe
	slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	freight = slices.CompactFunc(freight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	if len(opts.VerifiedIn) == 0 || opts.RequiredSoakTime == nil {
		// Nothing left to do
		return freight, nil
	}

	// Filter out Freight whose soak time has not yet elapsed
	filtered := make([]kargoapi.Freight, 0, len(freight))
	for _, f := range freight {
		if opts.ApprovedFor != "" {
			if f.IsApprovedFor(opts.ApprovedFor) {
				filtered = append(filtered, f)
				continue
			}
		}

		// Track set of VerifiedIn Stages in which the Freight has passed the
		// verification soak time.
		verifiedStages := sets.New[string]()
		for _, stage := range opts.VerifiedIn {
			if f.HasSoakedIn(stage, opts.RequiredSoakTime) {
				verifiedStages.Insert(stage)
			}
		}

		// Filter out Freight that has passed its verification soak time in ALL
		// the specified VerifiedIn Stages if AvailabilityStrategy is set to All.
		// Otherwise, include Freight if it has passed the soak time in a single
		// Stage.
		if opts.AvailabilityStrategy == kargoapi.FreightAvailabilityStrategyAll {
			// If Freight is verified in ALL upstream Stages, then it is
			// available.
			if verifiedStages.Equal(sets.New(opts.VerifiedIn...)) {
				filtered = append(filtered, f)
			}
			continue
		}

		// If Freight is verified in ANY upstream Stage, then it is available.
		if verifiedStages.Len() > 0 {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

func (storage *Client) ClearStageFreights(ctx context.Context, project string, stageName string) error {
	freights, err := storage.listFreightsReferencingStage(ctx, project, stageName)
	if err != nil {
		return err
	}

	for _, freight := range freights {
		newStatus := *freight.Status.DeepCopy()
		delete(newStatus.CurrentlyIn, stageName)
		delete(newStatus.VerifiedIn, stageName)
		delete(newStatus.ApprovedFor, stageName)

		if err = storage.patchFreightStatus(ctx, storage.client, &freight, newStatus); err != nil {
			return fmt.Errorf("patch freight status: %w", err)
		}
	}

	return nil
}

func (storage *Client) MarkFreightsCurrentlyInStage(
	ctx context.Context,
	project string, stageName string,
	freightNames []string,
) error {
	logger := logging.LoggerFromContext(ctx)

	freights := []*kargoapi.Freight{}
	freightNamesSet := map[string]struct{}{}
	for _, freightName := range freightNames {
		freightNamesSet[freightName] = struct{}{}
		freight, err := storage.GetFreight(ctx, project, freightName)
		if err != nil {
			return err
		}
		if freight == nil {
			logger.Error(err, "freight is current but not found", "freightName", freightName)
			continue
		}
		freights = append(freights, freight)
	}

	// Unmark other Freight before marking the new one.
	// Only one freight should be Current per origin at a time.
	// Caller ensures that FreightNames are one per origin.
	current, err := storage.listFreightsByCurrentStage(ctx, project, stageName)
	if err != nil {
		return err
	}
	for i := range current {
		other := &current[i]
		if _, ok := freightNamesSet[other.Name]; ok {
			// Freight will be set as current, skipping clean
			continue
		}
		newStatus := *other.Status.DeepCopy()
		newStatus.RemoveCurrentStage(stageName)

		if err = storage.patchFreightStatus(ctx, storage.client, other, newStatus); err != nil {
			return fmt.Errorf("patch status of freight %q: %w", other.Name, err)
		}
	}

	for _, freight := range freights {
		// Evaluated before the patch so the event is emitted only on the first
		// transition, not on every re-mark.
		newlyCurrentlyIn := !freight.IsCurrentlyIn(stageName)

		newStatus := *freight.Status.DeepCopy()
		newStatus.AddCurrentStage(stageName, time.Now())
		if err = storage.patchFreightStatus(ctx, storage.client, freight, newStatus); err != nil {
			return fmt.Errorf("patch status of freight %q: %w", freight.Name, err)
		}

		if newlyCurrentlyIn {
			// Include verifiedIn information in the CurrentlyIn event
			emitFreightCurrentlyInStage(ctx, storage.eventSender, freight, stageName, freight.IsVerifiedIn(stageName))
		}
	}

	return nil
}

// emitFreightCurrentlyInStage sends a FreightCurrentlyInStage event for the
// Freight. It is a no-op when no event sender is configured.
func emitFreightCurrentlyInStage(
	ctx context.Context,
	eventSender event.Sender,
	freight *kargoapi.Freight,
	stageName string,
	verifiedInStage bool,
) {
	if eventSender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight currently in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightCurrentlyInStage(msg, actor, stageName, verifiedInStage, freight)
	if err == nil {
		err = eventSender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight currently in Stage event")
	}
}

func (storage *Client) MarkFreightsVerifiedInStage(
	ctx context.Context,
	project string, stageName string,
	freightNames []string,
	verifiedAtTime time.Time,
) error {
	logger := logging.LoggerFromContext(ctx)

	for _, freightName := range freightNames {

		freight, err := storage.GetFreight(ctx, project, freightName)
		if err != nil {
			return err
		}
		if freight == nil {
			logger.Error(err, "freight is verified but not found", "freightName", freightName)
			continue
		}

		// Evaluated before the patch so the event is emitted only on the first
		// transition, not on every re-mark.
		newlyVerified := !freight.IsVerifiedIn(stageName)

		newStatus := *freight.Status.DeepCopy()
		newStatus.AddVerifiedStage(stageName, verifiedAtTime)

		if err = storage.patchFreightStatus(ctx, storage.client, freight, newStatus); err != nil {
			return fmt.Errorf("patch freight status: %w", err)
		}

		if newlyVerified {
			emitFreightVerifiedInStage(ctx, storage.eventSender, freight, stageName)
		}
	}

	return nil
}

// emitFreightVerifiedInStage sends a FreightVerifiedInStage event for the
// Freight. It is a no-op when no event sender is configured.
func emitFreightVerifiedInStage(
	ctx context.Context,
	eventSender event.Sender,
	freight *kargoapi.Freight,
	stageName string,
) {
	if eventSender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight verified in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightVerifiedInStage(msg, actor, stageName, freight)
	if err == nil {
		err = eventSender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight verified in Stage event")
	}
}

func (storage *Client) patchFreightStatus(
	ctx context.Context,
	cl client.Client,
	freight *kargoapi.Freight,
	newStatus kargoapi.FreightStatus,
) error {
	if err := kubeclient.PatchStatus(
		ctx,
		cl,
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		return fmt.Errorf(
			"error patching Freight %q status in namespace %q: %w",
			freight.Name,
			freight.Namespace,
			err,
		)
	}
	return nil
}

func (storage *Client) listFreightsReferencingStage(
	ctx context.Context,
	project string, stageName string,
) ([]kargoapi.Freight, error) {
	// Field names match the indexer's verifiedIn and approvedFor indexes;
	// Index fields are referenced here to avoid circular dependency.
	// Indexers will be removed soon anyway, so it's not critical.
	byName := make(map[string]kargoapi.Freight)
	for _, field := range []string{"verifiedIn", "approvedFor", "currentlyIn"} {
		list := kargoapi.FreightList{}
		if err := storage.client.List(
			ctx,
			&list,
			client.InNamespace(project),
			client.MatchingFieldsSelector{
				Selector: fields.OneTermEqualSelector(field, stageName),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight referencing Stage %q in namespace %q: %w",
				stageName, project, err,
			)
		}
		for _, f := range list.Items {
			byName[f.Name] = f
		}
	}
	freights := make([]kargoapi.Freight, 0, len(byName))
	for _, f := range byName {
		freights = append(freights, f)
	}
	slices.SortFunc(freights, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	return freights, nil
}

func (storage *Client) listFreightsByCurrentStage(
	ctx context.Context,
	project string, stageName string) ([]kargoapi.Freight, error) {
	freightList := kargoapi.FreightList{}
	if err := storage.client.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingFields{"currentlyIn": stageName},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight in namespace %q with current stage %q: %w",
			project,
			stageName,
			err,
		)
	}
	return freightList.Items, nil
}
