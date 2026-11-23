package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/logging"
)

func TestAPIOptionsRunRequiresDatabase(t *testing.T) {
	t.Parallel()
	o := &apiOptions{Logger: logging.NewDiscardLoggerOrDie()}
	err := o.run(t.Context())
	require.ErrorContains(t, err, "DATABASE_URL")
	require.ErrorContains(t, err, "DATABASE_HOST")
}
