package targetstatuses

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
)

// governingStage is a target-aware Stage and what the statuses need of it.
type governingStage struct {
	// id is the Stage's UID, which is the id its rows in the database carry.
	id        string
	name      string
	selectors []labels.Selector
	// freight names the Freight in the Stage's current collection.
	freight []string
}

// fleet is a Project's Targets and the Stages that govern them, as worked
// out for one request.
type fleet struct {
	// targets holds every Target of the Project by id, governed or not.
	targets map[uuid.UUID]database.ListTargetLabelsRow
	// stages holds the Stages asked about, ordered by name.
	stages []governingStage
	// pairs holds each Target with each Stage that governs it, ordered by
	// Target name and then Stage name, both in byte order. That is the order
	// entries are served and paged in.
	pairs []targetstatusstore.Pair
}

// stageName returns the name of the Stage with the id.
func (f *fleet) stageName(id string) string {
	for _, stage := range f.stages {
		if stage.id == id {
			return stage.name
		}
	}
	return ""
}

// scope returns what the store is asked about: the given pairs, which are
// all of the fleet's or those left of them, and each Stage's current
// Freight.
func (f *fleet) scope(pairs []targetstatusstore.Pair) targetstatusstore.Scope {
	freight := make(map[string][]string, len(f.stages))
	for _, stage := range f.stages {
		if len(stage.freight) > 0 {
			freight[stage.id] = stage.freight
		}
	}
	return targetstatusstore.Scope{Pairs: pairs, StageFreight: freight}
}

// govern works out which of the Project's Stages govern which of its
// Targets, for all of its target-aware Stages or for the one named. Nothing
// records this: a Target is governed by a Stage when it matches any of the
// Stage's selectors, so it is settled here, from the selectors as they are
// now, exactly as creating a PromotionRollout settles which Targets to
// promote to. It costs one read of the Project's Stages, served from the
// Kubernetes client's cache, and one of its Targets' names and labels.
func (h *Handler) govern(ctx context.Context, project, stageName string) (*fleet, error) {
	stages, err := h.governingStages(ctx, project, stageName)
	if err != nil {
		return nil, err
	}
	rows, err := h.store.Targets(ctx, project)
	if err != nil {
		return nil, err
	}
	f := &fleet{
		targets: make(map[uuid.UUID]database.ListTargetLabelsRow, len(rows)),
		stages:  stages,
	}
	for _, row := range rows {
		f.targets[row.ID] = row
		for _, stage := range stages {
			if api.AnySelectorMatches(stage.selectors, row.Labels) {
				f.pairs = append(f.pairs, targetstatusstore.Pair{
					TargetID:   row.ID,
					TargetName: row.Name,
					StageID:    stage.id,
				})
			}
		}
	}
	// The store returns Targets in name order and the Stages are in name
	// order, so the pairs already are; sorting holds that whatever the store.
	slices.SortStableFunc(f.pairs, func(a, b targetstatusstore.Pair) int {
		return strings.Compare(a.TargetName, b.TargetName)
	})
	return f, nil
}

// governingStages reads the Project's target-aware Stages, or the one named,
// ordered by name. A classic Stage governs no Targets and is left out, so a
// classic Stage asked for by name yields none. Among all of a Project's
// Stages, one whose selectors do not parse governs nothing and is left out
// too, so that one malformed Stage does not hide the fleet; asked for by
// name, it is an error, as it is when listing the Targets it governs.
func (h *Handler) governingStages(
	ctx context.Context,
	project string,
	stageName string,
) ([]governingStage, error) {
	var stages []kargoapi.Stage
	if stageName != "" {
		stage := kargoapi.Stage{}
		if err := h.kube.Get(
			ctx,
			client.ObjectKey{Namespace: project, Name: stageName},
			&stage,
		); err != nil {
			return nil, err
		}
		stages = []kargoapi.Stage{stage}
	} else {
		list := kargoapi.StageList{}
		if err := h.kube.List(ctx, &list, client.InNamespace(project)); err != nil {
			return nil, err
		}
		stages = list.Items
	}
	governing := make([]governingStage, 0, len(stages))
	for i := range stages {
		stage := &stages[i]
		selectors, err := api.TargetSelectorsForStage(stage)
		if err != nil {
			if stageName != "" {
				return nil, libhttp.Error(err, http.StatusBadRequest)
			}
			logging.LoggerFromContext(ctx).Error(
				err, "Stage's target selectors do not parse; it governs no Targets",
				"project", project,
				"stage", stage.Name,
			)
			continue
		}
		if selectors == nil {
			continue
		}
		governing = append(governing, governingStage{
			id:        string(stage.UID),
			name:      stage.Name,
			selectors: selectors,
			freight:   currentFreight(stage),
		})
	}
	slices.SortFunc(governing, func(a, b governingStage) int {
		return strings.Compare(a.name, b.name)
	})
	return governing, nil
}

// currentFreight names the Freight in the Stage's current collection: what a
// Target must be on not to be behind the Stage. The database does not hold
// it, so it is read from the Stage's status, which came with the Stage.
func currentFreight(stage *kargoapi.Stage) []string {
	refs := stage.Status.FreightHistory.Current().References()
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.Name
	}
	return names
}
