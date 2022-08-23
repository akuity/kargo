package controller

import (
	"context"
	"fmt"
	"time"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			return []string{ticket.(*api.Ticket).Track}
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
			apps := []string{}
			// nolint: forcetypeassert
			for _, station := range track.(*api.Track).Stations {
				for _, app := range station.Applications {
					if !app.Disabled {
						apps = append(apps, app.Name)
					}
				}
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
		result.Requeue = true
	case api.TicketStateNew:
		err = t.reconcileNewTicket(ctx, ticket)
	case api.TicketStateProgressing, api.TicketStateSuspended:
		err = t.reconcileProgressingOrSuspendedTicket(ctx, ticket)
	default:
		// Ignore all other states
		return result, nil
	}

	t.updateTicketStatus(ctx, ticket)
	return result, err
}

func (t *ticketReconciler) reconcileNewTicket(
	ctx context.Context,
	ticket *api.Ticket,
) error {
	// Find the associated Track
	track, err := t.getTrack(ctx, ticket.Track)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Track %q",
			ticket.Track,
		)
		return err
	}
	if track == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Track %q does not exist",
			ticket.Track,
		)
		return nil
	}

	// Find the "zero" Station that we want to migrate to first
	if len(track.Stations) == 0 {
		// This Ticket is implicitly complete
		ticket.Status.State = api.TicketStateCompleted
		ticket.Status.StateReason =
			"Associated Track has no Stations; Nothing to do"
		return nil
	}
	station := track.Stations[0]

	return t.promoteToStation(ctx, ticket, station)
}

func (t *ticketReconciler) reconcileProgressingOrSuspendedTicket(
	ctx context.Context,
	ticket *api.Ticket,
) error {
	// Find the most recent ProgressRecord to see what comes next
	lastProgressRecord :=
		ticket.Status.Progress[len(ticket.Status.Progress)-1]

	// For the moment, the only type of progress is a Migration. So we just need
	// to deal here with started Migrations and complete Migrations
	if lastProgressRecord.Migration != nil &&
		lastProgressRecord.Migration.Completed == nil {
		return t.reconcileLastMigrationStatus(ctx, ticket)
	}
	return t.performNextMigration(ctx, ticket)
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

// TODO: This logic is still not completely correct. It's possible for an Argo
// CD Application to be "fully synced" to a given commit, but healthy NOT on
// account of that commit, but on account of one that came after it. Not sure
// yet how to deal with these sort of scenarios.
func (t *ticketReconciler) reconcileLastMigrationStatus(
	ctx context.Context,
	ticket *api.Ticket,
) error {
	lastMigration :=
		ticket.Status.Progress[len(ticket.Status.Progress)-1].Migration
	// Keep track of what's true about ALL the Argo CD Applications involved in
	// the present migration.
	allAppsSynced := true
	allAppsHealthy := true
	allNonHealthyAppsSuspended := true
	for _, commit := range lastMigration.Commits {
		app, err := t.getArgoCDApplication(ctx, commit.TargetApplication)
		if err != nil {
			ticket.Status.State = api.TicketStateFailed
			ticket.Status.StateReason = fmt.Sprintf(
				"Error getting Argo CD Application %q for Station %q",
				commit.TargetApplication,
				lastMigration.TargetStation,
			)
			return nil
		}
		if app == nil {
			ticket.Status.State = api.TicketStateFailed
			ticket.Status.StateReason = fmt.Sprintf(
				"Argo CD Application %q for Station %q does not exist",
				commit.TargetApplication,
				lastMigration.TargetStation,
			)
			return nil
		}
		if !t.isAppFullySynced(app, commit.SHA) {
			allAppsSynced = false
		}
		switch app.Status.Health.Status {
		case health.HealthStatusHealthy:
		case health.HealthStatusMissing, health.HealthStatusProgressing:
			allAppsHealthy = false
			allNonHealthyAppsSuspended = false
		case health.HealthStatusSuspended:
			allAppsHealthy = false
		default:
			// For any other state, we cannot progress the ticket further, but we're
			// allowing a short grace period because we've seen cases where the Argo
			// CD Application is momentarily Degraded.
			// TODO: Make the grace period configurable?
			if time.Now().UTC().After(lastMigration.Started.Time.Add(time.Minute)) {
				ticket.Status.State = api.TicketStateFailed
				ticket.Status.StateReason = fmt.Sprintf(
					"Argo CD Application %q was fully synced but observed with "+
						"health %q; cannot progress further",
					app.Name,
					app.Status.Health.Status,
				)
				return nil
			}
			allAppsHealthy = false
			allNonHealthyAppsSuspended = false
		}
	}
	ticket.Status.State = api.TicketStateProgressing
	ticket.Status.StateReason = ""
	if allAppsSynced {
		if allAppsHealthy {
			// If all Argo CD Applications are fully synced and healthy, the present
			// migration is complete.
			ticket.Status.Progress[len(ticket.Status.Progress)-1].Migration.Completed = // nolint: lll
				&metav1.Time{Time: time.Now().UTC()}
		} else if allNonHealthyAppsSuspended {
			// If all Argo CD Applications are fully synced and and all those that are
			// NOT healthy ARE Suspended, then we count the Ticket as Suspended as
			// well.
			ticket.Status.State = api.TicketStateSuspended
			ticket.Status.StateReason = "The current migration is blocked by " +
				`one or more Argo CD Applications with health "Suspended"`
		}
	}
	return nil
}

// isAppFullySynced determines if an Argo CD Application is "fully synced" by
// not only examining the Application's sync status, but also by examining
// Application history to validate that the provided commitID is among those
// records.
func (t *ticketReconciler) isAppFullySynced(
	app *argocd.Application,
	commitID string,
) bool {
	if app.Status.Sync.Status == argocd.SyncStatusCodeOutOfSync ||
		app.Status.Sync.Status == argocd.SyncStatusCodeUnknown {
		return false
	}
	for _, revisionHistory := range app.Status.History {
		if revisionHistory.Revision == commitID {
			return true
		}
	}
	return false
}

func (t *ticketReconciler) performNextMigration(
	ctx context.Context,
	ticket *api.Ticket,
) error {
	// Find the associated Track
	track, err := t.getTrack(ctx, ticket.Track)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error getting Track %q",
			ticket.Track,
		)
		return err
	}
	if track == nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Track %q does not exist",
			ticket.Track,
		)
		return nil
	}

	lastProgressRecord := ticket.Status.Progress[len(ticket.Status.Progress)-1]
	var lastStation string
	if lastProgressRecord.Migration != nil {
		lastStation = lastProgressRecord.Migration.TargetStation
	} else {
		lastStation = lastProgressRecord.SkippedStation
	}

	// What's the next Migration? Or are we done?
	lastStationIndex := -1
	for i, station := range track.Stations {
		if station.Name == lastStation {
			lastStationIndex = i
			break
		}
	}

	// This is an edge case where the Track was redefined while the Ticket was
	// progressing and the last Station we migrated to is no longer on the Track.
	// It's not possible to know where to go next.
	if lastStationIndex == -1 {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = "Cannot determine next migration"
		return nil
	}

	// Check if we've reached the end of the Track
	if lastStationIndex == len(track.Stations)-1 {
		ticket.Status.State = api.TicketStateCompleted
		ticket.Status.StateReason = ""
		return nil
	}
	nextStation := track.Stations[lastStationIndex+1]

	return t.promoteToStation(ctx, ticket, nextStation)
}

func (t *ticketReconciler) promoteToStation(
	ctx context.Context,
	ticket *api.Ticket,
	station api.Station,
) error {
	if ticket.Status.Progress == nil {
		ticket.Status.Progress = []api.ProgressRecord{}
	}

	if station.Disabled {
		ticket.Status.State = api.TicketStateProgressing
		ticket.Status.Progress = append(
			ticket.Status.Progress,
			api.ProgressRecord{
				SkippedStation: station.Name,
			},
		)
		return nil
	}

	progressRecord := api.ProgressRecord{
		Migration: &api.Migration{
			TargetStation:       station.Name,
			SkippedApplications: []string{},
			SkippedTracks:       []string{},
			Started:             &metav1.Time{Time: time.Now().UTC()},
		},
	}

	// Find the corresponding Argo CD Applications
	apps := []*argocd.Application{}
	for _, appRef := range station.Applications {
		if appRef.Disabled {
			progressRecord.Migration.SkippedApplications = append(
				progressRecord.Migration.SkippedApplications,
				appRef.Name,
			)
		} else {
			app, err := t.getArgoCDApplication(ctx, appRef.Name)
			if err != nil {
				ticket.Status.State = api.TicketStateFailed
				ticket.Status.StateReason = fmt.Sprintf(
					"Error getting Argo CD Application %q for Station %q",
					appRef.Name,
					station.Name,
				)
				return nil
			}
			if app == nil {
				ticket.Status.State = api.TicketStateFailed
				ticket.Status.StateReason = fmt.Sprintf(
					"Argo CD Application %q for Station %q does not exist",
					appRef.Name,
					station.Name,
				)
				return nil
			}
			apps = append(apps, app)
		}
	}

	// Find the corresponding Tracks
	tracks := []*api.Track{}
	for _, trackRef := range station.Tracks {
		if trackRef.Disabled {
			progressRecord.Migration.SkippedTracks = append(
				progressRecord.Migration.SkippedTracks,
				trackRef.Name,
			)
		} else {
			track, err := t.getTrack(ctx, trackRef.Name)
			if err != nil {
				ticket.Status.State = api.TicketStateFailed
				ticket.Status.StateReason = fmt.Sprintf(
					"Error getting Track %q for Station %q",
					trackRef.Name,
					station.Name,
				)
				return nil
			}
			if track == nil {
				ticket.Status.State = api.TicketStateFailed
				ticket.Status.StateReason = fmt.Sprintf(
					"Track %q for Station %q does not exist",
					trackRef.Name,
					station.Name,
				)
				return nil
			}
			if track.Disabled {
				progressRecord.Migration.SkippedTracks = append(
					progressRecord.Migration.SkippedTracks,
					track.Name,
				)
			} else {
				tracks = append(tracks, track)
			}
		}
	}

	// Promote by making commits
	progressRecord.Migration.Commits = make([]api.Commit, len(apps))
	for i, app := range apps {
		commitSHA, err := t.promoteImages(ctx, ticket, app)
		if err != nil {
			ticket.Status.State = api.TicketStateFailed
			ticket.Status.StateReason = fmt.Sprintf(
				"Error promoting images to Argo CD Application %q in Station %q",
				app.Name,
				station.Name,
			)
			return err
		}
		progressRecord.Migration.Commits[i] = api.Commit{
			TargetApplication: app.Name,
			SHA:               commitSHA,
		}
	}

	// Promote by creating new Tickets
	progressRecord.Migration.Tickets = make([]api.TicketReference, len(tracks))
	for i, track := range tracks {
		newTicket := api.Ticket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewV4().String(),
				Namespace: t.config.Namespace,
			},
			Track:  track.Name,
			Change: ticket.Change,
		}
		if err := t.client.Create(
			ctx,
			&newTicket,
			&client.CreateOptions{},
		); err != nil {
			ticket.Status.State = api.TicketStateFailed
			ticket.Status.StateReason = fmt.Sprintf(
				"Error creating new Ticket for Track %q",
				track.Name,
			)
			return errors.Wrapf(
				err,
				"error creating Ticket %s",
				ticket.Name,
			)
		}
		t.logger.WithFields(log.Fields{
			"name":  ticket.Name,
			"track": ticket.Track,
		}).Debug("Created Ticket resource")
		progressRecord.Migration.Tickets[i] = api.TicketReference{
			Name:  newTicket.Name,
			Track: track.Name,
		}
	}

	ticket.Status.State = api.TicketStateProgressing
	ticket.Status.Progress = append(ticket.Status.Progress, progressRecord)

	t.logger.WithFields(log.Fields{
		"ticket":  ticket.Name,
		"track":   ticket.Track,
		"station": station.Name,
	}).Debug("promoted images")

	return nil
}
