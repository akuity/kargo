package server

import (
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// stageToSummary converts a Stage to the lightweight StageSummary projection
// returned by ListStageSummaries and WatchStageSummaries. See the StageSummary
// proto definition for the rationale. Returns nil if stage is nil.
func stageToSummary(stage *kargoapi.Stage) *svcv1alpha1.StageSummary {
	if stage == nil {
		return nil
	}

	meta := stage.ObjectMeta.DeepCopy()

	summary := &svcv1alpha1.StageSummary{
		Metadata: meta,
		Spec: &svcv1alpha1.StageSpecSummary{
			Shard:            stage.Spec.Shard,
			RequestedFreight: cloneFreightRequests(stage.Spec.RequestedFreight),
			HasVerification:  stage.Spec.Verification != nil,
		},
		Status: &svcv1alpha1.StageStatusSummary{
			Conditions:           cloneConditions(stage.Status.Conditions),
			LastHandledRefresh:   stage.Status.LastHandledRefresh,
			CurrentPromotion:     stage.Status.CurrentPromotion.DeepCopy(),
			LastPromotion:        stage.Status.LastPromotion.DeepCopy(),
			Health:               stage.Status.Health.DeepCopy(),
			ObservedGeneration:   stage.Status.ObservedGeneration,
			AutoPromotionEnabled: stage.Status.AutoPromotionEnabled,
		},
	}

	if stage.Spec.PromotionTemplate != nil {
		n := len(stage.Spec.PromotionTemplate.Spec.Steps)
		if n > int(math.MaxInt32) {
			n = int(math.MaxInt32)
		}
		summary.Spec.PromotionStepCount = int32(n)
	}

	if len(stage.Status.FreightHistory) > 0 {
		summary.Status.CurrentFreight = stage.Status.FreightHistory[0].DeepCopy()
	}

	return summary
}

// cloneFreightRequests returns a deep copy of the given FreightRequests as a
// slice of pointers (matching the shape of the generated protobuf field).
func cloneFreightRequests(src []kargoapi.FreightRequest) []*kargoapi.FreightRequest {
	if len(src) == 0 {
		return nil
	}
	dst := make([]*kargoapi.FreightRequest, len(src))
	for i := range src {
		dst[i] = src[i].DeepCopy()
	}
	return dst
}

// cloneConditions returns a deep copy of the given Conditions as a slice of
// pointers (matching the shape of the generated protobuf field). Each
// metav1.Condition contains only primitive fields, so a shallow per-element
// copy is sufficient.
func cloneConditions(src []metav1.Condition) []*metav1.Condition {
	if len(src) == 0 {
		return nil
	}
	dst := make([]*metav1.Condition, len(src))
	for i := range src {
		c := src[i]
		dst[i] = &c
	}
	return dst
}
