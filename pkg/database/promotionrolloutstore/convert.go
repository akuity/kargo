package promotionrolloutstore

import (
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
	// spec.targets is absent until the rollout's Targets have been resolved,
	// and an empty list once it resolved to none.
	var targets []kargoapi.PromotionRolloutTarget
	if rollout.TargetsResolvedAt != nil {
		targets = make([]kargoapi.PromotionRolloutTarget, len(rollout.Targets))
		for i, target := range rollout.Targets {
			targets[i] = kargoapi.PromotionRolloutTarget{Name: target.TargetName}
		}
	}
	return &kargoapi.PromotionRollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         rollout.ProjectName,
			Name:              rollout.Name,
			UID:               types.UID(rollout.ID.String()),
			ResourceVersion:   resourceVersion(rollout.UpdatedAt),
			CreationTimestamp: metav1.NewTime(rollout.CreatedAt),
			Labels:            map[string]string{kargoapi.LabelKeyStage: rollout.StageName},
		},
		Number: rollout.Number,
		Spec: kargoapi.PromotionRolloutSpec{
			Stage:   rollout.StageName,
			Freight: rollout.FreightName,
			Targets: targets,
			UpdateStrategy: kargoapi.TargetUpdateStrategy{
				MaxConcurrent: rollout.MaxConcurrent,
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

// FromSpec takes what the database stores from a PromotionRollout's spec:
// its Stage, Freight and max concurrency, under the given name. Which
// Targets it promotes to is given to Create separately, by id, since the
// spec names them only.
func FromSpec(name string, spec kargoapi.PromotionRolloutSpec) database.PromotionRolloutRow {
	return database.PromotionRolloutRow{
		Name:          name,
		StageName:     spec.Stage,
		FreightName:   spec.Freight,
		MaxConcurrent: spec.UpdateStrategy.MaxConcurrent,
	}
}

// readyCondition states the rollout's phase, reason and message as a Ready
// condition: True once the rollout succeeded, False with the reason it did
// not, and Unknown while it is still in progress.
func readyCondition(row database.PromotionRolloutRow) metav1.Condition {
	phase := kargoapi.PromotionRolloutPhase(row.StatusPhase)
	cond := metav1.Condition{
		Type:    kargoapi.ConditionTypeReady,
		Status:  metav1.ConditionUnknown,
		Reason:  row.StatusReason,
		Message: row.StatusMessage,
	}
	switch {
	case phase == kargoapi.PromotionRolloutPhaseSucceeded:
		cond.Status = metav1.ConditionTrue
	case phase.IsTerminal():
		cond.Status = metav1.ConditionFalse
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
