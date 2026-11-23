package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/logging"
)

func TestControllerOptionsRun(t *testing.T) {
	t.Setenv("CREDENTIAL_PROVIDERS_ENABLED", "true")
	opts := &controllerOptions{Logger: logging.NewDiscardLoggerOrDie()}
	require.ErrorContains(t, opts.run(t.Context()), "API_SERVER_ADDRESS")
}
