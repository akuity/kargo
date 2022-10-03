package bookkeeper

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/config"
)

// Service is an interface for components that can handle bookkeeping requests.
// Implementations of this interface are transport-agnostic.
type Service interface {
	// Handle handles a bookkeeping request.
	Handle(context.Context) error
}

type service struct {
	logger *log.Logger
}

// NewService returns an implementation of the Service interface for
// handling bookkeeping requests.
func NewService(config config.Config) Service {
	s := &service{
		logger: log.New(),
	}
	s.logger.SetLevel(config.LogLevel)
	return s
}

// TODO: Implement this
func (s *service) Handle(ctx context.Context) error {
	return nil
}
