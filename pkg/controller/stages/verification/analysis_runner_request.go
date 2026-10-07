package verification

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/oklog/ulid/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/kubernetes"
)

type analysisRunnerRequest struct {
	rolloutsControllerInstanceID string
	backoffCfg                   wait.Backoff
	client                       client.Client
	credentialsDB                credentials.Database
}

func requestRunToVerifierRun(ar kargoapi.AnalysisRunRequest) AnalysisRun {
	return AnalysisRun{
		Name:              ar.Name,
		Namespace:         ar.Namespace,
		CreationTimestamp: ar.CreationTimestamp,
		Status:            requestStatusToVerifierStatus(ar.Status),
		GVK:               kargoapi.AnalysisRunGVKRequest,
	}
}

func requestStatusToVerifierStatus(arStatus kargoapi.AnalysisRunRequestStatus) AnalysisRunStatus {
	return AnalysisRunStatus{
		CompletedAt: arStatus.CompletedAt,
		Phase:       string(arStatus.Phase),
		Message:     arStatus.Message,
	}
}

func NewAnalysisRunnerRequest(
	rolloutsControllerInstanceID string,
	backoffCfg wait.Backoff,
	cl client.Client,
	credentialsDB credentials.Database,
) AnalysisRunner {
	return &analysisRunnerRequest{
		rolloutsControllerInstanceID: rolloutsControllerInstanceID,
		backoffCfg:                   backoffCfg,
		client:                       cl,
		credentialsDB:                credentialsDB,
	}
}

func (runner analysisRunnerRequest) GetAnalysisRunStatus(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (*AnalysisRunStatus, error) {
	// TODO(hidde): This retry logic has been put in place because we have
	// observed the cache not being up-to-date with the API server in some
	// edge case scenarios. While this is not a long-term solution, it cures
	// the symptoms for now. We should investigate the root cause of this
	// issue and remove this retry logic when the root cause has been resolved.
	// FIXME: support different types here
	ar := kargoapi.AnalysisRunRequest{}
	err := retry.OnError(runner.backoffCfg, func(err error) bool {
		return apierrors.IsNotFound(err)
	}, func() error {
		return runner.client.Get(ctx, types.NamespacedName{
			Namespace: runRef.Namespace,
			Name:      runRef.Name,
		}, &ar)
	})

	if err != nil {
		return nil, err
	}

	return new(requestStatusToVerifierStatus(ar.Status)), nil
}

func (runner analysisRunnerRequest) AbortAnalysisRun(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) error {
	// Patch the AnalysisRunRequest to request the abort.
	ar := &kargoapi.AnalysisRunRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: runRef.Namespace,
			Name:      runRef.Name,
		},
	}
	return runner.client.Patch(
		ctx,
		ar,
		client.RawPatch(types.MergePatchType, []byte(`{"spec":{"terminate":true}}`)),
	)
}

func (runner analysisRunnerRequest) GetAnalysisRunPromotionName(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (string, error) {
	ar := &kargoapi.AnalysisRunRequest{}
	if err := runner.client.Get(ctx, types.NamespacedName{
		Namespace: runRef.Namespace,
		Name:      runRef.Name,
	}, ar); err != nil {
		return "", err
	}
	// AnalysisRun that triggered by a Promotion contains the Promotion name
	if promoName, ok := ar.Annotations[kargoapi.AnnotationKeyPromotion]; ok {
		return promoName, nil
	}
	return "", nil
}

func (runner analysisRunnerRequest) FindExistingAnalysisRun(
	ctx context.Context,
	stage types.NamespacedName,
	freightColID string,
) (*AnalysisRun, error) {
	analysisRuns := &kargoapi.AnalysisRunRequestList{}
	if err := runner.client.List(
		ctx,
		analysisRuns,
		client.InNamespace(stage.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(map[string]string{
				kargoapi.LabelKeyStage:             kubernetes.ShortenLabelValue(stage.Name),
				kargoapi.LabelKeyFreightCollection: freightColID,
			}),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing AnalysisRuns for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name, freightColID, stage.Namespace, err,
		)
	}

	if len(analysisRuns.Items) == 0 {
		return nil, nil
	}

	// Sort the AnalysisRuns by creation timestamp, so that the most recent
	// one is first.
	slices.SortFunc(analysisRuns.Items, func(lhs, rhs kargoapi.AnalysisRunRequest) int {
		return rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
	})
	return new(requestRunToVerifierRun(analysisRuns.Items[0])), nil
}

func (runner analysisRunnerRequest) CreateAnalysisRun(
	ctx context.Context,
	stage kargoapi.Stage,
	freightCollection kargoapi.FreightCollection,
	lastPromoName string,
) (*AnalysisRun, string, error) {
	// Verification should be set by now
	if stage.Spec.Verification == nil {
		return nil, "", fmt.Errorf("verification should be set in the Stage spec")
	}

	// Double-check: analysis run requests are only supported for target-aware stages
	if !stage.IsTargetAware() {
		return nil, "", fmt.Errorf("analysisrunrequests are only supported for target-aware stages")
	}

	targets, err := api.ListTargetsForStage(ctx, runner.client, &stage)
	if err != nil {
		return nil, "", fmt.Errorf(
			"error resolving Targets governed by Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}

	specTargets := make([]kargoapi.PromotionRequestTarget, len(targets))
	for i, target := range targets {
		specTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
	}

	// FreightCollection without verification info
	// TODO: look into creating a type without verification info to use in analysisrunrequests and promotions
	requestFreightCollection := kargoapi.FreightCollection{}
	requestFreightCollection.ID = freightCollection.ID
	requestFreightCollection.Freight = freightCollection.Freight

	ownerRefs := []metav1.OwnerReference{}

	for _, freightRef := range freightCollection.Freight {
		freight, freightErr := api.GetFreight(ctx, runner.client, types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      freightRef.Name,
		})
		if freightErr != nil {
			return nil, "", fmt.Errorf("cannot read freight for verification %q: %w", freightRef.Name, freightErr)
		}
		if freight != nil {
			owner := metav1.OwnerReference{
				APIVersion: kargoapi.GroupVersion.String(),
				Kind:       "Freight",
				Name:       freight.Name,
				UID:        freight.UID,
			}

			ownerRefs = append(ownerRefs, owner)
		}
	}

	shortStageName := kubernetes.ShortenLabelValue(stage.Name)
	arrLabels := map[string]string{
		kargoapi.LabelKeyStage:             shortStageName,
		kargoapi.LabelKeyFreightCollection: freightCollection.ID,
	}
	if shard, ok := stage.Labels[kargoapi.LabelKeyShard]; ok {
		arrLabels[kargoapi.LabelKeyShard] = shard
	}

	annotations := map[string]string{}
	if shortStageName != stage.Name {
		annotations[kargoapi.AnnotationKeyStage] = stage.Name
	}

	updateStrategy := kargoapi.TargetUpdateStrategy{}
	if stage.Spec.Targets.Verification != nil {
		updateStrategy = stage.Spec.Targets.Verification.UpdateStrategy
	}

	request := &kargoapi.AnalysisRunRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRequestName(stage.Name, freightCollection.ID),
			Namespace:       stage.Namespace,
			Labels:          arrLabels,
			Annotations:     annotations,
			OwnerReferences: ownerRefs,
		},
		Spec: kargoapi.AnalysisRunRequestSpec{
			StageName:            stage.Name,
			PromotionName:        lastPromoName,
			RequestedFreight:     stage.Spec.RequestedFreight,
			StageVars:            stage.Spec.Vars,
			FreightCollection:    requestFreightCollection,
			VerificationTemplate: *stage.Spec.Verification.DeepCopy(),
			// FIXME: We might want to add an extra check that these targets have a freight we're verifying
			Targets:        specTargets,
			UpdateStrategy: updateStrategy,
		},
	}

	if err = runner.client.Create(ctx, request); err != nil {
		return nil, fmt.Errorf(
			"error creating AnalysisRunRequest %q in namespace %q: %w",
			request.Name,
			request.Namespace,
			err,
		).Error(), err
	}
	return new(requestRunToVerifierRun(*request)), "", nil
}

const (
	ulidLength          = 26
	maxNameSuffixLength = 7
	maxNamePrefixLength = 253 - (1 + ulidLength) - (1 + maxNameSuffixLength)
)

func generateRequestName(prefix, suffix string) string {
	var parts []string
	if len(prefix) > maxNamePrefixLength {
		prefix = prefix[0:maxNamePrefixLength]
	}
	if prefix != "" {
		parts = append(parts, prefix)
	}

	parts = append(parts, ulid.Make().String())

	if len(suffix) > maxNameSuffixLength {
		suffix = suffix[0:maxNameSuffixLength]
	}
	if suffix != "" {
		parts = append(parts, suffix)
	}

	return strings.ToLower(strings.Join(parts, "."))
}
