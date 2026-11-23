package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/logging"
)

func TestExternalWebhooksServerOptionsRun(t *testing.T) {
	t.Parallel()
	opts := &externalWebhooksServerOptions{
		Logger: logging.NewDiscardLoggerOrDie(),
	}
	require.ErrorContains(t, opts.run(t.Context()), "API_SERVER_ADDRESS")
}
