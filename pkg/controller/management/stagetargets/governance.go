package stagetargets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
)

// pair is one Stage governing one Target.
type pair struct {
	stageID  string
	targetID uuid.UUID
}

// governance works out which of a Project's Stages govern which of its
// Targets: a Stage governs a Target when the Target's labels match any one of
// the Stage's selectors, exactly as the controller decides when it promotes.
// A Stage whose selectors cannot be parsed governs nothing, as it governs
// nothing for the controller, and is logged; one bad Stage does not stop the
// rest of the Project from being settled.
func governance(
	ctx context.Context,
	stages []database.ListStageSelectorsRow,
	targets []database.ListTargetLabelsRow,
) map[pair]struct{} {
	logger := logging.LoggerFromContext(ctx)
	governed := map[pair]struct{}{}
	for _, stage := range stages {
		selectors, err := parseSelectors(stage.TargetSelectors)
		if err != nil {
			logger.Error(
				err, "Stage's target selectors do not parse; it governs no Targets",
				"stage", stage.Name,
			)
			continue
		}
		for _, target := range targets {
			if api.AnySelectorMatches(selectors, target.Labels) {
				governed[pair{stageID: stage.ID, targetID: target.ID}] = struct{}{}
			}
		}
	}
	return governed
}

// parseSelectors parses a Stage's mirrored selectors, the JSON of its
// spec.targets.selectors, as the controller parses the Stage's own.
func parseSelectors(raw json.RawMessage) ([]labels.Selector, error) {
	var selectors []metav1.LabelSelector
	if err := json.Unmarshal(raw, &selectors); err != nil {
		return nil, fmt.Errorf("error decoding target selectors: %w", err)
	}
	if selectors == nil {
		selectors = []metav1.LabelSelector{}
	}
	return api.TargetSelectorsForStage(&kargoapi.Stage{
		Spec: kargoapi.StageSpec{Targets: &kargoapi.StageTargets{Selectors: selectors}},
	})
}
