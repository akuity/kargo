package promotionrolloutstore

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// ToAPI presents a stored rollout as the PromotionRollout resource the API
// serves. The row's id is the UID and its last update, which every write
// advances, the resource version. The status is the row's status columns,
// summarized as a Ready condition as well; what has become of each Target is
// not the store's to say, so status.targets and status.summary are left for
// the caller, who reads the Promotions.
func ToAPI(rollout Rollout) *kargoapi.PromotionRollout {
	// spec.targets is a required field, so it is a list even when the
	// rollout promotes to no Targets.
	targets := make([]kargoapi.PromotionRolloutTarget, len(rollout.Targets))
	for i, target := range rollout.Targets {
		targets[i] = kargoapi.PromotionRolloutTarget{Name: target.TargetName}
	}
	// Who asked for the rollout, and whether, and by whom, it was asked to
	// abort, are served the way a Promotion says them, so one reader serves
	// both.
	annotations := map[string]string{}
	if rollout.CreatedBy != "" {
		annotations[kargoapi.AnnotationKeyCreateActor] = rollout.CreatedBy
	}
	if rollout.AbortRequestedAt != nil {
		annotations[kargoapi.AnnotationKeyAbort] = (&kargoapi.AbortPromotionRequest{
			Action: kargoapi.AbortActionTerminate,
			Actor:  rollout.AbortRequestedBy,
		}).String()
	}
	if len(annotations) == 0 {
		annotations = nil
	}
	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         rollout.ProjectName,
			Name:              rollout.Name,
			UID:               types.UID(rollout.ID.String()),
			ResourceVersion:   resourceVersion(rollout.UpdatedAt),
			CreationTimestamp: metav1.NewTime(rollout.CreatedAt),
			Labels:            map[string]string{kargoapi.LabelKeyStage: rollout.StageName},
			Annotations:       annotations,
		},
		Number: rollout.Number,
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:   rollout.StageName,
			Freight: rollout.FreightName,
			Targets: targets,
			UpdateStrategy: kargoapi.PromotionRolloutUpdateStrategy{
				MaxConcurrent: rollout.MaxConcurrent,
				MaxFailures:   rollout.MaxFailures,
			},
		},
		Status: kargoapi.PromotionRolloutStatus{
			Conditions: []metav1.Condition{readyCondition(rollout.PromotionRolloutRow)},
			Phase:      kargoapi.PromotionRolloutPhase(rollout.StatusPhase),
			Message:    rollout.StatusMessage,
			StartedAt:  timeOrNil(rollout.StatusStartedAt),
			FinishedAt: timeOrNil(rollout.StatusFinishedAt),
		},
	}
}

// TargetStatuses presents the rows of ListTargetStatuses as the rollout's
// status.targets, one entry per Target in the order the rows came, with a
// tally of their phases as status.summary. Both are nil when there are no
// rows: no Target has a Promotion yet.
func TargetStatuses(
	rows []database.ListPromotionRolloutTargetStatusesRow,
) ([]kargoapi.PromotionRolloutTargetStatus, *kargoapi.PromotionRolloutSummary) {
	if len(rows) == 0 {
		return nil, nil
	}
	statuses := make([]kargoapi.PromotionRolloutTargetStatus, len(rows))
	summary := &kargoapi.PromotionRolloutSummary{}
	for i, row := range rows {
		phase := kargoapi.PromotionPhase(row.Phase)
		statuses[i] = kargoapi.PromotionRolloutTargetStatus{
			Name:      row.TargetName,
			Promotion: row.PromotionName,
			Phase:     phase,
		}
		switch phase {
		case kargoapi.PromotionPhaseRunning:
			summary.Running++
		case kargoapi.PromotionPhaseSucceeded:
			summary.Succeeded++
		case kargoapi.PromotionPhaseFailed:
			summary.Failed++
		case kargoapi.PromotionPhaseErrored:
			summary.Errored++
		case kargoapi.PromotionPhaseAborted:
			summary.Aborted++
		default:
			summary.Pending++
		}
	}
	return statuses, summary
}

// freightCollection presents the rows recorded for a rollout as the
// FreightCollection they are: each row's reference is a FreightReference as
// the API serves it, and the collection is keyed by origin and identified
// as any other is. No rows are an empty collection.
func freightCollection(
	rows []database.PromotionRolloutFreightRow,
) (*kargoapi.FreightCollection, error) {
	collection := &kargoapi.FreightCollection{}
	if len(rows) == 0 {
		return collection, nil
	}
	references := make([]kargoapi.FreightReference, len(rows))
	for i, row := range rows {
		if err := json.Unmarshal(row.Reference, &references[i]); err != nil {
			return nil, fmt.Errorf(
				"error decoding Freight %q from %s %q: %w",
				row.FreightName, row.OriginKind, row.OriginName, err,
			)
		}
	}
	collection.UpdateOrPush(references...)
	return collection, nil
}

// FromSpec takes what the database stores from a PromotionRollout's spec:
// its Stage, Freight and update strategy, under the given name. Which
// Targets it promotes to is given to Create separately, by id, since the
// spec names them only.
func FromSpec(name string, spec kargoapi.PromotionRolloutSpec) database.PromotionRolloutRow {
	return database.PromotionRolloutRow{
		Name:          name,
		StageName:     spec.Stage,
		FreightName:   spec.Freight,
		MaxConcurrent: spec.UpdateStrategy.MaxConcurrent,
		MaxFailures:   spec.UpdateStrategy.MaxFailures,
	}
}

// readyCondition states the rollout's phase, reason and message as a Ready
// condition. It tracks whether the rollout has finished well: True once it
// succeeded, and False otherwise, both while it is still in progress and once
// it has ended without succeeding. The reason tells those apart.
//
// No transition time is stored, so it is derived from the times that are. The
// condition is False from the rollout's creation, and changes only if the
// rollout succeeds, so it last changed when a succeeded rollout finished, or,
// for any other, when the rollout was created. A condition must carry one:
// without it the time is served as null, which clients that read the field as
// a required date-time fail to decode.
func readyCondition(row database.PromotionRolloutRow) metav1.Condition {
	phase := kargoapi.PromotionRolloutPhase(row.StatusPhase)
	cond := metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             row.StatusReason,
		Message:            row.StatusMessage,
		LastTransitionTime: metav1.NewTime(row.CreatedAt),
	}
	if phase == kargoapi.PromotionRolloutPhaseSucceeded {
		cond.Status = metav1.ConditionTrue
		if row.StatusFinishedAt != nil {
			cond.LastTransitionTime = metav1.NewTime(*row.StatusFinishedAt)
		}
	}
	if cond.Reason == "" {
		cond.Reason = string(phase)
	}
	if cond.Reason == "" {
		cond.Reason = string(kargoapi.PromotionRolloutPhasePending)
	}
	return cond
}

// resourceVersion renders a row's last update as the resource version of
// the object the API serves: microseconds since the epoch, which the
// database keeps unique per row.
func resourceVersion(updatedAt time.Time) string {
	return strconv.FormatInt(updatedAt.UnixMicro(), 10)
}

func timeOrNil(t *time.Time) *metav1.Time {
	if t == nil {
		return nil
	}
	mt := metav1.NewTime(*t)
	return &mt
}
