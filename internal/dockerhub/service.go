package dockerhub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-playground/webhooks/v6/docker"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/akuityio/k8sta/internal/scratch"
)

// TODO: This is duplicated in a few places. Fix that!
const LabelKeyComponent = "akuity.io/k8sta-component"

// Service is an interface for components that can handle webhooks (events) from
// Docker Hub. Implementations of this interface are transport-agnostic.
type Service interface {
	// Handle handles a webhook (event) from Docker Hub.
	Handle(context.Context, docker.BuildPayload) error
}

type service struct {
	config     scratch.Config
	kubeClient kubernetes.Interface
	logger     *log.Logger
}

// NewService returns an implementation of the Service interface for handling
// webhooks (events) from Docker Hub.
func NewService(
	config scratch.Config,
	kubeClient kubernetes.Interface,
) Service {
	s := &service{
		config:     config,
		kubeClient: kubeClient,
		logger:     log.New(),
	}
	s.logger.SetLevel(log.DebugLevel) // TODO: Make this configurable
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

	line, ok := s.config.GetLineByImageRepository(repo)
	if !ok {
		s.logger.WithFields(log.Fields{
			"repo": repo,
		}).Debug("No line is subscribed to this image repository; nothing to do")
		return nil
	}

	s.logger.WithFields(log.Fields{
		"repo": repo,
		"line": line.Name,
	}).Debug("A line is subscribed to this image repository")

	ticket := scratch.Ticket{
		// TODO: UUID seems sensible for now, but we may find a better option as
		// we move forward.
		ID:        uuid.NewV4().String(),
		Source:    "Docker Hub",
		Namespace: line.Namespace,
		Line:      line.Name,
		Change: scratch.Change{
			Type:  "NewImage",
			Image: fmt.Sprintf("%s:%s", repo, tag),
		},
	}
	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return errors.Wrapf(err, "error marshaling Ticket %s to JSON", ticket.ID)
	}
	if _, err := s.kubeClient.CoreV1().ConfigMaps(line.Namespace).Create(
		ctx,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: line.Namespace,
				Name:      ticket.ID,
				Labels: map[string]string{
					LabelKeyComponent: "ticket",
				},
			},
			Data: map[string]string{
				"ticket": string(ticketBytes),
			},
		},
		metav1.CreateOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error creating ConfigMap for Ticket %s",
			ticket.ID)

	}

	s.logger.WithFields(log.Fields{
		"namespace": ticket.Namespace,
		"name":      ticket.ID,
		"line":      ticket.Line,
		"image":     ticket.Change.Image,
	}).Debug("Created Ticket (ConfigMap) resource")

	return nil
}
