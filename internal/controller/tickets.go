package controller

import (
	"context"
	"fmt"
	"os/exec"

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

const applicationsIndexField = "applications"

// ticketReconciler reconciles Ticket resources.
type ticketReconciler struct {
	client client.Client
	argoDB db.ArgoDB
	logger *log.Logger
	// The following internal functions are overridable for testing purposes
	promoteImageFn func(
		ctx context.Context,
		imageRepoName string,
		imageTag string,
		gitopsRepoURL string,
		envBranch string,
	) (string, error)
	execCommandFn func(*exec.Cmd) error
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

	// Index Tickets by associated ArgoCD Applications
	// TODO: How do we keep this index up to date?
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Ticket{},
		applicationsIndexField,
		func(ticket client.Object) []string {
			line := api.Line{}
			lineName := ticket.(*api.Ticket).Spec.Line // nolint: forcetypeassert
			if err := mgr.GetClient().Get(
				ctx,
				types.NamespacedName{
					Namespace: ticket.GetNamespace(),
					Name:      lineName,
				},
				&line,
			); err != nil {
				logger.WithFields(log.Fields{
					"ticket": ticket.GetName(),
					"line":   lineName,
				}).Errorf(
					"could not get Argo CD Applications associated with Ticket; "+
						"error getting intermediate Line resource: %s",
					err,
				)
				return nil
			}
			return line.Environments
		},
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing Tickets by associated ArgoCD Applications",
		)
	}

	t := &ticketReconciler{
		client: mgr.GetClient(),
		argoDB: argoDB,
		logger: logger,
	}
	t.promoteImageFn = t.promoteImage
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

// findTicketsForApplication returns reconciliation requests for all Tickets
// related to a given Argo CD Application. This takes advantage of an index
// established by SetupTicketReconcilerWithManager() and is used to propagate
// reconciliation requests to Tickets whose state should be affected by changes
// to relates Application resources.
func (t *ticketReconciler) findTicketsForApplication(
	application client.Object,
) []reconcile.Request {
	tickets := &api.TicketList{}
	if err := t.client.List(
		context.Background(),
		tickets,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				applicationsIndexField,
				application.GetName(),
			),
		},
	); err != nil {
		t.logger.WithFields(log.Fields{
			"application": application.GetName(),
		}).Error("error listing Tickets associated with Argo CD application")
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(tickets.Items))
	for i, item := range tickets.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (t *ticketReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	t.logger.WithFields(log.Fields{
		"name": req.NamespacedName.Name,
	}).Debug("reconciling Ticket")

	// No matter what happens, we're not requeueing
	result := ctrl.Result{}

	var ticket api.Ticket
	if err := t.client.Get(ctx, req.NamespacedName, &ticket); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"name": req.NamespacedName.Name,
			}).Warn("Ticket not found")
		} else {
			t.logger.WithFields(log.Fields{
				"name": req.NamespacedName.Name,
			}).Error("error getting Ticket")
		}
		return result, err
	}

	// What's the current state of the ticket?
	switch ticket.Status.State {
	case api.TicketStateNew:
	default:
		// We don't have anything to do in the current state
		return result, nil
	}

	// Find the associated Line
	line := api.Line{}
	if err := t.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: req.Namespace,
			Name:      ticket.Spec.Line,
		},
		&line,
	); err != nil {
		ticket.Status.State = api.TicketStateFailed
		if err = client.IgnoreNotFound(err); err == nil {
			ticket.Status.StateReason = fmt.Sprintf(
				"Line %s does not exist",
				ticket.Spec.Line,
			)
			t.logger.WithFields(log.Fields{
				"ticket": ticket.Name,
				"line":   ticket.Spec.Line,
			}).Warn("No Line found for Ticket")
		} else {
			ticket.Status.StateReason = fmt.Sprintf(
				"Error getting Line %s",
				ticket.Spec.Line,
			)
			t.logger.WithFields(log.Fields{
				"ticket": ticket.Name,
				"line":   ticket.Spec.Line,
			}).Errorf("Error getting line for Ticket: %s", err)
		}
		if err = t.client.Status().Update(ctx, &ticket); err != nil {
			t.logger.WithFields(log.Fields{
				"name":        ticket.Name,
				"state":       ticket.Status.State,
				"stateReason": ticket.Status.StateReason,
			}).Errorf("Error updating Ticket status: %s", err)
		}
		return result, err
	}

	// What's the zero environment?
	if len(line.Environments) == 0 {
		// This Ticket is implicitly complete
		ticket.Status.State = api.TicketStateCompleted
		ticket.Status.StateReason =
			"Associated Line has no environments; Nothing to do"
		err := t.client.Status().Update(ctx, &ticket)
		if err != nil {
			t.logger.WithFields(log.Fields{
				"name":        ticket.Name,
				"state":       ticket.Status.State,
				"stateReason": ticket.Status.StateReason,
			}).Errorf("Error updating Ticket status: %s", err)
		}
		return result, err
	}
	env := line.Environments[0]

	// Find the associated Argo CD Application
	app := argocd.Application{}
	if err := t.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: req.Namespace,
			Name:      env,
		},
		&app,
	); err != nil {
		ticket.Status.State = api.TicketStateFailed
		if err = client.IgnoreNotFound(err); err == nil {
			ticket.Status.StateReason = fmt.Sprintf(
				"Argo CD Application %s does not exist",
				env,
			)
			t.logger.WithFields(log.Fields{
				"ticket":      ticket.Name,
				"line":        ticket.Spec.Line,
				"environment": env,
			}).Warn("No Argo CD Application found for environment")
		} else {
			ticket.Status.StateReason = fmt.Sprintf(
				"Error getting Argo CD Application for environment %s",
				env,
			)
			t.logger.WithFields(log.Fields{
				"ticket":      ticket.Name,
				"line":        ticket.Spec.Line,
				"environment": env,
			}).Errorf("Error getting Argo CD Application for environment: %s", err)
		}
		if err = t.client.Status().Update(ctx, &ticket); err != nil {
			t.logger.WithFields(log.Fields{
				"name":        ticket.Name,
				"state":       ticket.Status.State,
				"stateReason": ticket.Status.StateReason,
			}).Errorf("Error updating Ticket status: %s", err)
		}
		return result, err
	}

	// Now see what this Application tells us about how to proceed with applying
	// the change represented by the Ticket. e.g. What repo and branch do we
	// commit to?
	gitopsRepoURL := app.Spec.Source.RepoURL
	envBranch := app.Spec.Source.TargetRevision

	// Promote
	commitSHA, err := t.promoteImageFn(
		ctx,
		ticket.Spec.Change.ImageRepo,
		ticket.Spec.Change.ImageTag,
		gitopsRepoURL,
		envBranch,
	)
	if err != nil {
		ticket.Status.State = api.TicketStateFailed
		ticket.Status.StateReason = fmt.Sprintf(
			"Error promoting image to environment %s",
			env,
		)
		t.logger.WithFields(log.Fields{
			"ticket":           ticket.Name,
			"line":             ticket.Spec.Line,
			"environment":      env,
			"imageRepo":        ticket.Spec.Change.ImageRepo,
			"imageTag":         ticket.Spec.Change.ImageTag,
			"gitopsRepoURL":    gitopsRepoURL,
			"gitopsRepoBranch": envBranch,
		}).Errorf("Error promoting image: %s", err)
		if err = t.client.Status().Update(ctx, &ticket); err != nil {
			t.logger.WithFields(log.Fields{
				"name":        ticket.Name,
				"state":       ticket.Status.State,
				"stateReason": ticket.Status.StateReason,
			}).Errorf("Error updating Ticket status: %s", err)
		}
		return result, nil
	}

	t.logger.WithFields(log.Fields{
		"ticket":           ticket.Name,
		"line":             ticket.Spec.Line,
		"environment":      env,
		"imageRepo":        ticket.Spec.Change.ImageRepo,
		"imageTag":         ticket.Spec.Change.ImageTag,
		"gitopsRepoURL":    gitopsRepoURL,
		"gitopsRepoBranch": envBranch,
		"gitopsRepoCommit": commitSHA,
	}).Debug("promoted image")

	ticket.Status.State = api.TicketStateProgressing
	ticket.Status.StateReason = fmt.Sprintf(
		"Image has been promoted to environment %s",
		env,
	)
	if err = t.client.Status().Update(ctx, &ticket); err != nil {
		t.logger.WithFields(log.Fields{
			"name":        ticket.Name,
			"state":       ticket.Status.State,
			"stateReason": ticket.Status.StateReason,
		}).Errorf("Error updating Ticket status: %s", err)
	}
	return result, err
}
