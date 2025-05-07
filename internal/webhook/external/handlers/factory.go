package handlers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/logging"
)

type Factory struct {
	client.Client
	log *logging.Logger
}

func NewFactory(kClient client.Client) *Factory {
	return &Factory{
		Client: kClient,
		log:    logging.NewLogger(logging.InfoLevel),
	}
}
