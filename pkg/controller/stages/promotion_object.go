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

type promotionObject interface {
	// Getters:
	metav1.Object // Covers Name, CreationTimestamp and Annotations access
	getPhase() promotionObjectPhase
	getFinishedAt() *metav1.Time
	getFreightName() string
	getArgoCDRefs() []api.ArgoCDAppRef
	getStartedAt() *metav1.Time

	toLastPromotionReference() promotionObjectReference

	updateCurrentPromotion(kargoapi.StageStatus) kargoapi.StageStatus
}

type promotionObjectReference interface {
	getName() string
	getPhase() promotionObjectPhase

	getFinishedAt() *metav1.Time

	// For LastPromotion only:
	getHealthChecks() []health.Criteria
	// Maybe Freight reference??
	getFreightReference() *kargoapi.FreightReference
	getFreightCollection() *kargoapi.FreightCollection
	setFreightCollection(*kargoapi.FreightCollection)

	getMessage() string

	updateLastPromotion(kargoapi.StageStatus) kargoapi.StageStatus
}

type promotionObjectPhase interface {
	isSucceeded() bool
	isAborted() bool
	isRunning() bool
	isTerminal() bool
	string() string
}

// promotionObjectPhase implementations

type promotionObjectPhasePromotion struct {
	kargoapi.PromotionPhase
}
type promotionObjectPhaseRequest struct {
	kargoapi.PromotionRequestPhase
}

func (pphase promotionObjectPhasePromotion) isTerminal() bool {
	return pphase.IsTerminal()
}

func (pphase promotionObjectPhasePromotion) isSucceeded() bool {
	return pphase.PromotionPhase == kargoapi.PromotionPhaseSucceeded
}

func (pphase promotionObjectPhasePromotion) isAborted() bool {
	return pphase.PromotionPhase == kargoapi.PromotionPhaseAborted
}

func (pphase promotionObjectPhasePromotion) isRunning() bool {
	return pphase.PromotionPhase == kargoapi.PromotionPhaseRunning
}

func (pphase promotionObjectPhasePromotion) string() string {
	return string(pphase.PromotionPhase)
}

func (rphase promotionObjectPhaseRequest) isTerminal() bool {
	return rphase.IsTerminal()
}

func (rphase promotionObjectPhaseRequest) isSucceeded() bool {
	return rphase.PromotionRequestPhase == kargoapi.PromotionRequestPhaseSucceeded
}

func (rphase promotionObjectPhaseRequest) isAborted() bool {
	return rphase.PromotionRequestPhase == kargoapi.PromotionRequestPhaseAborted
}

func (rphase promotionObjectPhaseRequest) isRunning() bool {
	return rphase.PromotionRequestPhase == kargoapi.PromotionRequestPhaseRunning
}

func (rphase promotionObjectPhaseRequest) string() string {
	return string(rphase.PromotionRequestPhase)
}

// promotionObjectReference implementations

type promotionObjectReferencePromotion struct {
	kargoapi.PromotionReference
}
type promotionObjectReferenceRequest struct {
	kargoapi.PromotionRequestReference
}

// promotionObjectReference for PromotionReference

func newPromotionObjectReferencePromotion(ref *kargoapi.PromotionReference) promotionObjectReference {
	if ref == nil {
		return nil
	}
	return &promotionObjectReferencePromotion{
		PromotionReference: *ref,
	}
}

func (pref *promotionObjectReferencePromotion) getName() string {
	return pref.Name
}

func (pref *promotionObjectReferencePromotion) getPhase() promotionObjectPhase {
	if pref.Status == nil {
		return promotionObjectPhasePromotion{
			PromotionPhase: "",
		}
	}
	return promotionObjectPhasePromotion{
		PromotionPhase: pref.Status.Phase,
	}
}
func (pref *promotionObjectReferencePromotion) getFinishedAt() *metav1.Time {
	return pref.FinishedAt
}
func (pref *promotionObjectReferencePromotion) getHealthChecks() []health.Criteria {
	if pref.Status == nil {
		return nil
	}
	return healthChecksToCriteria(pref.Status.HealthChecks)
}
func (pref *promotionObjectReferencePromotion) getFreightReference() *kargoapi.FreightReference {
	return pref.Freight
}
func (pref *promotionObjectReferencePromotion) getFreightCollection() *kargoapi.FreightCollection {
	if pref.Status == nil {
		return nil
	}
	return pref.Status.FreightCollection
}

func (pref *promotionObjectReferencePromotion) setFreightCollection(collection *kargoapi.FreightCollection) {
	// We use this method under assumption that status is not nil
	if pref.Status == nil {
		return
	}
	pref.Status.FreightCollection = collection
}
func (pref *promotionObjectReferencePromotion) getMessage() string {
	if pref.Status == nil {
		return ""
	}
	return pref.Status.Message
}
func (pref *promotionObjectReferencePromotion) updateLastPromotion(
	stageStatus kargoapi.StageStatus,
) kargoapi.StageStatus {
	stageStatus.LastPromotion = &pref.PromotionReference
	return stageStatus
}

// promotionObjectReference for PromotionRequestReference

func newPromotionObjectReferenceRequest(ref *kargoapi.PromotionRequestReference) promotionObjectReference {
	if ref == nil {
		return nil
	}
	return &promotionObjectReferenceRequest{
		PromotionRequestReference: *ref,
	}
}

func (prref *promotionObjectReferenceRequest) getName() string {
	return prref.Name
}

func (prref *promotionObjectReferenceRequest) getPhase() promotionObjectPhase {
	return promotionObjectPhaseRequest{PromotionRequestPhase: prref.Phase}
}
func (prref *promotionObjectReferenceRequest) getFinishedAt() *metav1.Time {
	return prref.FinishedAt
}
func (prref *promotionObjectReferenceRequest) getHealthChecks() []health.Criteria {
	// FIXME: propagate healthchecks to promotion request reference or resolve them differently?
	// FIXME: we might want to extract healthchecks from targets or report specifically
	// that targets will do healthchecking
	return nil
}

func (prref *promotionObjectReferenceRequest) getFreightReference() *kargoapi.FreightReference {
	return prref.Freight
}
func (prref *promotionObjectReferenceRequest) getFreightCollection() *kargoapi.FreightCollection {
	return prref.FreightCollection
}

func (prref *promotionObjectReferenceRequest) setFreightCollection(collection *kargoapi.FreightCollection) {
	prref.FreightCollection = collection
}

func (prref *promotionObjectReferenceRequest) getMessage() string {
	return prref.getPhase().string()
}
func (prref *promotionObjectReferenceRequest) updateLastPromotion(
	stageStatus kargoapi.StageStatus,
) kargoapi.StageStatus {
	stageStatus.LastPromotionRequest = &prref.PromotionRequestReference
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

// promotionObject implementations

type promotionObjectPromotion struct {
	kargoapi.Promotion
}

type promotionObjectRequest struct {
	kargoapi.PromotionRequest
}

// promotionObject implementation for Promotion

func newPromotionObjectPromotion(promo *kargoapi.Promotion) *promotionObjectPromotion {
	if promo == nil {
		return nil
	}
	return &promotionObjectPromotion{
		Promotion: *promo,
	}
}

func (promo promotionObjectPromotion) getPhase() promotionObjectPhase {
	return promotionObjectPhasePromotion{PromotionPhase: promo.Status.Phase}
}
func (promo promotionObjectPromotion) getFinishedAt() *metav1.Time {
	return promo.Status.FinishedAt
}
func (promo promotionObjectPromotion) getFreightName() string {
	return promo.Spec.Freight
}
func (promo promotionObjectPromotion) getArgoCDRefs() []api.ArgoCDAppRef {
	return api.ArgoCDAppRefsFromPromo(&promo.Promotion)
}
func (promo promotionObjectPromotion) getStartedAt() *metav1.Time {
	return promo.Status.StartedAt
}
func (promo promotionObjectPromotion) toLastPromotionReference() promotionObjectReference {
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

// promotionObject implementation for PromotionReference

func newPromotionObjectRequest(promo *kargoapi.PromotionRequest) *promotionObjectRequest {
	if promo == nil {
		return nil
	}
	return &promotionObjectRequest{
		PromotionRequest: *promo,
	}
}

func (req promotionObjectRequest) getPhase() promotionObjectPhase {
	return promotionObjectPhaseRequest{PromotionRequestPhase: req.Status.Phase}
}
func (req promotionObjectRequest) getFinishedAt() *metav1.Time {
	return req.Status.FinishedAt
}
func (req promotionObjectRequest) getFreightName() string {
	return req.Spec.Freight
}
func (req promotionObjectRequest) getArgoCDRefs() []api.ArgoCDAppRef {
	// FIXME: extract target-aware argocd refs from promo request status
	return nil
}
func (req promotionObjectRequest) getStartedAt() *metav1.Time {
	return req.Status.StartedAt
}
func (req promotionObjectRequest) toLastPromotionReference() promotionObjectReference {
	ref := &kargoapi.PromotionRequestReference{
		Name:       req.Name,
		Phase:      req.Status.Phase,
		FinishedAt: req.Status.FinishedAt,
	}

	if req.Status.Freight != nil {
		ref.Freight = req.Status.Freight.DeepCopy()
	}
	if req.Status.FreightCollection != nil {
		ref.FreightCollection = req.Status.FreightCollection.DeepCopy()
	}
	return newPromotionObjectReferenceRequest(ref)
}

func (req promotionObjectRequest) updateCurrentPromotion(status kargoapi.StageStatus) kargoapi.StageStatus {
	status.CurrentPromotionRequest = &kargoapi.PromotionRequestReference{
		Name: req.Name,
	}
	if freight := req.Status.Freight; freight != nil {
		status.CurrentPromotionRequest.Freight = freight.DeepCopy()
	}
	return status
}

// Current and Last promotion access functions

func getLastPromoObject(stage *kargoapi.Stage) promotionObjectReference {
	return getLastPromoObjectFromStatus(stage, stage.Status)
}

func getLastPromoObjectFromStatus(stage *kargoapi.Stage, status kargoapi.StageStatus) promotionObjectReference {
	if api.IsTargetAware(stage) {
		return newPromotionObjectReferenceRequest(status.LastPromotionRequest)
	}
	return newPromotionObjectReferencePromotion(status.LastPromotion)
}

func getCurrentPromoObject(stage *kargoapi.Stage) promotionObjectReference {
	return getCurrentPromoObjectFromStatus(stage, stage.Status)
}

func getCurrentPromoObjectFromStatus(stage *kargoapi.Stage, status kargoapi.StageStatus) promotionObjectReference {
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
) ([]promotionObject, error) {
	return r.getPromotionObjectsByStageAndFreight(ctx, stage, "")
}

// getPromotionObjectsByStageAndFreight returns a list of promotion objects for a stage and freight
// If the stage is target aware - the objects are wrapping PromotionRequests, otherwise Promotions
// If freightName is empty - all objects for stage are returned
func (r *RegularStageReconciler) getPromotionObjectsByStageAndFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) ([]promotionObject, error) {
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

func promotionsToPromotionObjects(promos []kargoapi.Promotion) []promotionObject {
	promoObjs := make([]promotionObject, len(promos))
	for i, promo := range promos {
		promoObjs[i] = newPromotionObjectPromotion(&promo)
	}
	return promoObjs
}

func promotionRequestsToPromotionObjects(requests []kargoapi.PromotionRequest) []promotionObject {
	requestObjs := make([]promotionObject, len(requests))
	for i, req := range requests {
		requestObjs[i] = newPromotionObjectRequest(&req)
	}
	return requestObjs
}
