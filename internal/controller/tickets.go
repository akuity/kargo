package controller

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
)

const (
	ticketsByTrackIndexField      = ".spec.track"
	tracksByApplicationIndexField = "applications"
)

// ticketReconciler reconciles Ticket resources.
type ticketReconciler struct {
	config config.Config
	client client.Client
	argoDB db.ArgoDB
	logger *log.Logger
	// Promotions are a critical section of the code
	promoMutex sync.Mutex
	// The following internal functions are overridable for testing purposes
	promoteImageFn func(
		context.Context,
		*api.Ticket,
		*argocd.Application,
	) (string, error)
	setupGitAuthFn    func(ctx context.Context, repoURL string) error
	tearDownGitAuthFn func()
	execCommandFn     func(*exec.Cmd) ([]byte, error)
}

// SetupTicketReconcilerWithManager initializes a reconciler for Ticket
// resources and registers it with the provided Manager.
func SetupTicketReconcilerWithManager(
	ctx context.Context,
	config config.Config,
	mgr manager.Manager,
	argoDB db.ArgoDB,
) error {
	logger := log.New()
	logger.SetLevel(config.LogLevel)

	// NB: We build TWO indices here. Tickets do not directly reference associated
	// Argo CD Applications. They are associated with Applications via an
	// intermediate resource -- a Track. If we want to reconcile related Tickets
	// every time the state of an Application changes, we need to first find
	// related Tracks, then, for each Track, find the related Tickets. To make
	// these list operations as efficient as possible, we index Tickets by Track
	// AND Tracks by Application.

	// Index Tickets by Track
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Ticket{},
		ticketsByTrackIndexField,
		func(ticket client.Object) []string {
			// nolint: forcetypeassert
			return []string{ticket.(*api.Ticket).Spec.Track}
		},
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing Tickets by Track",
		)
	}

	// Index Tracks by Argo CD Applications
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Track{},
		tracksByApplicationIndexField,
		func(track client.Object) []string {
			envs := track.(*api.Track).Environments // nolint: forcetypeassert
			apps := make([]string, len(envs))
			for i, env := range envs {
				apps[i] = env.Application
			}
			return apps
		},
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing Tracks by ArgoCD Applications",
		)
	}

	t := &ticketReconciler{
		config: config,
		client: mgr.GetClient(),
		argoDB: argoDB,
		logger: logger,
	}
	t.promoteImageFn = t.promoteImage
	t.setupGitAuthFn = t.setupGitAuth
	t.tearDownGitAuthFn = t.tearDownGitAuth
	t.execCommandFn = t.execCommand

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Ticket{}).WithEventFilter(predicate.Funcs{
		DeleteFunc: func(event.DeleteEvent) bool {
			// We're not interested in any deletes
			return false
		},
	}).Watches(
		&source.Kind{Type: &argocd.Application{}},
		handler.EnqueueRequestsFromMapFunc(t.findTicketsForApplication),
	).Complete(t)
}

// findTicketsForApplication dynamically returns reconciliation requests for all
// Tickets related to a given Argo CD Application. This takes advantage of both
// indices established by SetupTicketReconcilerWithManager() and is used to
// propagate reconciliation requests to Tickets whose state should be affected
// by changes to relates Application resources.
func (t *ticketReconciler) findTicketsForApplication(
	application client.Object,
) []reconcile.Request {
	tracks := api.TrackList{}
	if err := t.client.List(
		context.Background(),
		&tracks,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				tracksByApplicationIndexField,
				application.GetName(),
			),
		},
	); err != nil {
		t.logger.WithFields(log.Fields{
			"application": application.GetName(),
		}).Error("error listing Tracks associated with Argo CD Application")
		return nil
	}
	requests := []reconcile.Request{}
	for _, track := range tracks.Items {
		tickets := &api.TicketList{}
		if err := t.client.List(
			context.Background(),
			tickets,
			&client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					ticketsByTrackIndexField,
					track.GetName(),
				),
			},
		); err != nil {
			t.logger.WithFields(log.Fields{
				"track": track.Name,
			}).Error("error listing Tickets associated with Track")
			return nil
		}
		reqs := make([]reconcile.Request, len(tickets.Items))
		for i, item := range tickets.Items {
			reqs[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
		}
		requests = append(requests, reqs...)
	}
	return requests
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (t *ticketReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{}

	t.logger.WithFields(log.Fields{
		"name": req.NamespacedName.Name,
	}).Debug("reconciling Ticket")

	// Find the Ticket
	ticket, err := t.getTicket(ctx, req.Name)
	if err != nil {
		return result, err
	}
	if ticket == nil {
		// Ignore if not found. This can happen if the Ticket was deleted after the
		// current reconciliation request was issued.
		return result, nil
	}

	// What's the current state of the ticket?
	switch ticket.Status.State {
	case "":
		// Add the initial state and requeue
		ticket.Status.State = api.TicketStateNew
		t.updateTicketStatus(ctx, ticket)
		result.Requeue = true
		return result, nil
	case api.TicketStateNew, api.TicketStateProgressing:
		// Proceed if one of the above
	default:
		// Ignore all other states
		return result, nil
	}

	// Find the associated Track
	track, err := t.getTrack(ctx, ticket.Spec.Track)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Track %q",
			ticket.Spec.Track,
		)
		t.updateTicketStatus(ctx, ticket)
		return result, err
	}
	if track == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Track %q does not exist",
			ticket.Spec.Track,
		)
		t.updateTicketStatus(ctx, ticket)
		return result, nil
	}

	// What's the current state of the Ticket?
	switch ticket.Status.State {
	case api.TicketStateNew:
		return result, t.reconcileNewTicket(ctx, ticket, track)
	case api.TicketStateProgressing:
		return result, t.reconcileProgressingTicket(ctx, ticket, track)
	}

	// We don't have anything to do in the current state
	return result, nil
}

func (t *ticketReconciler) reconcileNewTicket(
	ctx context.Context,
	ticket *api.Ticket,
	track *api.Track,
) error {
	// Find the "zero" environment that we want to migrate to first
	if len(track.Environments) == 0 {
		// This Ticket is implicitly complete
		ticket.Status.State = api.TicketStateCompleted
		ticket.Status.StateReason =
			"Associated Track has no environments; Nothing to do"
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	env := track.Environments[0]

	// Find the corresponding Argo CD Application
	app, err := t.getArgoCDApplication(ctx, env.Application)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Argo CD Application %q for environment %q",
			env.Application,
			env.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return nil
	}
	if app == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Argo CD Application %q for environment %q does not exist",
			env.Application,
			env.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	loggerFields := log.Fields{
		"ticket":           ticket.Name,
		"track":            ticket.Spec.Track,
		"environment":      env.Name,
		"application":      env.Application,
		"imageRepo":        ticket.Spec.Change.ImageRepo,
		"imageTag":         ticket.Spec.Change.ImageTag,
		"gitopsRepoURL":    app.Spec.Source.RepoURL,
		"gitopsRepoBranch": app.Spec.Source.TargetRevision,
	}

	// Promote
	commitSHA, err := t.promoteImageFn(ctx, ticket, app)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error promoting image to environment %q",
			env.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return err
	}

	loggerFields["gitopsRepoCommit"] = commitSHA
	t.logger.WithFields(loggerFields).Debug("promoted image")

	ticket.Status.State = api.TicketStateProgressing
	ticket.Status.StateReason = fmt.Sprintf(
		"Image has been promoted to environment %q",
		env.Name,
	)
	ticket.Status.Progress = []api.Transition{
		{
			TargetEnvironment: env.Name,
			TargetApplication: env.Application,
			CommitSHA:         commitSHA,
		},
	}
	t.updateTicketStatus(ctx, ticket)
	return nil
}

func (t *ticketReconciler) reconcileProgressingTicket(
	ctx context.Context,
	ticket *api.Ticket,
	track *api.Track,
) error {
	// Find the most recent environment the change represented by the Ticket was
	// migrated to
	lastEnvName :=
		ticket.Status.Progress[len(ticket.Status.Progress)-1].TargetEnvironment
	lastAppName :=
		ticket.Status.Progress[len(ticket.Status.Progress)-1].TargetApplication

	// Find the corresponding Argo CD Application
	app, err := t.getArgoCDApplication(ctx, lastAppName)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Argo CD Application %q for environment %q",
			lastAppName,
			lastEnvName,
		)
		t.updateTicketStatus(ctx, ticket)
		return nil
	}
	if app == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Argo CD Application %q for environment %q does not exist",
			lastAppName,
			lastEnvName,
		)
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	// Determine if the last Transition is complete
	lastTransition := ticket.Status.Progress[len(ticket.Status.Progress)-1]
	var lastTransitionInAppHistory bool
	for _, record := range app.Status.History {
		if record.Revision == lastTransition.CommitSHA {
			lastTransitionInAppHistory = true
			break
		}
	}

	// TODO: This logic isn't quite correct. This leaves open the possibility that
	// the change we migrated didn't sync successfully , but a subsequent change
	// HAS synced successfully. Not sure yet how to deal with that scenario yet.
	if !(lastTransitionInAppHistory &&
		app.Status.Sync.Status == argocd.SyncStatusCodeSynced) {
		return nil // Nothing to do
	}

	// What's the next transition? Or are we done?
	lastEnvIndex := -1
	for i, env := range track.Environments {
		if env.Name == lastTransition.TargetEnvironment {
			lastEnvIndex = i
			break
		}
	}

	// This is an edge case where the Track was redefined while the Ticket was
	// progressing and the last environment we migrated into is no longer on the
	// Track. It's not possible to know where to go next.
	if lastEnvIndex == -1 {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = "Cannot determine next migration"
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	// Check if we've reached the end of the Track
	if lastEnvIndex == len(track.Environments)-1 {
		ticket.Status.State = api.TicketStateCompleted
		ticket.Status.StateReason = ""
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	// If we get to here, we can migrate into the next environment
	nextEnv := track.Environments[lastEnvIndex+1]
	nextApp, err := t.getArgoCDApplication(ctx, nextEnv.Application)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Argo CD Application %q for environment %q",
			nextEnv.Application,
			nextEnv.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return err
	}
	if nextApp == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Argo CD Application %q for environment %q does not exist",
			nextEnv.Application,
			nextEnv.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return nil
	}

	loggerFields := log.Fields{
		"ticket":           ticket.Name,
		"track":            ticket.Spec.Track,
		"environment":      nextEnv.Name,
		"application":      nextEnv.Application,
		"imageRepo":        ticket.Spec.Change.ImageRepo,
		"imageTag":         ticket.Spec.Change.ImageTag,
		"gitopsRepoURL":    nextApp.Spec.Source.RepoURL,
		"gitopsRepoBranch": nextApp.Spec.Source.TargetRevision,
	}

	// Promote
	commitSHA, err := t.promoteImageFn(ctx, ticket, nextApp)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error promoting image to environment %q",
			nextEnv.Name,
		)
		t.updateTicketStatus(ctx, ticket)
		return errors.Wrapf(
			err,
			"error promoting image to environment %q",
			nextEnv.Name,
		)
	}

	loggerFields["gitopsRepoCommit"] = commitSHA
	t.logger.WithFields(loggerFields).Debug("promoted image")

	ticket.Status.State = api.TicketStateProgressing
	ticket.Status.StateReason = fmt.Sprintf(
		"Image has been promoted to environment %q",
		nextEnv.Name,
	)
	ticket.Status.Progress = append(
		ticket.Status.Progress,
		api.Transition{
			TargetEnvironment: nextEnv.Name,
			TargetApplication: nextEnv.Application,
			CommitSHA:         commitSHA,
		},
	)
	t.updateTicketStatus(ctx, ticket)
	return nil
}

// getTicket returns a pointer to the Ticket resource having the name specified
// by the name argument. If no such resource is found, nil is returned instead.
func (t *ticketReconciler) getTicket(
	ctx context.Context,
	name string,
) (*api.Ticket, error) {
	ticket := api.Ticket{}
	if err := t.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: t.config.Namespace,
			Name:      name,
		},
		&ticket,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"name": name,
			}).Warn("Ticket not found")
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error getting Ticket %q", name)
	}
	return &ticket, nil
}

// updateTicketStatus updates the status subresource of the provided Ticket.
func (t *ticketReconciler) updateTicketStatus(
	ctx context.Context,
	ticket *api.Ticket,
) {
	if err := t.client.Status().Update(ctx, ticket); err != nil {
		t.logger.WithFields(log.Fields{
			"name": ticket.Name,
		}).Error("error updating ticket status")
	}
}

// getTrack returns a pointer to the Track resource having the name specified by
// the name argument. If no such resource is found, nil is returned instead.
func (t *ticketReconciler) getTrack(
	ctx context.Context,
	name string,
) (*api.Track, error) {
	track := api.Track{}
	if err := t.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: t.config.Namespace,
			Name:      name,
		},
		&track,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"name": name,
			}).Warn("Track not found")
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error getting Track %q", name)
	}
	return &track, nil
}

// getArgoCDApplication returns a pointer to the Argo CD Application resource
// having the name specified by the name argument. If no such resource is found,
// nil is returned instead.
func (t *ticketReconciler) getArgoCDApplication(
	ctx context.Context,
	name string,
) (*argocd.Application, error) {
	app := argocd.Application{}
	if err := t.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: t.config.Namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"name": name,
			}).Warn("Argo CD Application not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Application %q",
			name,
		)
	}
	return &app, nil
}
