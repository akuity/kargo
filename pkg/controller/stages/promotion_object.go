package stages

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/indexer"
)

type PromotionObject interface {
	// Getters:
	metav1.Object // Covers Name, CreationTimestamp and Annotations access
	GetPhase() PromotionObjectPhase
	GetFinishedAt() *metav1.Time
	GetFreightName() string
	GetArgoCDRefs() []api.ArgoCDAppRef
	GetStartedAt() *metav1.Time

	ToLastPromotionReference() PromotionObjectReference

	updateCurrentPromotion(kargoapi.StageStatus) kargoapi.StageStatus
}

type PromotionObjectReference interface {
	GetName() string
	GetPhase() PromotionObjectPhase

	GetFinishedAt() *metav1.Time

	// For LastPromotion only:
	GetHealthChecks() []health.Criteria
	// Maybe Freight reference??
	GetFreightReference() *kargoapi.FreightReference
	GetFreightCollection() *kargoapi.FreightCollection
	SetFreightCollection(*kargoapi.FreightCollection)

	GetMessage() string

	updateLastPromotion(kargoapi.StageStatus) kargoapi.StageStatus
}

type PromotionObjectPhase interface {
	IsSucceeded() bool
	IsAborted() bool
	IsRunning() bool
	IsTerminal() bool
	String() string
}

// PromotionObjectPhase implementations

type promotionObjectPhasePromotion kargoapi.PromotionPhase
type promotionObjectPhaseRequest kargoapi.PromotionRequestPhase

func (p promotionObjectPhasePromotion) IsTerminal() bool {
	phase := kargoapi.PromotionPhase(p)
	return phase.IsTerminal()
}

func (p promotionObjectPhasePromotion) IsSucceeded() bool {
	return kargoapi.PromotionPhase(p) == kargoapi.PromotionPhaseSucceeded
}

func (p promotionObjectPhasePromotion) IsAborted() bool {
	return kargoapi.PromotionPhase(p) == kargoapi.PromotionPhaseAborted
}

func (p promotionObjectPhasePromotion) IsRunning() bool {
	return kargoapi.PromotionPhase(p) == kargoapi.PromotionPhaseRunning
}

func (p promotionObjectPhasePromotion) String() string {
	return string(kargoapi.PromotionPhase(p))
}

func (p promotionObjectPhaseRequest) IsTerminal() bool {
	phase := kargoapi.PromotionRequestPhase(p)
	return phase.IsTerminal()
}

func (p promotionObjectPhaseRequest) IsSucceeded() bool {
	return kargoapi.PromotionRequestPhase(p) == kargoapi.PromotionRequestPhaseSucceeded
}

func (p promotionObjectPhaseRequest) IsAborted() bool {
	// FIXME: implement abort for promotion requests
	return false
	// return kargoapi.PromotionRequestPhase(p) == kargoapi.PromotionRequestPhaseAborted
}

func (p promotionObjectPhaseRequest) IsRunning() bool {
	return kargoapi.PromotionRequestPhase(p) == kargoapi.PromotionRequestPhaseRunning
}

func (p promotionObjectPhaseRequest) String() string {
	return string(kargoapi.PromotionRequestPhase(p))
}

// PromotionObjectReference implementations

type promotionObjectReferencePromotion struct {
	kargoapi.PromotionReference
}
type promotionObjectReferenceRequest struct {
	kargoapi.PromotionRequestReference
}

// PromotionObjectReference for PromotionReference

func newPromotionObjectReferencePromotion(ref *kargoapi.PromotionReference) PromotionObjectReference {
	if ref == nil {
		return nil
	}
	return &promotionObjectReferencePromotion{
		PromotionReference: *ref,
	}
}

func (r *promotionObjectReferencePromotion) GetName() string {
	return r.Name
}

// FIXME: what to do with nil phase? Check usages
func (r *promotionObjectReferencePromotion) GetPhase() PromotionObjectPhase {
	if r.Status == nil {
		return promotionObjectPhasePromotion("")
	}
	return promotionObjectPhasePromotion(r.Status.Phase)
}
func (r *promotionObjectReferencePromotion) GetFinishedAt() *metav1.Time {
	return r.FinishedAt
}
func (r *promotionObjectReferencePromotion) GetHealthChecks() []health.Criteria {
	if r.Status == nil {
		return nil
	}
	return healthChecksToCriteria(r.Status.HealthChecks)
}
func (r *promotionObjectReferencePromotion) GetFreightReference() *kargoapi.FreightReference {
	return r.Freight
}
func (r *promotionObjectReferencePromotion) GetFreightCollection() *kargoapi.FreightCollection {
	if r.Status == nil {
		return nil
	}
	return r.Status.FreightCollection
}

func (r *promotionObjectReferencePromotion) SetFreightCollection(collection *kargoapi.FreightCollection) {
	// We use this method under assumption that status is not nil
	if r.Status == nil {
		return
	}
	r.Status.FreightCollection = collection
}
func (r *promotionObjectReferencePromotion) GetMessage() string {
	if r.Status == nil {
		return ""
	}
	return r.Status.Message
}
func (r *promotionObjectReferencePromotion) updateLastPromotion(stageStatus kargoapi.StageStatus) kargoapi.StageStatus {
	stageStatus.LastPromotion = &r.PromotionReference
	return stageStatus
}

// PromotionObjectReference for PromotionRequestReference

func newPromotionObjectReferenceRequest(ref *kargoapi.PromotionRequestReference) PromotionObjectReference {
	if ref == nil {
		return nil
	}
	return &promotionObjectReferenceRequest{
		PromotionRequestReference: *ref,
	}
}

func (r *promotionObjectReferenceRequest) GetName() string {
	return r.Name
}

// FIXME: what to do with nil phase? Check usages
// FIXME: add status to PromotionRequestReference
func (r *promotionObjectReferenceRequest) GetPhase() PromotionObjectPhase {
	return promotionObjectPhasePromotion(r.Phase)
}
func (r *promotionObjectReferenceRequest) GetFinishedAt() *metav1.Time {
	return r.FinishedAt
}
func (r *promotionObjectReferenceRequest) GetHealthChecks() []health.Criteria {
	// FIXME: propagate healthchecks to promotion request reference??
	return nil
}

// FIXME: should we use the full FreightReference??
func (r *promotionObjectReferenceRequest) GetFreightReference() *kargoapi.FreightReference {
	freight := r.Freight
	if freight == nil {
		return nil
	}
	return &kargoapi.FreightReference{
		Name:   r.Freight.Name,
		Origin: r.Freight.Origin,
	}
}
func (r *promotionObjectReferenceRequest) GetFreightCollection() *kargoapi.FreightCollection {
	return r.FreightCollection
}

func (r *promotionObjectReferenceRequest) SetFreightCollection(collection *kargoapi.FreightCollection) {
	r.FreightCollection = collection
}

// FIXME: do we have a message for promotoin request? Do we want to?
func (r *promotionObjectReferenceRequest) GetMessage() string {
	return r.GetPhase().String()
}
func (r *promotionObjectReferenceRequest) updateLastPromotion(stageStatus kargoapi.StageStatus) kargoapi.StageStatus {
	stageStatus.LastPromotionRequest = &r.PromotionRequestReference
	return stageStatus
}

func healthChecksToCriteria(healthChecks []kargoapi.HealthCheckStep) []health.Criteria {
	var criteria []health.Criteria
	for _, check := range healthChecks {
		criteria = append(criteria, health.Criteria{
			Kind:  check.Uses,
			Input: check.GetConfig(),
		})
	}
	return criteria
}

// PromotionObject implementations

type promotionObjectPromotion struct {
	kargoapi.Promotion
}

type promotionObjectRequest struct {
	kargoapi.PromotionRequest
}

// PromotionObject implementation for Promotion

func newPromotionObjectPromotion(promo *kargoapi.Promotion) *promotionObjectPromotion {
	if promo == nil {
		return nil
	}
	return &promotionObjectPromotion{
		Promotion: *promo,
	}
}

func (promo promotionObjectPromotion) GetPhase() PromotionObjectPhase {
	return promotionObjectPhasePromotion(promo.Status.Phase)
}
func (promo promotionObjectPromotion) GetFinishedAt() *metav1.Time {
	return promo.Status.FinishedAt
}
func (promo promotionObjectPromotion) GetFreightName() string {
	return promo.Spec.Freight
}
func (promo promotionObjectPromotion) GetArgoCDRefs() []api.ArgoCDAppRef {
	return api.ArgoCDAppRefsFromPromo(&promo.Promotion)
}
func (promo promotionObjectPromotion) GetStartedAt() *metav1.Time {
	return promo.Status.StartedAt
}
func (promo promotionObjectPromotion) ToLastPromotionReference() PromotionObjectReference {
	ref := kargoapi.PromotionReference{
		Name:       promo.Name,
		Status:     promo.Status.DeepCopy(),
		FinishedAt: promo.Status.FinishedAt,
	}
	if promo.Status.Freight != nil {
		ref.Freight = promo.Status.Freight.DeepCopy()
	}
	return &promotionObjectReferencePromotion{
		PromotionReference: ref,
	}
}

func (promo promotionObjectPromotion) updateCurrentPromotion(status kargoapi.StageStatus) kargoapi.StageStatus {
	status.CurrentPromotion = &kargoapi.PromotionReference{
		Name: promo.Name,
	}
	if freight := promo.Status.Freight; freight != nil {
		status.CurrentPromotion.Freight = freight.DeepCopy()
	}
	return status
}

// PromotionObject implementation for PromotionReference

func newPromotionObjectRequest(promo *kargoapi.PromotionRequest) *promotionObjectRequest {
	if promo == nil {
		return nil
	}
	return &promotionObjectRequest{
		PromotionRequest: *promo,
	}
}

func (promo promotionObjectRequest) GetPhase() PromotionObjectPhase {
	return promotionObjectPhaseRequest(promo.Status.Phase)
}
func (promo promotionObjectRequest) GetFinishedAt() *metav1.Time {
	return promo.Status.FinishedAt
}
func (promo promotionObjectRequest) GetFreightName() string {
	return promo.Spec.Freight
}
func (promo promotionObjectRequest) GetArgoCDRefs() []api.ArgoCDAppRef {
	// FIXME: extract target-aware argocd refs from promo request status
	return nil
}
func (promo promotionObjectRequest) GetStartedAt() *metav1.Time {
	return promo.Status.StartedAt
}
func (promo promotionObjectRequest) ToLastPromotionReference() PromotionObjectReference {
	ref := &kargoapi.PromotionRequestReference{
		Name:       promo.Name,
		Phase:      promo.Status.Phase,
		FinishedAt: promo.Status.FinishedAt,
	}

	// FIXME: we're going to populate a freight reference in promotion request status
	if promo.Status.Freight != nil {
		ref.Freight = promo.Status.Freight.DeepCopy()
	}
	if promo.Status.FreightCollection != nil {
		ref.FreightCollection = promo.Status.FreightCollection.DeepCopy()
	}
	return newPromotionObjectReferenceRequest(ref)
}

func (promo promotionObjectRequest) updateCurrentPromotion(status kargoapi.StageStatus) kargoapi.StageStatus {
	status.CurrentPromotionRequest = &kargoapi.PromotionRequestReference{
		Name: promo.Name,
	}
	if freight := promo.Status.Freight; freight != nil {
		status.CurrentPromotion.Freight = freight.DeepCopy()
	}
	return status
}

// Current and Last promotion access functions

func getLastPromoObject(stage *kargoapi.Stage) PromotionObjectReference {
	return getLastPromoObjectFromStatus(stage, stage.Status)
}

func getLastPromoObjectFromStatus(stage *kargoapi.Stage, status kargoapi.StageStatus) PromotionObjectReference {
	if api.IsTargetAware(stage) {
		return newPromotionObjectReferenceRequest(status.LastPromotionRequest)
	}
	return newPromotionObjectReferencePromotion(status.LastPromotion)
}

func getCurrentPromoObject(stage *kargoapi.Stage) PromotionObjectReference {
	return getCurrentPromoObjectFromStatus(stage, stage.Status)
}

func getCurrentPromoObjectFromStatus(stage *kargoapi.Stage, status kargoapi.StageStatus) PromotionObjectReference {
	if api.IsTargetAware(stage) {
		return newPromotionObjectReferenceRequest(status.CurrentPromotionRequest)
	}
	return newPromotionObjectReferencePromotion(status.CurrentPromotion)
}

func cleanCurrentPromotionObject(newStatus kargoapi.StageStatus, stage *kargoapi.Stage) kargoapi.StageStatus {
	if api.IsTargetAware(stage) {
		newStatus.CurrentPromotionRequest = nil
	} else {
		newStatus.CurrentPromotion = nil
	}
	return newStatus
}

// Listing functions

func (r *RegularStageReconciler) getPromotions(
	ctx context.Context,
	stageName, namespace, freightName string,
) ([]kargoapi.Promotion, error) {
	var selector client.MatchingFieldsSelector
	if freightName == "" {
		selector = client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByStageField,
				stageName,
			),
		}
	} else {
		selector = client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByStageAndFreightField,
				indexer.StageAndFreightKey(stageName, freightName),
			),
		}
	}
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(ctx, promotions, client.InNamespace(namespace), selector); err != nil {
		return nil, fmt.Errorf(
			"failed to list Promotions for Stage %q in namespace %q: %w",
			stageName, namespace, err,
		)
	}
	return promotions.Items, nil
}

func (r *RegularStageReconciler) getPromotionRequests(
	ctx context.Context,
	stageName, namespace, freightName string,
) ([]kargoapi.PromotionRequest, error) {
	var selector client.MatchingFieldsSelector
	if freightName == "" {
		selector = client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionRequestsByStageField,
				stageName,
			),
		}
	} else {
		selector = client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionRequestsByStageAndFreightField,
				indexer.StageAndFreightKey(stageName, freightName),
			),
		}
	}
	promotionRequests := &kargoapi.PromotionRequestList{}
	if err := r.client.List(ctx, promotionRequests, client.InNamespace(namespace), selector); err != nil {
		return nil, fmt.Errorf(
			"failed to list PromotionRequests for Stage %q in namespace %q: %w",
			stageName, namespace, err,
		)
	}
	return promotionRequests.Items, nil
}

// getPromotionObjectsByStageAndFreight returns a list of promotion objects for a stage
// If the stage is target aware - the objects are wrapping PromotionRequests, otherwise Promotions
func (r *RegularStageReconciler) getPromotionObjectsByStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) ([]PromotionObject, error) {
	return r.getPromotionObjectsByStageAndFreight(ctx, stage, "")
}

// getPromotionObjectsByStageAndFreight returns a list of promotion objects for a stage and freight
// If the stage is target aware - the objects are wrapping PromotionRequests, otherwise Promotions
// If freightName is empty - all objects for stage are returned
func (r *RegularStageReconciler) getPromotionObjectsByStageAndFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) ([]PromotionObject, error) {
	if api.IsTargetAware(stage) {
		requests, err := r.getPromotionRequests(ctx, stage.Name, stage.Namespace, freightName)
		if err != nil {
			return nil, err
		}
		return promotionRequestsToPromotionObjects(requests), nil
	}
	promotions, err := r.getPromotions(ctx, stage.Name, stage.Namespace, freightName)
	if err != nil {
		return nil, err
	}
	return promotionsToPromotionObjects(promotions), nil
}

func promotionsToPromotionObjects(promos []kargoapi.Promotion) []PromotionObject {
	promoObjs := make([]PromotionObject, len(promos))
	for i, promo := range promos {
		promoObjs[i] = newPromotionObjectPromotion(&promo)
	}
	return promoObjs
}

func promotionRequestsToPromotionObjects(requests []kargoapi.PromotionRequest) []PromotionObject {
	requestObjs := make([]PromotionObject, len(requests))
	for i, req := range requests {
		requestObjs[i] = newPromotionObjectRequest(&req)
	}
	return requestObjs
}
