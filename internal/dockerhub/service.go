package dockerhub

import (
	"context"

	"github.com/go-playground/webhooks/v6/docker"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
)

// Service is an interface for components that can handle webhooks (events) from
// Docker Hub. Implementations of this interface are transport-agnostic.
type Service interface {
	// Handle handles a webhook (event) from Docker Hub.
	Handle(context.Context, docker.BuildPayload) error
}

type service struct {
	config                  config.Config
	controllerRuntimeClient client.Client
	logger                  *log.Logger
}

// NewService returns an implementation of the Service interface for handling
// webhooks (events) from Docker Hub.
func NewService(
	config config.Config,
	controllerRuntimeClient client.Client,
) Service {
	s := &service{
		config:                  config,
		controllerRuntimeClient: controllerRuntimeClient,
		logger:                  log.New(),
	}
	s.logger.SetLevel(config.LogLevel)
	return s
}

func (s *service) Handle(
	ctx context.Context,
	payload docker.BuildPayload,
) error {
	repo := payload.Repository.RepoName
	tag := payload.PushData.Tag
	s.logger.WithFields(log.Fields{
		"repo": repo,
		"tag":  tag,
	}).Debug("An image was pushed to a Docker Hub image repository")

	// Find subscribed lines
	lines, err := s.getLinesByImageRepo(ctx, repo)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Lines subscribed to image repo %s",
			repo,
		)
	}

	for _, line := range lines {
		s.logger.WithFields(log.Fields{
			"repo": repo,
			"line": line.Name,
		}).Debug("A line is subscribed to this image repository")

		ticket := api.Ticket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewV4().String(),
				Namespace: s.config.K8sTANamespace,
			},
			Spec: api.TicketSpec{
				Line: line.Name,
				Change: api.Change{
					Type:      api.ChangeTypeNewImage,
					ImageRepo: repo,
					ImageTag:  tag,
				},
			},
			Status: api.TicketStatus{
				State:       api.TicketStateNew,
				StateReason: "This is a brand new Ticket",
			},
		}

		if err := s.controllerRuntimeClient.Create(ctx, &ticket); err != nil {
			return errors.Wrapf(
				err,
				"error creating Ticket %s",
				ticket.Name,
			)
		}

		s.logger.WithFields(log.Fields{
			"name":      ticket.Name,
			"line":      ticket.Spec.Line,
			"imageRepo": ticket.Spec.Change.ImageRepo,
			"imageTag":  ticket.Spec.Change.ImageTag,
		}).Debug("Created Ticket resource")
	}

	return nil
}

func (s *service) getLinesByImageRepo(
	ctx context.Context,
	repo string,
) ([]api.Line, error) {
	subscribedLines := []api.Line{}
	lines := api.LineList{}
	if err := s.controllerRuntimeClient.List(
		ctx, &lines,
		&client.ListOptions{
			Namespace: s.config.K8sTANamespace,
		},
	); err != nil {
		return subscribedLines, errors.Wrap(err, "error retrieving Lines")
	}
lines:
	for _, line := range lines.Items {
		for _, subscribedRepo := range line.ImageRepositories {
			if subscribedRepo == repo {
				subscribedLines = append(subscribedLines, line)
				continue lines
			}
		}
	}
	return subscribedLines, nil
}
