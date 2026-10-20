package targets

import (
	"context"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
)

// TargetSummary is what the views of a whole fleet draw for one Target: its
// name, its labels, and where it stands with each Stage that governs it. It
// leaves out what only a view of the one Target needs, such as its params,
// which the Target itself carries.
type TargetSummary struct {
	// Name is the Target's name.
	Name string `json:"name"`
	// Labels are the Target's labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Stages describes the Target with respect to each Stage that governs it,
	// keyed by Stage name. A Stage that does not govern the Target is absent,
	// even if it once promoted to it.
	Stages map[string]TargetStageSummary `json:"stages,omitempty"`
} // @name TargetSummary

// TargetStageSummary describes a Target with respect to one Stage that
// governs it.
type TargetStageSummary struct {
	// CollectionID identifies the Freight the Stage's most recent successful
	// Promotion to the Target left there. It is computed as the Stage computes
	// the ID of its own current collection, so the two are equal exactly when
	// the Target has the Stage's current Freight. It is absent until a
	// Promotion from the Stage to the Target has succeeded.
	CollectionID string `json:"collectionID,omitempty"`
	// Freight names the Freight in that collection, sorted.
	Freight []string `json:"freight,omitempty"`
	// LastPromotion is the Stage's most recent Promotion to the Target,
	// whatever its phase. It is absent until there has been one.
	LastPromotion *TargetPromotion `json:"lastPromotion,omitempty"`
	// Health is the Target's health with respect to the Stage's Freight. It
	// is absent until it has been assessed, and nothing assesses it yet.
	Health *kargoapi.Health `json:"health,omitempty"`
} // @name TargetStageSummary

// TargetPromotion is a Promotion to a Target, as the Promotion history
// records it. It outlives the Promotion in Kubernetes.
type TargetPromotion struct {
	// Name is the Promotion's name.
	Name string `json:"name"`
	// Stage names the Stage that promoted.
	Stage string `json:"stage"`
	// Freight names the Freight promoted.
	Freight string `json:"freight"`
	// Phase is the Promotion's phase, absent until it has one.
	Phase string `json:"phase,omitempty"`
	// CreatedAt is when the Promotion was created.
	CreatedAt metav1.Time `json:"createdAt"`
	// FinishedAt is when the Promotion finished, absent until it has.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
} // @name TargetPromotion

// TargetSummaryList is a list of TargetSummaries.
type TargetSummaryList struct {
	Items []TargetSummary `json:"items"`
} // @name TargetSummaryList

// @id ListTargetSummaries
// @Summary List Target summaries
// @Description List a summary of each of a project's Targets: its name, its
// @Description labels, and, for each Stage that governs it, the Freight it
// @Description runs and the latest Promotion to it. Optionally narrowed to
// @Description the Targets a Stage governs or those matching a label
// @Description selector. Returns a TargetSummaryList.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only Targets governed by this Stage, i.e. selected by its targets.selectors"
// @Param labelSelector query string false "Kubernetes label selector to filter Targets by, e.g. region=us,tier!=canary"
// @Produce json
// @Success 200 {object} TargetSummaryList "TargetSummaryList"
// @Router /v1beta1/projects/{project}/target-summaries [get]
func (h *Handler) listSummaries(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	selector, err := parseLabelSelectorQuery(c.Query("labelSelector"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	stages, err := h.governingStages(c, project, c.Query("stage"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	rows, err := h.store.List(ctx, project)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// A Target is summarized when it matches the label selector, if any, and,
	// when the list is narrowed to a Stage, that Stage governs it.
	ids := make([]uuid.UUID, 0, len(rows))
	summaries := make([]TargetSummary, 0, len(rows))
	for _, row := range rows {
		if selector != nil && !selector.Matches(labels.Set(row.Labels)) {
			continue
		}
		governing := map[string]TargetStageSummary{}
		for _, stage := range stages {
			if api.AnySelectorMatches(stage.selectors, row.Labels) {
				governing[stage.name] = TargetStageSummary{}
			}
		}
		if c.Query("stage") != "" && len(governing) == 0 {
			continue
		}
		var lbls map[string]string
		if len(row.Labels) > 0 {
			lbls = row.Labels
		}
		ids = append(ids, row.ID)
		summaries = append(summaries, TargetSummary{Name: row.Name, Labels: lbls, Stages: governing})
	}
	if err = h.setSummaryStatuses(ctx, project, ids, summaries); err != nil {
		_ = c.Error(err)
		return
	}
	for i := range summaries {
		if len(summaries[i].Stages) == 0 {
			summaries[i].Stages = nil
		}
	}
	c.JSON(http.StatusOK, &TargetSummaryList{Items: summaries})
}

// stageSelectors is a target-aware Stage's name and parsed target selectors.
type stageSelectors struct {
	name      string
	selectors []labels.Selector
}

// governingStages returns the target-aware Stages of the Project, or only the
// named one, with their selectors parsed. A named Stage that does not exist
// is an error. A Stage among all of them whose selectors do not parse governs
// nothing and is left out, so that one malformed Stage does not hide the
// fleet; the named one is an error, as it is for a list of Targets.
func (h *Handler) governingStages(
	c *gin.Context,
	project string,
	stageName string,
) ([]stageSelectors, error) {
	ctx := c.Request.Context()
	if stageName != "" {
		selectors, err := h.stageSelectors(c, project, stageName)
		if err != nil {
			return nil, err
		}
		return []stageSelectors{{name: stageName, selectors: selectors}}, nil
	}
	list := &kargoapi.StageList{}
	if err := h.kube.List(ctx, list, client.InNamespace(project)); err != nil {
		return nil, err
	}
	stages := make([]stageSelectors, 0, len(list.Items))
	for i := range list.Items {
		stage := &list.Items[i]
		selectors, err := api.TargetSelectorsForStage(stage)
		if err != nil {
			logging.LoggerFromContext(ctx).Error(
				err, "Stage's target selectors do not parse; it governs no Targets",
				"project", project,
				"stage", stage.Name,
			)
			continue
		}
		if selectors != nil {
			stages = append(stages, stageSelectors{name: stage.Name, selectors: selectors})
		}
	}
	return stages, nil
}

// setSummaryStatuses fills in, for each summarized Target and Stage that
// governs it, the Freight the Stage's latest successful Promotion left there
// and the Stage's latest Promotion to it. The summaries are those of the
// Targets with the given ids, in the same order. Records of Stages that no
// longer govern a Target are ignored. Nothing is read for no Targets.
func (h *Handler) setSummaryStatuses(
	ctx context.Context,
	project string,
	ids []uuid.UUID,
	summaries []TargetSummary,
) error {
	if len(summaries) == 0 {
		return nil
	}
	byID := make(map[uuid.UUID]*TargetSummary, len(ids))
	for i, id := range ids {
		byID[id] = &summaries[i]
	}
	freight, err := h.store.CurrentFreight(ctx, project, "")
	if err != nil {
		return err
	}
	collections := map[uuid.UUID]map[string]*kargoapi.FreightCollection{}
	for _, row := range freight {
		summary, ok := byID[row.TargetID]
		if !ok {
			continue
		}
		if _, governs := summary.Stages[row.StageName]; !governs {
			continue
		}
		if collections[row.TargetID] == nil {
			collections[row.TargetID] = map[string]*kargoapi.FreightCollection{}
		}
		collection := collections[row.TargetID][row.StageName]
		if collection == nil {
			collection = &kargoapi.FreightCollection{}
			collections[row.TargetID][row.StageName] = collection
		}
		collection.UpdateOrPush(freightReference(row))
	}
	for id, byStage := range collections {
		summary := byID[id]
		for stage, collection := range byStage {
			entry := summary.Stages[stage]
			entry.CollectionID = collection.ID
			for _, ref := range collection.References() {
				entry.Freight = append(entry.Freight, ref.Name)
			}
			slices.Sort(entry.Freight)
			summary.Stages[stage] = entry
		}
	}

	latest, err := h.store.LatestPromotions(ctx, project)
	if err != nil {
		return err
	}
	for _, row := range latest {
		summary, ok := byID[row.TargetID]
		if !ok {
			continue
		}
		entry, governs := summary.Stages[row.StageName]
		if !governs {
			continue
		}
		entry.LastPromotion = targetPromotion(database.PromotionRow{
			Name:        row.Name,
			StageName:   row.StageName,
			FreightName: row.FreightName,
			Phase:       row.Phase,
			CreatedAt:   row.CreatedAt,
			FinishedAt:  row.FinishedAt,
		})
		summary.Stages[row.StageName] = entry
	}
	return nil
}

// freightReference names one origin's Freight in a Target's current
// collection.
func freightReference(row database.ListTargetCurrentFreightRow) kargoapi.FreightReference {
	return kargoapi.FreightReference{
		Name: row.FreightName,
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKind(row.OriginKind),
			Name: row.OriginName,
		},
	}
}

// targetPromotion presents a row of the Promotion history.
func targetPromotion(row database.PromotionRow) *TargetPromotion {
	promotion := &TargetPromotion{
		Name:      row.Name,
		Stage:     row.StageName,
		Freight:   row.FreightName,
		Phase:     row.Phase,
		CreatedAt: metav1.NewTime(row.CreatedAt),
	}
	if row.FinishedAt != nil {
		finished := metav1.NewTime(*row.FinishedAt)
		promotion.FinishedAt = &finished
	}
	return promotion
}
