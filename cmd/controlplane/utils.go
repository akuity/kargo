package main

import (
	"context"
	"fmt"
	stdos "os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/telemetry"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

// telemetryShutdownTimeout bounds how long a component waits at shutdown for
// spans not yet exported to be flushed.
const telemetryShutdownTimeout = 10 * time.Second

// setupTelemetry configures tracing for the named component according to the
// environment and returns a function that flushes and shuts tracing down. The
// returned function never fails; a problem flushing at shutdown is logged.
// When tracing is disabled, it is a no-op.
func setupTelemetry(
	ctx context.Context,
	logger *logging.Logger,
	component string,
	attrs ...attribute.KeyValue,
) (func(), error) {
	cfg := telemetry.ConfigFromEnv()
	shutdown, err := telemetry.Setup(
		logging.ContextWithLogger(ctx, logger),
		cfg,
		telemetry.Service{
			Name:       "kargo-" + component,
			Version:    versionpkg.GetVersion().Version,
			Attributes: attrs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing tracing: %w", err)
	}
	if cfg.Enabled {
		logger.Info(
			"tracing is enabled",
			"protocol", cfg.EffectiveTracesProtocol(),
		)
	}
	return func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			telemetryShutdownTimeout,
		)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logger.Error(err, "error shutting down tracing")
		}
	}, nil
}

func argoCDExists(
	ctx context.Context,
	restCfg *rest.Config,
	namespace string,
) (bool, error) {
	c, err := dynamic.NewForConfig(restCfg)
	if err == nil {
		if _, err = c.Resource(
			schema.GroupVersionResource{
				Group:    "argoproj.io",
				Version:  "v1alpha1",
				Resource: "applications",
			},
		).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1}); err == nil {
			return true, nil
		}
	}
	return false, client.IgnoreNotFound(err)
}

func argoRolloutsExists(ctx context.Context, restCfg *rest.Config) (bool, error) {
	c, err := dynamic.NewForConfig(restCfg)
	if err == nil {
		if _, err = c.Resource(
			schema.GroupVersionResource{
				Group:    "argoproj.io",
				Version:  "v1alpha1",
				Resource: "analysistemplates",
			},
		).List(ctx, metav1.ListOptions{Limit: 1}); err == nil {
			return true, nil
		}
	}
	return false, client.IgnoreNotFound(err)
}

func getLogVars() (logging.Level, logging.Format) {
	logLevelStr := os.GetEnv(logging.LogLevelEnvVar, "info")
	logLevel, err := logging.ParseLevel(logLevelStr)
	if err != nil {
		fmt.Fprintf(
			stdos.Stderr,
			"invalid LOG_LEVEL %q, defaulting to info: %v\n",
			logLevelStr,
			err,
		)
		logLevel = logging.InfoLevel
	}

	logFormatStr := os.GetEnv(logging.LogFormatEnvVar, string(logging.DefaultFormat))
	logFormat, err := logging.ParseFormat(logFormatStr)
	if err != nil {
		fmt.Fprintf(
			stdos.Stderr,
			"invalid LOG_FORMAT %q, defaulting to %q: %v\n",
			logFormatStr,
			logging.DefaultFormat,
			err,
		)
		logFormat = logging.DefaultFormat
	}
	return logLevel, logFormat
}
