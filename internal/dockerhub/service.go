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

	// Find subscribed Tracks
	tracks, err := s.getTracksByImageRepo(ctx, repo)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Tracks subscribed to image repo %s",
			repo,
		)
	}

	for _, track := range tracks {
		s.logger.WithFields(log.Fields{
			"repo":      repo,
			"name":      track.Name,
			"namespace": track.Namespace,
		}).Debug("A track is subscribed to this image repository")

		ticket := api.Ticket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewV4().String(),
				Namespace: track.Namespace,
			},
			Track: track.Name,
			Change: api.Change{
				NewImages: &api.NewImagesChange{
					Images: []api.Image{
						{
							Repo: repo,
							Tag:  tag,
						},
					},
				},
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
			"track":     ticket.Track,
			"namespace": ticket.Namespace,
			"imageRepo": repo,
			"imageTag":  tag,
		}).Debug("Created Ticket resource")
	}

	return nil
}

// TODO: Can we make use of an index here?
func (s *service) getTracksByImageRepo(
	ctx context.Context,
	repo string,
) ([]api.Track, error) {
	subscribedTracks := []api.Track{}
	tracks := api.TrackList{}
	if err := s.controllerRuntimeClient.List(ctx, &tracks); err != nil {
		return subscribedTracks, errors.Wrap(err, "error retrieving Tracks")
	}
tracks:
	for _, track := range tracks.Items {
		if track.Spec.Disabled {
			continue tracks
		}
		for _, sub := range track.Spec.ImageRepositorySubscriptions {
			if sub.RepoURL == repo {
				subscribedTracks = append(subscribedTracks, track)
				continue tracks
			}
		}
	}
	return subscribedTracks, nil
}
