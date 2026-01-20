package warehouses

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/kelseyhightower/envconfig"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/expressions/function"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	intpredicate "github.com/akuity/kargo/pkg/predicate"
	"github.com/akuity/kargo/pkg/subscription"
)

type ReconcilerConfig struct {
	IsDefaultController       bool          `envconfig:"IS_DEFAULT_CONTROLLER"`
	ShardName                 string        `envconfig:"SHARD_NAME"`
	MaxConcurrentReconciles   int           `envconfig:"MAX_CONCURRENT_WAREHOUSE_RECONCILES" default:"4"`
	MinReconciliationInterval time.Duration `envconfig:"MIN_WAREHOUSE_RECONCILIATION_INTERVAL"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Warehouse resources.
type reconciler struct {
	client             client.Client
	credentialsDB      credentials.Database
	subscriberRegistry subscription.SubscriberRegistry
	cfg                ReconcilerConfig
	shardPredicate     controller.ResponsibleFor[kargoapi.Warehouse]

	// The following behaviors are overridable for testing purposes:

	discoverArtifactsFn func(
		ctx context.Context,
		project string,
		subs []kargoapi.RepoSubscription,
	) (*kargoapi.DiscoveredArtifacts, error)

	buildFreightFromLatestArtifactsFn func(string, *kargoapi.DiscoveredArtifacts) (*kargoapi.Freight, error)

	createFreightFn func(context.Context, client.Object, ...client.CreateOption) error

	patchStatusFn func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error
}

// SetupReconcilerWithManager initializes a reconciler for Warehouse resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	credentialsDB credentials.Database,
	subscriberRegistry subscription.SubscriberRegistry,
	cfg ReconcilerConfig,
) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
		WithEventFilter(controller.ResponsibleFor[client.Object]{
			IsDefaultController: cfg.IsDefaultController,
			ShardName:           cfg.ShardName,
		}).
		WithEventFilter(intpredicate.IgnoreDelete[client.Object]{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				kargo.RefreshRequested{},
			),
		).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Complete(newReconciler(
			mgr.GetClient(),
			credentialsDB,
			subscriberRegistry,
			cfg,
		)); err != nil {
		return fmt.Errorf("error building Warehouse reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized Warehouse reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)

	return nil
}

func newReconciler(
	kubeClient client.Client,
	credentialsDB credentials.Database,
	subscriberRegistry subscription.SubscriberRegistry,
	cfg ReconcilerConfig,
) *reconciler {
	r := &reconciler{
		client:             kubeClient,
		credentialsDB:      credentialsDB,
		subscriberRegistry: subscriberRegistry,
		cfg:                cfg,
		shardPredicate: controller.ResponsibleFor[kargoapi.Warehouse]{
			IsDefaultController: cfg.IsDefaultController,
			ShardName:           cfg.ShardName,
		},
		createFreightFn: kubeClient.Create,
	}
	r.discoverArtifactsFn = r.discoverArtifacts
	r.buildFreightFromLatestArtifactsFn = r.buildFreightFromLatestArtifacts
	r.patchStatusFn = r.patchStatus
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx)

	logger = logger.WithValues(
		"namespace", req.Namespace,
		"warehouse", req.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Warehouse")

	// Find the Warehouse
	warehouse, err := api.GetWarehouse(ctx, r.client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if warehouse == nil {
		// Ignore if not found. This can happen if the Warehouse was deleted after
		// the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	if !r.shardPredicate.IsResponsible(warehouse) {
		logger.Debug("ignoring Warehouse because it is not assigned to this shard")
		return ctrl.Result{}, nil
	}

	newStatus, err := r.syncWarehouse(ctx, warehouse)
	if err != nil {
		logger.Error(err, "error syncing Warehouse")
	}

	updateErr := r.patchStatusFn(
		ctx,
		warehouse,
		func(status *kargoapi.WarehouseStatus) {
			*status = newStatus
		},
	)
	if updateErr != nil {
		logger.Error(updateErr, "error updating Warehouse status")
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}
	logger.Debug("done reconciling Warehouse")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	if err != nil {
		return ctrl.Result{}, err
	}

	// Everything succeeded, look for new changes on the defined interval.
	return ctrl.Result{
		RequeueAfter: warehouse.GetInterval(r.cfg.MinReconciliationInterval),
	}, nil
}

func (r *reconciler) syncWarehouse(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (kargoapi.WarehouseStatus, error) {
	logger := logging.LoggerFromContext(ctx)

	status := *warehouse.Status.DeepCopy()

	// Record the current refresh token as having been handled.
	if token, ok := api.RefreshAnnotationValue(warehouse.GetAnnotations()); ok {
		status.LastHandledRefresh = token
	}

	// Discover the latest artifacts.
	if shouldDiscoverArtifacts(warehouse, status.LastHandledRefresh) {
		// As this is a long-running operation, we need to update the status
		// conditions to reflect that we are currently reconciling.
		conditions.Set(
			&status,
			&metav1.Condition{
				Type:   kargoapi.ConditionTypeReconciling,
				Status: metav1.ConditionTrue,
				Reason: "ScheduledDiscovery",
				Message: fmt.Sprintf(
					"Discovering artifacts for %d subscriptions",
					len(warehouse.Spec.InternalSubscriptions),
				),
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "DiscoveryInProgress",
				Message:            "Waiting for discovery to complete",
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionUnknown,
				Reason:             "Pending",
				Message:            "Health status cannot be determined until artifact discovery is finished",
				ObservedGeneration: warehouse.GetGeneration(),
			},
		)
		conditions.Delete(&status, kargoapi.ConditionTypeFreightCreationCriteriaSatisfied)
		conditions.Delete(&status, kargoapi.ConditionTypeFreightCreated)
		if err := r.patchStatusFn(ctx, warehouse, func(s *kargoapi.WarehouseStatus) {
			s.SetConditions(status.GetConditions())
		}); err != nil {
			logger.Error(err, "error updating Warehouse status")
		}

		// Discover the latest artifacts.
		discoveredArtifacts, err := r.discoverArtifactsFn(
			ctx,
			warehouse.Namespace,
			warehouse.Spec.InternalSubscriptions,
		)
		if err != nil {
			// Mark the Warehouse as unhealthy and not ready if we failed to
			// discover artifacts.
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "DiscoveryFailed",
					Message:            fmt.Sprintf("Unable to discover artifacts: %s", err.Error()),
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "DiscoveryFailure",
					Message:            fmt.Sprintf("Artifact discovery failed: %s", err.Error()),
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return status, fmt.Errorf("error discovering artifacts: %w", err)
		}
		logger.Debug("discovered latest artifacts")

		// Update the status with the discovered artifacts, and mark the
		// Warehouse as healthy.
		status.DiscoveredArtifacts = discoveredArtifacts
	}

	// At this point, we have successfully discovered the latest artifacts
	// for the Warehouse using the current subscriptions.
	status.ObservedGeneration = warehouse.GetGeneration()

	// Validate the discovered artifacts.
	if !validateDiscoveredArtifacts(warehouse, &status) {
		// If validation returned false, the Healthy and Ready conditions will
		// already have been updated appropriately.

		// Remove the reconciling condition and return early if the validation
		// failed. We do not return an error here, to prevent a requeue loop
		// which would cause unnecessary pressure on the upstream sources.
		conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
		return status, nil
	}

	// Automatically create a Freight from the latest discovered artifacts
	// if the Warehouse is configured to do so.
	if pol := warehouse.Spec.FreightCreationPolicy; pol == kargoapi.FreightCreationPolicyAutomatic || pol == "" {
		criteriaSatisfied, err := freightCreationCriteriaSatisfied(ctx,
			warehouse.Spec.FreightCreationCriteria,
			status.DiscoveredArtifacts,
		)
		if err != nil {
			logger.Error(err, "error evaluating freight creation criteria")
			msg := fmt.Sprintf(
				"Evaluation of Freight creation criteria failed: %s", err.Error(),
			)
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "CriteriaEvaluationFailed",
					Message:            msg,
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "CriteriaEvaluationFailed",
					Message:            msg,
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
			// return a nil error to avoid a requeue loop since subsequent
			// retries are not going to make the expression any more valid.
			return status, nil
		}
		if !criteriaSatisfied {
			logger.Debug("freight creation criteria not satisfied; skipping freight creation")
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeFreightCreationCriteriaSatisfied,
					Status:             metav1.ConditionFalse,
					Reason:             "CriteriaNotSatisfied",
					Message:            "Freight creation criteria were not satisfied",
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
		} else {
			logger.Debug("freight creation criteria satisfied")
			// Mark the Warehouse as reconciling while we create the Freight.
			//
			// As this should be a quick operation, we do not issue an immediate
			// patch to the Warehouse status. However, we do update the conditions
			// to reflect current state to ensure they're correct if we run into
			// an error, after which the status will be patched.
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReconciling,
					Status:             metav1.ConditionTrue,
					Reason:             "FreightCreationInProgress",
					Message:            "Creating Freight from latest artifacts",
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "AwaitingFreightCreation",
					Message:            "Freight creation from latest artifacts is in progress",
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionUnknown,
					Reason:             "Pending",
					Message:            "Health status cannot be determined until Freight creation is finished",
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeFreightCreationCriteriaSatisfied,
					Status:             metav1.ConditionTrue,
					Reason:             "CriteriaSatisfied",
					Message:            "Freight creation criteria satisfied",
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)

			// Build Freight from the latest discovered artifacts.
			freight, err := r.buildFreightFromLatestArtifactsFn(warehouse.Namespace, status.DiscoveredArtifacts)
			if err != nil {
				// Make the error visible in the status and mark the Warehouse as
				// not ready.
				msg := fmt.Sprintf(
					"Error building Freight from latest artifacts: %s",
					err.Error(),
				)
				conditions.Set(
					&status,
					&metav1.Condition{
						Type:               kargoapi.ConditionTypeHealthy,
						Status:             metav1.ConditionFalse,
						Reason:             "FreightBuildFailure",
						Message:            msg,
						ObservedGeneration: warehouse.GetGeneration(),
					},
					&metav1.Condition{
						Type:               kargoapi.ConditionTypeReady,
						Status:             metav1.ConditionFalse,
						Reason:             "FreightBuildFailure",
						Message:            msg,
						ObservedGeneration: warehouse.GetGeneration(),
					},
				)
				return status, fmt.Errorf("failed to build Freight from latest artifacts: %w", err)
			}

			freight.Origin = kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: warehouse.Name,
			}

			// Attempt to create the Freight.
			if err = r.createFreightFn(ctx, freight); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					// Make the error visible in the status and mark the Warehouse as
					// not ready.
					msg := fmt.Sprintf(
						"Error creating Freight %q in namespace %q: %s",
						freight.Name,
						freight.Namespace,
						err.Error(),
					)
					conditions.Set(
						&status,
						&metav1.Condition{
							Type:               kargoapi.ConditionTypeHealthy,
							Status:             metav1.ConditionFalse,
							Reason:             "FreightBuildFailure",
							Message:            msg,
							ObservedGeneration: warehouse.GetGeneration(),
						},
						&metav1.Condition{
							Type:               kargoapi.ConditionTypeReady,
							Status:             metav1.ConditionFalse,
							Reason:             "FreightCreationFailure",
							Message:            msg,
							ObservedGeneration: warehouse.GetGeneration(),
						},
					)
					return status, fmt.Errorf(
						"error creating Freight %q in namespace %q: %w",
						freight.Name,
						freight.Namespace,
						err,
					)
				}
				conditions.Set(
					&status,
					&metav1.Condition{
						Type:               kargoapi.ConditionTypeFreightCreated,
						Status:             metav1.ConditionFalse,
						Reason:             "AlreadyExists",
						Message:            "Freight composed of the newest artifacts already exists",
						ObservedGeneration: warehouse.GetGeneration(),
					},
				)
			} else {
				logger.Debug(
					"created Freight",
					"freight", freight.Name,
					"namespace", freight.Namespace,
				)
				conditions.Set(
					&status,
					&metav1.Condition{
						Type:               kargoapi.ConditionTypeFreightCreated,
						Status:             metav1.ConditionTrue,
						Reason:             "NewFreight",
						Message:            "No Freight composed of the newest artifacts already existed",
						ObservedGeneration: warehouse.GetGeneration(),
					},
				)
			}

			status.LastFreightID = freight.Name
		}
	}

	// Make all conditions reflect success
	msg := fmt.Sprintf(
		"Successfully discovered artifacts from %d subscriptions",
		len(warehouse.Spec.InternalSubscriptions),
	)
	conditions.Set(
		&status,
		&metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionTrue,
			Reason:             "ReconciliationSucceeded",
			ObservedGeneration: warehouse.GetGeneration(),
		},
	)
	conditions.Set(
		&status,
		&metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             "ArtifactsDiscovered",
			Message:            msg,
			ObservedGeneration: warehouse.GetGeneration(),
		},
	)
	conditions.Delete(&status, kargoapi.ConditionTypeReconciling)

	return status, nil
}

func (r *reconciler) discoverArtifacts(
	ctx context.Context,
	project string,
	subs []kargoapi.RepoSubscription,
) (*kargoapi.DiscoveredArtifacts, error) {
	discovered := &kargoapi.DiscoveredArtifacts{
		Charts:  []kargoapi.ChartDiscoveryResult{},
		Git:     []kargoapi.GitDiscoveryResult{},
		Images:  []kargoapi.ImageDiscoveryResult{},
		Results: []kargoapi.DiscoveryResult{},
	}
	for _, sub := range subs {
		subReg, err := r.subscriberRegistry.Get(ctx, sub)
		if err != nil {
			return nil,
				fmt.Errorf("error finding subscriber for subscription: %w", err)
		}
		// The registration's value is a factory function
		subscriber, err := subReg.Value(ctx, r.credentialsDB)
		if err != nil {
			return nil, fmt.Errorf("error instantiating subscriber: %w", err)
		}
		res, err := subscriber.DiscoverArtifacts(ctx, project, sub)
		if err != nil {
			return nil, fmt.Errorf("error discovering artifacts: %w", err)
		}
		switch typedRes := res.(type) {
		case kargoapi.ChartDiscoveryResult:
			discovered.Charts = append(discovered.Charts, typedRes)
		case kargoapi.GitDiscoveryResult:
			discovered.Git = append(discovered.Git, typedRes)
		case kargoapi.ImageDiscoveryResult:
			discovered.Images = append(discovered.Images, typedRes)
		case kargoapi.DiscoveryResult:
			discovered.Results = append(discovered.Results, typedRes)
		default:
			return nil, fmt.Errorf(
				"subscriber returned unrecognized result type %T", typedRes,
			)
		}
	}
	discovered.DiscoveredAt = metav1.Now()
	return discovered, nil
}

func (r *reconciler) buildFreightFromLatestArtifacts(
	namespace string,
	artifacts *kargoapi.DiscoveredArtifacts,
) (*kargoapi.Freight, error) {
	if artifacts == nil {
		return nil, fmt.Errorf("no artifacts discovered")
	}

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

	for _, result := range artifacts.Git {
		if len(result.Commits) == 0 {
			return nil, fmt.Errorf("no commits discovered for repository %q", result.RepoURL)
		}
		latestCommit := result.Commits[0]
		freight.Commits = append(freight.Commits, kargoapi.GitCommit{
			RepoURL:   result.RepoURL,
			ID:        latestCommit.ID,
			Branch:    latestCommit.Branch,
			Tag:       latestCommit.Tag,
			Message:   latestCommit.Subject,
			Author:    latestCommit.Author,
			Committer: latestCommit.Committer,
		})
	}

	for _, result := range artifacts.Images {
		if len(result.References) == 0 {
			return nil, fmt.Errorf("no images discovered for repository %q", result.RepoURL)
		}
		latestImage := result.References[0]
		freight.Images = append(freight.Images, kargoapi.Image{
			RepoURL:     result.RepoURL,
			Tag:         latestImage.Tag,
			Digest:      latestImage.Digest,
			Annotations: latestImage.Annotations,
		})
	}

	for _, result := range artifacts.Charts {
		if len(result.Versions) == 0 {
			return nil, fmt.Errorf(
				"no versions discovered for chart %q from repository %q",
				result.RepoURL,
				result.Name,
			)
		}
		latestChart := result.Versions[0]
		freight.Charts = append(freight.Charts, kargoapi.Chart{
			RepoURL: result.RepoURL,
			Name:    result.Name,
			Version: latestChart,
		})
	}

	for _, result := range artifacts.Results {
		if len(result.ArtifactReferences) == 0 {
			return nil, errors.New("no versions discovered for subscription")
		}
		freight.Artifacts = append(
			freight.Artifacts,
			result.ArtifactReferences[0],
		)
	}

	// Generate a unique ID for the Freight based on its contents.
	freight.Name = api.GenerateFreightID(freight)

	return freight, nil
}

func (r *reconciler) patchStatus(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
	update func(*kargoapi.WarehouseStatus),
) error {
	return kubeclient.PatchStatus(ctx, r.client, warehouse, update)
}

// validateDiscoveredArtifacts validates the discovered artifacts and updates
// the Warehouse status with the results. Returns true if the artifacts are
// valid, false otherwise.
func validateDiscoveredArtifacts(
	warehouse *kargoapi.Warehouse,
	newStatus *kargoapi.WarehouseStatus,
) bool {
	artifacts := newStatus.DiscoveredArtifacts

	if artifacts == nil ||
		len(artifacts.Git)+
			len(artifacts.Images)+
			len(artifacts.Charts)+
			len(artifacts.Results) == 0 {
		message := "No artifacts discovered"
		conditions.Set(
			newStatus,
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionFalse,
				Reason:             "MissingArtifacts",
				Message:            message,
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "MissingArtifacts",
				Message:            message,
				ObservedGeneration: warehouse.GetGeneration(),
			},
		)
		return false
	}

	var subscriptions int
	var commits int
	for _, artifact := range artifacts.Git {
		count := len(artifact.Commits)

		if count == 0 {
			message := fmt.Sprintf("No commits discovered for Git repository %q", artifact.RepoURL)
			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoCommitsDiscovered",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingCommits",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}

		subscriptions++
		commits += count
	}

	var images int
	for _, artifact := range artifacts.Images {
		count := len(artifact.References)

		if count == 0 {
			message := fmt.Sprintf("No references discovered for image repository %q", artifact.RepoURL)
			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoImageReferencesDiscovered",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingImageReferences",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}

		subscriptions++
		images += count
	}

	var charts int
	for _, artifact := range artifacts.Charts {
		count := len(artifact.Versions)
		if count == 0 {
			var sb strings.Builder
			_, _ = sb.WriteString("No versions discovered for chart ")
			if artifact.Name != "" {
				_, _ = sb.WriteString(fmt.Sprintf("%q", artifact.Name))
			}
			_, _ = sb.WriteString(" from repository ")
			_, _ = sb.WriteString(fmt.Sprintf("%q", artifact.RepoURL))

			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoChartVersionsDiscovered",
					Message:            sb.String(),
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingChartVersions",
					Message:            sb.String(),
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}
		subscriptions++
		charts += count
	}

	return true
}

// freightCreationCriteriaSatisfied evaluates the freight creation criteria
// expression, if defined, and returns true if the expression is satisfied,
// no expression is defined, or no discovered artifacts are present.
// A non-nil error is returned if there was an error evaluating the expression.
func freightCreationCriteriaSatisfied(
	ctx context.Context,
	fcc *kargoapi.FreightCreationCriteria,
	artifacts *kargoapi.DiscoveredArtifacts,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)

	if fcc == nil {
		logger.Trace("no freight creation criteria")
		return true, nil
	}

	expression := strings.TrimSpace(fcc.Expression)
	if expression == "" {
		logger.Trace("no freight creation criteria expression")
		return true, nil
	}

	if artifacts == nil || (len(artifacts.Git) == 0 && len(artifacts.Images) == 0 && len(artifacts.Charts) == 0) {
		logger.Trace("no artifacts discovered")
		return true, nil
	}

	program, err := expr.Compile(expression, function.DiscoveredArtifactsOperations(artifacts)...)
	if err != nil {
		return false, fmt.Errorf("error compiling expression: %w", err)
	}

	result, err := expr.Run(program, nil)
	if err != nil {
		return false, fmt.Errorf("error running expression: %w", err)
	}

	logger.WithValues(
		"criteriaExpression", expression,
		"result", result,
	).Trace("evaluated freight creation criteria expression")

	switch result := result.(type) {
	case bool:
		return result, nil
	default:
		parsedBool, err := strconv.ParseBool(fmt.Sprintf("%v", result))
		if err != nil {
			return false, fmt.Errorf(
				"failed to parse freight creation criteria expression result %q as bool: %w", result, err,
			)
		}
		return parsedBool, nil
	}
}

// shouldDiscoverArtifacts returns true if the Warehouse should attempt to
// discover new artifacts. This is determined by the following conditions:
//
//   - The Warehouse has not yet discovered any artifacts.
//   - The Warehouse has been updated since the last time we discovered artifacts.
//   - The interval has passed since the last time we discovered artifacts.
//   - A manual refresh was requested.
func shouldDiscoverArtifacts(
	warehouse *kargoapi.Warehouse,
	refreshToken string,
) bool {
	switch {
	// We have not yet discovered any artifacts.
	case warehouse.Status.DiscoveredArtifacts == nil:
		return true
	// We have discovered artifacts, but before we started tracking the
	// last time we did so.
	case warehouse.Status.DiscoveredArtifacts.DiscoveredAt.IsZero():
		return true
	// The Warehouse has been updated since the last time we discovered
	// artifacts.
	case warehouse.Generation > warehouse.Status.ObservedGeneration:
		return true
	// A manual refresh was requested.
	case warehouse.Status.LastHandledRefresh != refreshToken:
		return true
	// We have discovered artifacts, but it's been longer than the interval
	// since we last did so.
	case warehouse.Status.DiscoveredArtifacts.DiscoveredAt.Add(
		warehouse.Spec.Interval.Duration,
	).Before(metav1.Now().Time):
		return true
	default:
		return false
	}
}
