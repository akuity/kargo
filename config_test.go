package main

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/akuityio/k8sta/internal/common/config"
)

func TestK8sTAConfig(t *testing.T) {
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
			name: "ARGOCD_NAMESPACE not set",
			setup: func() {
				t.Setenv("LOG_LEVEL", "INFO")
			},
			assertions: func(_ config.Config, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "ARGOCD_NAMESPACE")
			},
		},
		{
			name: "K8STA_NAMESPACE not set",
			setup: func() {
				t.Setenv("ARGOCD_NAMESPACE", "argocd")
			},
			assertions: func(_ config.Config, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "K8STA_NAMESPACE")
			},
		},
		{
			name: "success",
			setup: func() {
				t.Setenv("K8STA_NAMESPACE", "k8sta")
			},
			assertions: func(config config.Config, err error) {
				require.NoError(t, err)
				require.Equal(t, log.InfoLevel, config.LogLevel)
				require.Equal(t, "argocd", config.ArgoCDNamespace)
				require.Equal(t, "k8sta", config.K8sTANamespace)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			config, err := k8staConfig()
			testCase.assertions(config, err)
		})
	}
}
