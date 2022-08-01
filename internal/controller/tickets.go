package controller

import (
	"context"
	"fmt"
	"os/exec"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
)

// ticketReconciler reconciles a Ticket object
type ticketReconciler struct {
	config config.Config
	client client.Client
	argoDB db.ArgoDB
	logger *log.Logger
	// All of these internal functions are overridable for testing purposes
	promoteImageFn func(
		ctx context.Context,
		imageRepoName string,
		imageTag string,
		gitopsRepoURL string,
		envBranch string,
	) (string, error)
	execCommandFn func(*exec.Cmd) error
}

func SetupWithManager(
	config config.Config,
	mgr manager.Manager,
	argoDB db.ArgoDB,
) error {
	t := &ticketReconciler{
		config: config,
		client: mgr.GetClient(),
		argoDB: argoDB,
		logger: log.New(),
	}
	t.logger.SetLevel(config.LogLevel)
	t.promoteImageFn = t.promoteImage
	t.execCommandFn = t.execCommand
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Ticket{}).WithEventFilter(predicate.Funcs{
		DeleteFunc: func(event.DeleteEvent) bool {
			// We're not interested in any deletes
			return false
		},
	}).Complete(t)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (t *ticketReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// No matter what happens, we're not requeueing
	result := ctrl.Result{}

	var ticket api.Ticket
	if err := t.client.Get(ctx, req.NamespacedName, &ticket); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			t.logger.WithFields(log.Fields{
				"name": req.NamespacedName.Name,
			}).Warn("ticket not found")
		} else {
			t.logger.WithFields(log.Fields{
				"name": req.NamespacedName.Name,
			}).Error("error getting ticket")
		}
		return result, err
	}

	// Do not attempt to further reconcile the Ticket if it is being deleted.
	// TODO: Do we really need this here given that we're filtering out deletes
	// using a predicate?
	if ticket.DeletionTimestamp != nil {
		return result, nil
	}

	// What's the current state of the ticket?
	switch ticket.Status.State {
	// TODO: Undo this change
	case api.TicketStateNew, "":
	default:
		// We don't have anything to do in the current state
		return result, nil
	}

	// Find the associated Line
	line := api.Line{}
	if err := t.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: t.config.Namespace,
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
				"ticketName": ticket.Name,
				"lineName":   ticket.Spec.Line,
			}).Warn("No Line found for Ticket")
		} else {
			ticket.Status.StateReason = fmt.Sprintf(
				"Error getting Line %s",
				ticket.Spec.Line,
			)
			t.logger.WithFields(log.Fields{
				"ticketName": ticket.Name,
				"lineName":   ticket.Spec.Line,
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
			Namespace: t.config.Namespace,
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
				"ticketName":  ticket.Name,
				"lineName":    ticket.Spec.Line,
				"environment": env,
			}).Warn("No Argo CD Application found for environment")
		} else {
			ticket.Status.StateReason = fmt.Sprintf(
				"Error getting Argo CD Application for environment %s",
				env,
			)
			t.logger.WithFields(log.Fields{
				"ticketName":  ticket.Name,
				"lineName":    ticket.Spec.Line,
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
			"ticketName":       ticket.Name,
			"lineName":         ticket.Spec.Line,
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
		"ticketName":       ticket.Name,
		"lineName":         ticket.Spec.Line,
		"environment":      env,
		"imageRepo":        ticket.Spec.Change.ImageRepo,
		"imageTag":         ticket.Spec.Change.ImageTag,
		"gitopsRepoURL":    gitopsRepoURL,
		"gitopsRepoBranch": envBranch,
		"gitopsRepoCommit": commitSHA,
	}).Debug("Promoted image")

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
