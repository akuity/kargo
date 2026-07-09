package promotion

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	"github.com/akuity/kargo/pkg/kubernetes"
	libEvent "github.com/akuity/kargo/pkg/kubernetes/event"
	"github.com/akuity/kargo/pkg/logging"
	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

var (
	promotionGroupKind = schema.GroupKind{
		Group: kargoapi.GroupVersion.Group,
		Kind:  "Promotion",
	}
	promotionGroupResource = schema.GroupResource{
		Group:    kargoapi.GroupVersion.Group,
		Resource: "Promotion",
	}
)

type webhook struct {
	client  client.Client
	decoder admission.Decoder

	sender kargoEvent.Sender

	// The following behaviors are overridable for testing purposes:

	getFreightFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Freight, error)

	listFreightAvailableToStageFn func(
		context.Context,
		client.Client,
		*kargoapi.Stage,
	) ([]kargoapi.Freight, error)

	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)

	validateProjectFn func(
		context.Context,
		client.Client,
		client.Object,
	) error

	authorizeFn func(
		ctx context.Context,
		promo *kargoapi.Promotion,
		action string,
	) error

	admissionRequestFromContextFn func(context.Context) (admission.Request, error)

	createSubjectAccessReviewFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	isAutoPromotionEnabledFn func(
		context.Context,
		client.Client,
		metav1.ObjectMeta,
	) (bool, error)

	isRequestFromKargoControlplaneFn libWebhook.IsRequestFromKargoControlplaneFn

	// externalWebhooksServerUsername is the exact username of the external
	// webhooks server service account. When an admission request originates
	// from this subject, the "promote" verb authorization check is bypassed.
	externalWebhooksServerUsername string
}

func SetupWebhookWithManager(
	ctx context.Context,
	cfg libWebhook.Config,
	mgr ctrl.Manager,
) error {
	w := newWebhook(
		cfg,
		mgr.GetClient(),
		admission.NewDecoder(mgr.GetScheme()),
		k8sevent.NewEventSender(libEvent.NewRecorder(ctx, mgr.GetScheme(), mgr.GetClient(), "promotion-webhook")),
	)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kargoapi.Promotion{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

func newWebhook(
	cfg libWebhook.Config,
	kubeClient client.Client,
	decoder admission.Decoder,
	sender kargoEvent.Sender,
) *webhook {
	w := &webhook{
		client:  kubeClient,
		decoder: decoder,
		sender:  sender,
	}
	w.getFreightFn = api.GetFreight
	w.listFreightAvailableToStageFn = api.ListFreightAvailableToStage
	w.getStageFn = api.GetStage
	w.isAutoPromotionEnabledFn = api.IsAutoPromotionEnabled
	w.validateProjectFn = libWebhook.ValidateProject
	w.authorizeFn = w.authorize
	w.admissionRequestFromContextFn = admission.RequestFromContext
	w.createSubjectAccessReviewFn = w.client.Create
	w.isRequestFromKargoControlplaneFn = libWebhook.IsRequestFromKargoControlplane(cfg.ControlplaneUserRegex)
	w.externalWebhooksServerUsername = cfg.ExternalWebhooksServerUsername
	return w
}

func (w *webhook) Default(ctx context.Context, obj runtime.Object) error {
	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		return fmt.Errorf("get admission request from context: %w", err)
	}

	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert

	if promo.Annotations == nil {
		promo.Annotations = make(map[string]string, 2)
	}

	stage, err := w.getStageFn(
		ctx,
		w.client,
		types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"error finding Stage %q in namespace %q: %w",
			promo.Spec.Stage,
			promo.Namespace,
			err,
		)
	}
	if stage == nil {
		return fmt.Errorf(
			"could not find Stage %q in namespace %q",
			promo.Spec.Stage,
			promo.Namespace,
		)
	}

	switch req.Operation {
	case admissionv1.Create:
		// Note: Validation makes these mutually exclusive, but defaulting webhooks
		// fire before validating webhooks. Only try to resolve origin to the
		// candidate Freight for that origin if the Freight field is also empty.
		// If Origin is non-nil AND Freight is non-empty, validation will catch it
		// when it eventually fires, so skipping resolution is all we need to do.
		if promo.Spec.Origin != nil && promo.Spec.Freight == "" {
			// Note: we could theoretically infer an omitted origin when the Stage
			// requests Freight from only one origin, but we've elected not to.
			// Promotion specs are clearer and more stable when promote-by-origin
			// is explicit; endpoint-only conveniences like freightAlias stay out here.
			var freight *kargoapi.Freight
			if freight, err = w.resolveOriginToFreight(ctx, *promo.Spec.Origin, stage); err != nil {
				return err
			}
			promo.Spec.Freight = freight.Name
			promo.Spec.Origin = nil
		}

		// Comparison and sorting logic elsewhere in the Kargo code base depend on
		// adhering to a specific naming convention. Always overwrite the name with
		// one generated by Kargo.
		promo.Name = api.GeneratePromotionName(stage.Name, promo.Spec.Freight)

		// Set actor as an admission request's user info when the promotion is
		// created to allow controllers to track who created it.
		if !w.isRequestFromKargoControlplaneFn(req) {
			promo.Annotations[kargoapi.AnnotationKeyCreateActor] = api.FormatEventKubernetesUserActor(req.UserInfo)
		}

		// Enrich the annotation with the actor and control plane information.
		w.setAbortAnnotationActor(req, nil, promo)

		// Reject the Promotion if the Stage's PromotionTemplate is missing or
		// defines no steps. Done here (in the mutating webhook) for the sake of
		// surfacing a nicer, more informative error message than the one that one
		// would otherwise get when the CRD's declarative validation fails due to
		// the missing steps.
		if stage.Spec.PromotionTemplate == nil || len(stage.Spec.PromotionTemplate.Spec.Steps) == 0 {
			// nolint:staticcheck
			return fmt.Errorf(
				"Stage %q in namespace %q defines no promotion steps",
				promo.Spec.Stage,
				promo.Namespace,
			)
		}

		// Promotions must always follow the process defined by the Stage they
		// reference. Deviation from that is not supported. Unconditionally
		// (re-)populate the Promotion's steps and vars according to the Stage's
		// PromotionTemplate.
		promo.Spec.Steps = stage.Spec.PromotionTemplate.Spec.Steps
		vars := make(
			[]kargoapi.ExpressionVariable,
			0,
			len(stage.Spec.Vars)+len(stage.Spec.PromotionTemplate.Spec.Vars),
		)
		vars = append(vars, stage.Spec.Vars...)
		vars = append(vars, stage.Spec.PromotionTemplate.Spec.Vars...)
		promo.Spec.Vars = vars

		// Inflate any PromotionTasks in the Promotion's steps.
		if err = api.InflateSteps(ctx, w.client, promo); err != nil {
			return fmt.Errorf("failed to inflate Promotion steps: %w", err)
		}

		w.syncHoldAnnotations(ctx, req, promo, stage)
	case admissionv1.Update:
		// We need to decode the old object manually since controller-runtime
		// doesn't decode it for us.
		oldPromo := &kargoapi.Promotion{}
		if err = w.decoder.DecodeRaw(req.OldObject, oldPromo); err != nil {
			return fmt.Errorf("decode old object: %w", err)
		}

		// These annotations describe the creation request. Preserve the admitted
		// values on every update so status writers and clients cannot change hold
		// or release intent after the Promotion has been created.
		preserveAnnotation := func(key string) {
			if oldValue, ok := oldPromo.Annotations[key]; ok {
				promo.Annotations[key] = oldValue
				return
			}
			delete(promo.Annotations, key)
		}
		preserveAnnotation(kargoapi.AnnotationKeyCreateActor)
		preserveAnnotation(kargoapi.AnnotationKeyAutoPromotionHold)
		preserveAnnotation(kargoapi.AnnotationKeyAutoPromotionResume)

		// Enrich the annotation with the actor and control plane information.
		w.setAbortAnnotationActor(req, oldPromo, promo)
	}

	// Make sure the Promotion has the same shard as the Stage.
	if promo.Labels == nil {
		promo.Labels = make(map[string]string, 2)
	}
	if stage.Spec.Shard != "" {
		promo.Labels[kargoapi.LabelKeyShard] = stage.Spec.Shard
	} else {
		delete(promo.Labels, kargoapi.LabelKeyShard)
	}

	// Always label/annotate the Promotion with the Stage name for easy filtering.
	promo.Labels[kargoapi.LabelKeyStage] = kubernetes.ShortenLabelValue(promo.Spec.Stage)
	promo.Annotations[kargoapi.AnnotationKeyStage] = promo.Spec.Stage

	ownerRef := metav1.NewControllerRef(stage, kargoapi.GroupVersion.WithKind("Stage"))
	promo.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	return nil
}

// resolveOriginToFreight resolves origin to the Freight that the origin's
// selection policy would choose for stage. Returns an error (denying
// admission) if no Freight is available.
func (w *webhook) resolveOriginToFreight(
	ctx context.Context,
	origin kargoapi.FreightOrigin,
	stage *kargoapi.Stage,
) (*kargoapi.Freight, error) {
	availableFreight, err := w.listFreightAvailableToStageFn(ctx, w.client, stage)
	if err != nil {
		return nil, fmt.Errorf("list available freight: %w", err)
	}
	candidates := api.SelectAutoPromotionCandidates(ctx, stage, availableFreight)
	originKey := origin.String()
	candidate, ok := candidates[originKey]
	if !ok {
		return nil, fmt.Errorf(
			"no auto-promotion candidate found for origin %q on Stage %q",
			originKey,
			stage.Name,
		)
	}
	return &candidate, nil
}

// syncHoldAnnotations stamps the correct hold/resume intent annotation on a
// Promotion so the Stage controller can maintain auto-promotion holds. Errors
// are swallowed; intent inference is best-effort and must not block Promotion
// creation.
//
// A Promotion is treated as user-initiated when the admission request did
// not originate from the Kargo control plane OR when it did, but the
// Promotion carries a create-actor annotation identifying a non-controller
// actor. The latter covers the Kargo API server, which creates Promotions on
// users' behalf using its own service account (a control-plane identity) and
// records the requesting user in that annotation. Genuinely system-generated
// Promotions (the auto-promotion loop, which sets no create-actor; rollback
// controllers, which set a controller: actor) are left untouched — they carry
// no user intent, and any intent annotations they set explicitly are
// preserved as-is.
//
// For user-initiated Promotions, any caller-supplied intent annotation is
// stripped first to prevent circumvention, then intent is inferred: resume if
// the promoted Freight matches the current auto-promotion candidate, hold
// otherwise (including when no candidate exists yet).
//
// Accepted races:
//
// 1. A Promotion can be created for the current candidate, then newer
// Freight can become available before this webhook infers intent. The Promotion
// may be stamped as hold intent even though it selected the candidate visible
// at creation time. This is benign: the selected Freight is still promoted, and
// promoting the new current candidate releases the hold. Resolving this fully
// would require extra candidate identity plumbing and still could not eliminate
// every stale-read window.
//
// 2. A Promotion can select non-candidate Freight while an
// auto-promotion for the candidate is also being created. If the auto-promotion
// reaches the queue first, it runs first; the hold-intent Promotion runs after
// it and blocks future auto-promotion. That order is self-correcting, so
// avoiding it is not worth more coordination state. In the mirror ordering --
// the hold-intent Promotion is created first, but only after auto-promotion
// already decided to act -- the queued auto-promotion runs second and
// supersedes the user's Freight once. The hold still takes effect (the
// auto-promotion carries no resume intent), so the anomaly is visible in
// Stage status, future auto-promotion stays blocked, and re-promoting the
// older Freight recovers. No read-time guard can see a Promotion created
// after the read, so this too is accepted.
func (w *webhook) syncHoldAnnotations(
	ctx context.Context,
	req admission.Request,
	promo *kargoapi.Promotion,
	stage *kargoapi.Stage,
) {
	// System-generated Promotions carry no user intent; leave them untouched.
	// Control-plane requests are user-initiated only when the Promotion names
	// a non-controller actor: the API server records the requesting user in
	// the create-actor annotation before creating a Promotion on their behalf,
	// while the Stage controller's auto-promotions set no actor at all and
	// other controllers identify themselves with a controller: actor.
	if w.isRequestFromKargoControlplaneFn(req) {
		actor := promo.Annotations[kargoapi.AnnotationKeyCreateActor]
		if actor == "" || strings.HasPrefix(actor, kargoapi.EventActorControllerPrefix) {
			return
		}
	}
	// Strip any caller-supplied intent to prevent circumvention, then infer.
	delete(promo.Annotations, kargoapi.AnnotationKeyAutoPromotionHold)
	delete(promo.Annotations, kargoapi.AnnotationKeyAutoPromotionResume)
	if w.getFreightFn == nil || w.listFreightAvailableToStageFn == nil ||
		w.isAutoPromotionEnabledFn == nil || promo.Spec.Freight == "" {
		return
	}
	logger := logging.LoggerFromContext(ctx)
	// Only stamp intent for Stages that have auto-promotion enabled. Use the
	// live ProjectConfig rather than the cached stage.Status.AutoPromotionEnabled,
	// which may lag behind ProjectConfig changes.
	enabled, err := w.isAutoPromotionEnabledFn(ctx, w.client, stage.ObjectMeta)
	if err != nil {
		logger.Error(err, "skipping auto-promotion intent annotation")
		return
	}
	if !enabled {
		return
	}
	freight, err := w.getFreightFn(ctx, w.client, types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Freight,
	})
	if err != nil {
		logger.Error(err, "skipping auto-promotion intent annotation")
		return
	}
	if freight == nil {
		return
	}
	origin := freight.Origin

	availableFreight, err := w.listFreightAvailableToStageFn(ctx, w.client, stage)
	if err != nil {
		logger.Error(err, "skipping auto-promotion intent annotation")
		return
	}
	candidates := api.SelectAutoPromotionCandidates(ctx, stage, availableFreight)

	candidate, ok := candidates[origin.String()]
	if !ok {
		// No candidate exists right now e.g. MatchUpstream policy when
		// upstream Freight has not yet passed verification or its soak time.
		// Stamp hold intent so auto-promotion does not immediately proceed the
		// moment a candidate does appear.
		api.SetAutoPromotionHoldAnnotation(promo, origin)
		return
	}
	if candidate.Name == promo.Spec.Freight {
		// Promoted Freight is the candidate; stamp resume intent. The Stage
		// controller resumes auto-promotion for this origin when a succeeded
		// Promotion carries this annotation.
		api.SetAutoPromotionResumeAnnotation(promo, origin)
		return
	}
	// Promoted Freight is not the candidate; stamp hold intent.
	api.SetAutoPromotionHoldAnnotation(promo, origin)
}

func (w *webhook) ValidateCreate(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert

	if err := w.validateProjectFn(ctx, w.client, promo); err != nil {
		var statusErr *apierrors.StatusError
		if ok := errors.As(err, &statusErr); ok {
			return nil, statusErr
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); !ok {
			return nil, apierrors.NewInternalError(err)
		}
		return nil, apierrors.NewInvalid(
			promotionGroupKind,
			promo.Name,
			field.ErrorList{fieldErr},
		)
	}

	if err := w.authorizeFn(ctx, promo, "create"); err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	if (promo.Spec.Freight == "") == (promo.Spec.Origin == nil) {
		return nil, apierrors.NewInvalid(
			promotionGroupKind,
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec"),
					promo.Spec,
					"exactly one of spec.freight or spec.origin must be set",
				),
			},
		)
	}

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(
			fmt.Errorf("get admission request from context: %w", err),
		)
	}

	stage, err := w.getStageFn(ctx, w.client, types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Stage,
	})
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("get stage: %w", err))
	}

	freight, err := w.getFreightFn(ctx, w.client, types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Freight,
	})
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("get freight: %w", err))
	}

	if !stage.IsFreightAvailable(freight) {
		return nil, apierrors.NewInvalid(
			promotionGroupKind,
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "freight"),
					promo.Spec.Freight,
					"Freight is not available to this Stage",
				),
			},
		)
	}

	// Record Promotion created event if the request doesn't come from Kargo controlplane
	if !w.isRequestFromKargoControlplaneFn(req) {
		w.recordPromotionCreatedEvent(ctx, req, promo, freight)
	}

	return nil, nil
}

func (w *webhook) ValidateUpdate(
	ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) (admission.Warnings, error) {
	promo := newObj.(*kargoapi.Promotion)    // nolint: forcetypeassert
	oldPromo := oldObj.(*kargoapi.Promotion) // nolint: forcetypeassert
	if err := w.authorizeFn(ctx, promo, "update"); err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	// PromotionSpecs are meant to be immutable
	if !reflect.DeepEqual(promo.Spec, oldPromo.Spec) {
		return nil, apierrors.NewInvalid(
			promotionGroupKind,
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec"),
					promo.Spec,
					"spec is immutable",
				),
			},
		)
	}

	for _, key := range []string{
		kargoapi.AnnotationKeyAutoPromotionHold,
		kargoapi.AnnotationKeyAutoPromotionResume,
	} {
		if promo.Annotations[key] == oldPromo.Annotations[key] {
			continue
		}
		return nil, apierrors.NewInvalid(
			promotionGroupKind,
			promo.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("metadata", "annotations").Key(key),
					promo.Annotations[key],
					"annotation is immutable",
				),
			},
		)
	}

	return nil, nil
}

func (w *webhook) ValidateDelete(
	ctx context.Context,
	obj runtime.Object,
) (admission.Warnings, error) {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	return nil, w.authorizeFn(ctx, promo, "delete")
}

func (w *webhook) authorize(
	ctx context.Context,
	promo *kargoapi.Promotion,
	action string,
) error {
	logger := logging.LoggerFromContext(ctx)

	req, err := w.admissionRequestFromContextFn(ctx)
	if err != nil {
		logger.Error(err, "")
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			fmt.Errorf(
				"error retrieving admission request from context; refusing to "+
					"%s Promotion",
				action,
			),
		)
	}

	// The external webhooks server is trusted to refresh running Promotions on
	// behalf of webhook callers. Skip the "promote" verb check for this subject.
	if req.UserInfo.Username == w.externalWebhooksServerUsername {
		return nil
	}

	accessReview := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			User:   req.UserInfo.Username,
			Groups: req.UserInfo.Groups,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     kargoapi.GroupVersion.Group,
				Resource:  "stages",
				Name:      promo.Spec.Stage,
				Verb:      "promote",
				Namespace: promo.Namespace,
			},
		},
	}
	if err := w.createSubjectAccessReviewFn(ctx, accessReview); err != nil {
		logger.Error(err, "")
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			fmt.Errorf(
				"error creating SubjectAccessReview; refusing to %s Promotion",
				action,
			),
		)
	}

	if !accessReview.Status.Allowed {
		return apierrors.NewForbidden(
			promotionGroupResource,
			promo.Name,
			fmt.Errorf(
				"subject %q is not permitted to %s Promotions for Stage %q",
				req.UserInfo.Username,
				action,
				promo.Spec.Stage,
			),
		)
	}

	return nil
}

func (w *webhook) recordPromotionCreatedEvent(
	ctx context.Context,
	req admission.Request,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) {
	actor := api.FormatEventKubernetesUserActor(req.UserInfo)
	evt := kargoEvent.NewPromotionCreated(fmt.Sprintf("Promotion created for Stage %q by %q",
		p.Spec.Stage,
		actor), actor, p, f)
	if err := w.sender.Send(ctx, evt); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err,
			"failed to send Promotion created event",
		)
	}
}

func (w *webhook) setAbortAnnotationActor(req admission.Request, existing, updated *kargoapi.Promotion) {
	if abortReq, ok := api.AbortPromotionAnnotationValue(updated.Annotations); ok {
		var oldAbortReq *kargoapi.AbortPromotionRequest
		if existing != nil {
			oldAbortReq, _ = api.AbortPromotionAnnotationValue(existing.Annotations)
		}
		// If the abort request has changed, enrich the annotation with the
		// actor and control plane information.
		if existing == nil || oldAbortReq == nil || !abortReq.Equals(oldAbortReq) {
			abortReq.ControlPlane = w.isRequestFromKargoControlplaneFn(req)
			if !abortReq.ControlPlane {
				// If the abort request is not from the control plane, then it's
				// from a specific Kubernetes user. Without this check we would
				// overwrite the actor field set by the control plane.
				abortReq.Actor = api.FormatEventKubernetesUserActor(req.UserInfo)
			}
			updated.Annotations[kargoapi.AnnotationKeyAbort] = abortReq.String()
		}
	}
}
