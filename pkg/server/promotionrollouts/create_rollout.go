package promotionrollouts

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/targets"
	"github.com/akuity/kargo/pkg/server/user"
)

// Creator creates PromotionRollouts. The create endpoint and the promote
// endpoints all promote to a target-aware Stage through one, so that a
// rollout is built the same way whichever of them asked for it.
type Creator interface {
	// Create records the intent to promote the Freight to the Targets the
	// Stage governs, or to the named ones among them, and returns the rollout
	// as stored. Which Targets those are is decided here, once: the Stage's
	// selectors are resolved against the Project's Targets, narrowed to
	// targetNames if any are given, and the rollout's update strategy is
	// resolved against how many that leaves; both are recorded with the
	// rollout for good. The strategy is the Stage's, or updateStrategy in its
	// place when the caller gives one, which is how a retry runs one Target
	// at a time. A name the Stage does not govern is refused rather than
	// promoted to, since the Stage's selectors are what decide where its
	// Freight may go. The Stage must be target-aware and the Freight
	// available to it: the caller has checked both, and that the caller may
	// promote to the Stage, which is the whole authorization decision. The
	// identity bound to the context, if any, is recorded as the rollout's
	// creator.
	//
	// A Creator built with a check asks it first whether the Stage may be
	// promoted to at this time, and creates nothing if it says not; see
	// NewCreator.
	//
	// The errors it returns carry an HTTP status where one applies, as a
	// handler's would.
	Create(
		ctx context.Context,
		stage *kargoapi.Stage,
		freight *kargoapi.Freight,
		targetNames []string,
		updateStrategy *kargoapi.TargetUpdateStrategy,
	) (*kargoapi.PromotionRollout, error)
}

// creator is the Creator backed by the rollout store.
type creator struct {
	// store is where rollouts are written.
	store promotionrolloutstore.Store
	// targets is where the Targets a rollout promotes to are read from.
	targets targets.Lister
	// check, when set, is asked before a rollout is created whether the Stage
	// may be promoted to at this time. An error from it refuses the rollout.
	check func(context.Context, *kargoapi.Stage) error
}

// NewCreator returns a Creator that writes rollouts to the store and reads
// the Targets they promote to from targetLister.
//
// check may be nil. When it is not, it is asked before each rollout is
// created whether the Stage may be promoted to at this time, and an error
// from it refuses the rollout: Create returns the error as it is, so one that
// carries an HTTP status is answered with that status and its message.
func NewCreator(
	store promotionrolloutstore.Store,
	targetLister targets.Lister,
	check func(context.Context, *kargoapi.Stage) error,
) Creator {
	return &creator{store: store, targets: targetLister, check: check}
}

// Create implements Creator.
//
// The Project's Targets are read without the caller's authorization. Which
// Targets the Stage governs is a detail of carrying out the promotion the
// caller is allowed to make; checking it as the caller would make promoting
// depend on also being allowed to list Targets, which a role granting the
// promote verb need not include.
func (c *creator) Create(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	targetNames []string,
	updateStrategy *kargoapi.TargetUpdateStrategy,
) (*kargoapi.PromotionRollout, error) {
	// The check comes before anything is read or written, so a rollout it
	// refuses costs nothing. Its error is the answer and is not wrapped: the
	// message is whatever the check has to say to the caller.
	if c.check != nil {
		if err := c.check(ctx, stage); err != nil {
			return nil, err
		}
	}
	all, err := targets.ListFromStore(ctx, c.targets, stage.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", stage.Namespace, err)
	}
	governed, err := api.FilterTargetsForStage(stage, all)
	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}
	var rollout *kargoapi.PromotionRollout
	if len(targetNames) > 0 {
		rollout, err = api.NewPromotionRolloutForSelectTargets(
			stage, freight.Name, governed, targetNames, updateStrategy)
	} else {
		rollout, err = api.NewPromotionRolloutForAllTargets(
			stage, freight.Name, governed, updateStrategy)
	}

	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}

	targetIds, err := getTargetIds(rollout, governed)
	if err != nil {
		return nil, err
	}

	var actor string
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
	}
	row := promotionrolloutstore.FromSpec(rollout.Name, rollout.Spec)
	row.CreatedBy = actor
	stored, err := c.store.Create(ctx, stage.Namespace, row, targetIds)
	if err != nil {
		return nil, storeError(err, rollout.Name)
	}
	logChange(ctx, "PromotionRollout created", stage.Namespace, rollout.Name, actor)
	return promotionrolloutstore.ToAPI(stored), nil
}

// TODO: we might instead add UID as a field to PromotionRolloutTarget and resolve IDs in NewPromotionRollout functions
func getTargetIds(rollout *kargoapi.PromotionRollout, stageTargets []kargoapi.Target) ([]uuid.UUID, error) {
	stageTargetsByName := make(map[string]kargoapi.Target, len(stageTargets))
	for _, stageTarget := range stageTargets {
		stageTargetsByName[stageTarget.Name] = stageTarget
	}
	ids := make([]uuid.UUID, len(rollout.Spec.Targets))
	var err error
	for i, target := range rollout.Spec.Targets {
		if ids[i], err = uuid.Parse(string(stageTargetsByName[target.Name].UID)); err != nil {
			return nil, fmt.Errorf("error reading the id of Target %q: %w", target.Name, err)
		}
	}
	return ids, nil
}

// logChange records a change to a PromotionRollout and who made it, which
// the caller has already worked out from the request. PromotionRollouts are
// not Kubernetes resources, so the Kubernetes audit log never sees these.
func logChange(ctx context.Context, what, project, name, actor string) {
	if actor == "" {
		actor = "unknown"
	}
	logging.LoggerFromContext(ctx).Info(
		what,
		"project", project,
		"promotionRollout", name,
		"actor", actor,
	)
}
