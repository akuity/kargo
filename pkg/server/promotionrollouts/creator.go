package promotionrollouts

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/targets"
)

// Creator creates PromotionRollouts. The promote endpoints and the create
// endpoint share one, so that a rollout is built the same way whichever
// path asks for it.
type Creator struct {
	store   promotionrolloutstore.Store
	targets targets.Lister
}

// NewCreator returns a Creator that records rollouts in the store and reads
// the Targets they promote to from targetLister.
func NewCreator(store promotionrolloutstore.Store, targetLister targets.Lister) *Creator {
	return &Creator{store: store, targets: targetLister}
}

// Create records the intent to promote the Freight to the Targets the Stage
// governs, or to the named ones among them, and returns the rollout as
// stored. Which Targets those are is decided here, once: the Stage's
// selectors are resolved against the Project's Targets, narrowed to
// targetNames if any are given, and the rollout's update strategy is
// resolved against how many that leaves; both are recorded with the rollout
// for good. A name the Stage does not govern is refused rather than
// promoted to, since the Stage's selectors are what decide where its
// Freight may go. The Stage must be target-aware and the Freight available
// to it: the caller has checked both, and that the caller may promote to the
// Stage, which is the whole authorization decision. actor is recorded as the
// rollout's creator.
//
// The Project's Targets are read without the caller's authorization. Which
// Targets the Stage governs is a detail of carrying out the promotion the
// caller is allowed to make; checking it as the caller would make promoting
// depend on also being allowed to list Targets, which a role granting the
// promote verb need not include.
func (cr *Creator) Create(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
	targetNames []string,
	actor string,
) (*kargoapi.PromotionRollout, error) {
	all, err := targets.ListFromStore(ctx, cr.targets, stage.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", stage.Namespace, err)
	}
	governed, err := api.FilterTargetsForStage(stage, all)
	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}
	if len(targetNames) > 0 {
		if governed, err = selectTargets(stage.Name, governed, targetNames); err != nil {
			return nil, err
		}
	}
	rollout, err := api.NewPromotionRollout(stage, freight.Name, governed)
	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}
	ids := make([]uuid.UUID, len(governed))
	for i, target := range governed {
		if ids[i], err = uuid.Parse(string(target.UID)); err != nil {
			return nil, fmt.Errorf("error reading the id of Target %q: %w", target.Name, err)
		}
	}
	row := promotionrolloutstore.FromSpec(rollout.Name, rollout.Spec)
	row.CreatedBy = actor
	stored, err := cr.store.Create(ctx, stage.Namespace, row, ids)
	if err != nil {
		return nil, storeError(err, rollout.Name)
	}
	return promotionrolloutstore.ToAPI(stored), nil
}

// selectTargets returns those of the Stage's governed Targets that are
// named, in the order governed lists them. Every name must be governed; the
// ones that are not are reported together, as a bad request.
func selectTargets(stage string, governed []kargoapi.Target, names []string) ([]kargoapi.Target, error) {
	selected := make([]kargoapi.Target, 0, len(names))
	for _, target := range governed {
		if slices.Contains(names, target.Name) {
			selected = append(selected, target)
		}
	}
	if len(selected) == len(names) {
		return selected, nil
	}
	var ungoverned []string
	for _, name := range names {
		if !slices.ContainsFunc(governed, func(t kargoapi.Target) bool { return t.Name == name }) {
			ungoverned = append(ungoverned, name)
		}
	}
	return nil, libhttp.ErrorStr(
		fmt.Sprintf(
			"Stage %q does not govern Targets %s; a PromotionRollout promotes only to Targets its Stage selects",
			stage, strings.Join(ungoverned, ", "),
		),
		http.StatusBadRequest,
	)
}
