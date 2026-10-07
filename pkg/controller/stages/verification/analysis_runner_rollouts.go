package verification

import (
	"context"
	"fmt"
	"slices"

	gocache "github.com/patrickmn/go-cache"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	exprfn "github.com/akuity/kargo/pkg/expressions/function"
	"github.com/akuity/kargo/pkg/kubernetes"
	"github.com/akuity/kargo/pkg/rollouts"
)

type analysisRunnerRollouts struct {
	rolloutsControllerInstanceID string
	backoffCfg                   wait.Backoff
	client                       client.Client
	credentialsDB                credentials.Database
}

func rolloutsRunToVerifierRun(ar rolloutsapi.AnalysisRun) AnalysisRun {
	return AnalysisRun{
		Name:              ar.Name,
		Namespace:         ar.Namespace,
		CreationTimestamp: ar.CreationTimestamp,
		Status:            rolloutsStatusToVerifierStatus(ar.Status),
		GVK:               kargoapi.AnalysisRunGVKRun,
	}
}

func rolloutsStatusToVerifierStatus(arStatus rolloutsapi.AnalysisRunStatus) AnalysisRunStatus {
	return AnalysisRunStatus{
		CompletedAt: arStatus.CompletedAt,
		Phase:       string(arStatus.Phase),
		Message:     arStatus.Message,
	}
}

func NewAnalysisRunnerRollouts(
	rolloutsControllerInstanceID string,
	backoffCfg wait.Backoff,
	cl client.Client,
	credentialsDB credentials.Database,
) AnalysisRunner {
	return &analysisRunnerRollouts{
		rolloutsControllerInstanceID: rolloutsControllerInstanceID,
		backoffCfg:                   backoffCfg,
		client:                       cl,
		credentialsDB:                credentialsDB,
	}
}

func (runner *analysisRunnerRollouts) GetAnalysisRunStatus(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (*AnalysisRunStatus, error) {
	// TODO(hidde): This retry logic has been put in place because we have
	// observed the cache not being up-to-date with the API server in some
	// edge case scenarios. While this is not a long-term solution, it cures
	// the symptoms for now. We should investigate the root cause of this
	// issue and remove this retry logic when the root cause has been resolved.
	// FIXME: support different types here
	ar := rolloutsapi.AnalysisRun{}
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

	return new(rolloutsStatusToVerifierStatus(ar.Status)), nil
}

func (runner *analysisRunnerRollouts) AbortAnalysisRun(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) error {
	// Patch the AnalysisRun to request the abort.
	ar := &rolloutsapi.AnalysisRun{
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

func (runner *analysisRunnerRollouts) GetAnalysisRunPromotionName(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (string, error) {
	ar := &rolloutsapi.AnalysisRun{}
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

func (runner *analysisRunnerRollouts) FindExistingAnalysisRun(
	ctx context.Context,
	stage types.NamespacedName,
	freightColID string,
) (*AnalysisRun, error) {
	// FIXME: implement it for other run types
	analysisRuns := &rolloutsapi.AnalysisRunList{}

	reqNotTarget, errLabelNoTarget := labels.NewRequirement(
		kargoapi.LabelKeyAnalysisRunTarget,
		selection.DoesNotExist,
		[]string{})

	if errLabelNoTarget != nil {
		return nil, fmt.Errorf("invalid label selectors %w", errLabelNoTarget)
	}

	if err := runner.client.List(
		ctx,
		analysisRuns,
		client.InNamespace(stage.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(map[string]string{
				kargoapi.LabelKeyStage:             kubernetes.ShortenLabelValue(stage.Name),
				kargoapi.LabelKeyFreightCollection: freightColID,
			}).Add(*reqNotTarget),
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
	mostRecent := slices.MinFunc(analysisRuns.Items, func(lhs, rhs rolloutsapi.AnalysisRun) int {
		return rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
	})
	return new(rolloutsRunToVerifierRun(mostRecent)), nil
}

func (runner *analysisRunnerRollouts) CreateAnalysisRun(
	ctx context.Context,
	stage kargoapi.Stage,
	freightCollection kargoapi.FreightCollection,
	lastPromoName string,
) (*AnalysisRun, string, error) {
	shortStageName := kubernetes.ShortenLabelValue(stage.Name)
	builder := rollouts.NewAnalysisRunBuilder(runner.client, rollouts.Config{
		ControllerInstanceID: runner.rolloutsControllerInstanceID,
	})
	builderOpts := []rollouts.AnalysisRunOption{
		rollouts.WithNamePrefix(stage.Name),
		rollouts.WithNameSuffix(freightCollection.ID),
		rollouts.WithExtraLabels(map[string]string{
			kargoapi.LabelKeyStage:             shortStageName,
			kargoapi.LabelKeyFreightCollection: freightCollection.ID,
		}),
		rollouts.WithArgumentEvaluationConfig{
			Env: map[string]any{
				"ctx": map[string]any{
					"project": stage.Namespace,
					"stage":   stage.Name,
				},
			},
			Options: slices.Concat(
				exprfn.DataOperations(
					ctx,
					runner.client,
					runner.credentialsDB,
					gocache.New(gocache.NoExpiration, gocache.NoExpiration),
					stage.Namespace,
				),
				exprfn.FreightOperations(
					ctx,
					runner.client,
					stage.Namespace,
					stage.Spec.RequestedFreight,
					freightCollection.References(),
				),
				exprfn.UtilityOperations(),
			),
			Vars: stage.Spec.Vars,
		},
	}

	if stage.Name != shortStageName {
		builderOpts = append(builderOpts, rollouts.WithExtraAnnotations(map[string]string{
			kargoapi.AnnotationKeyStage: stage.Name,
		}))
	}

	for _, freightRef := range freightCollection.Freight {
		builderOpts = append(builderOpts, rollouts.WithOwner{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Freight",
			Reference:  types.NamespacedName{Namespace: stage.Namespace, Name: freightRef.Name},
		})
	}

	if lastPromoName != "" {
		builderOpts = append(builderOpts, rollouts.WithExtraAnnotations{
			kargoapi.AnnotationKeyPromotion: lastPromoName,
		})
	}

	ar, err := builder.Build(ctx, stage.Namespace, stage.Spec.Verification, builderOpts...)
	if err != nil {
		return nil, fmt.Errorf(
			"error building AnalysisRun for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name,
			freightCollection.ID,
			stage.Namespace,
			err,
		).Error(), nil
	}
	if err = runner.client.Create(ctx, ar); err != nil {
		return nil, fmt.Errorf(
			"error creating AnalysisRun %q in namespace %q: %w",
			ar.Name,
			ar.Namespace,
			err,
		).Error(), err
	}
	return new(rolloutsRunToVerifierRun(*ar)), "", nil
}
