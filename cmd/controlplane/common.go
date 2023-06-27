package main

import (
	"context"

	"github.com/akuity/kargo/internal/logging"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getRestConfig(ctx context.Context, path string) (*rest.Config, error) {
	logger := logging.LoggerFromContext(ctx)

	// clientcmd.BuildConfigFromFlags will fall back on in-cluster config if path
	// is empty, but will issue a warning that we can suppress by checking for
	// that condition ourselves and calling rest.InClusterConfig() directly.
	if path == "" {
		logger.Debug("loading in-cluster REST config")
		cfg, err := rest.InClusterConfig()
		return cfg, errors.Wrap(err, "error loading in-cluster REST config")
	}

	logger.WithField("path", path).Debug("loading REST config from path")
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	return cfg, errors.Wrapf(err, "error loading REST config from %q", path)
}
