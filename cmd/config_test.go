package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/akuityio/kargo/internal/config"
)

func TestKargoConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(config.Config, error)
	}{
		{
			name: "LOG_LEVEL invalid",
			setup: func() {
				t.Setenv("LOG_LEVEL", "BOGUS")
			},
			assertions: func(_ config.Config, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not a valid logrus Level")
			},
		},
		{
			name: "success",
			setup: func() {
				t.Setenv("LOG_LEVEL", "INFO")
			},
			assertions: func(config config.Config, err error) {
				require.NoError(t, err)
				require.Equal(t, log.InfoLevel, config.LogLevel)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			config, err := kargoConfig()
			testCase.assertions(config, err)
		})
	}
}
