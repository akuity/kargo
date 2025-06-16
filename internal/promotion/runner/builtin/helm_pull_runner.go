package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// helmPullRunner is an implementation of the promotion.StepRunner interface
// that pulls/downloads a Helm chart from a repository and extracts it to a
// specified directory.
type helmPullRunner struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmPullRunner returns an implementation of the promotion.StepRunner
// interface that pulls/downloads a Helm chart from a repository.
func newHelmPullRunner(credsDB credentials.Database) promotion.StepRunner {
	r := &helmPullRunner{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *helmPullRunner) Name() string {
	return "helm-pull"
}

// Run implements the promotion.StepRunner interface.
func (h *helmPullRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	// Validate the configuration against the JSON Schema
	if err := validate(
		h.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		h.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.HelmPullConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.run(ctx, stepCtx, cfg)
}

func (h *helmPullRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmPullConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	// Determine chart details from freight or configuration
	chartRepoURL, chartName, chartVersion, err := h.getChartDetails(stepCtx, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to determine chart details: %w", err)
	}

	logger.Debug("pulling Helm chart",
		"repoURL", chartRepoURL,
		"chart", chartName,
		"version", chartVersion,
	)

	// Secure join the output path
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create output directory %q: %w", outPath, err)
	}

	// Create a temporary Helm home directory
	helmHome, err := os.MkdirTemp("", "helm-pull-*")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create temporary Helm home: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(helmHome); removeErr != nil {
			logger.Error(removeErr, "failed to remove temporary Helm home directory")
		}
	}()

	// Set up Helm environment
	env := cli.New()
	env.RepositoryConfig = filepath.Join(helmHome, "repositories.yaml")
	env.RepositoryCache = filepath.Join(helmHome, "repository")

	// Create registry client for OCI repositories
	registryClient, err := helm.NewRegistryClient(helmHome)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create Helm registry client: %w", err)
	}

	// Set up credentials if needed
	if err := h.setupCredentials(ctx, stepCtx.Project, chartRepoURL, chartName, registryClient); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to setup credentials: %w", err)
	}

	// Create pull action
	pullAction := action.NewPull()
	pullAction.Settings = env
	pullAction.DestDir = outPath
	pullAction.Untar = true
	pullAction.UntarDir = outPath
	pullAction.Version = chartVersion

	// Configure action based on repository type
	if strings.HasPrefix(chartRepoURL, "oci://") {
		// For OCI repositories, the chart reference includes the chart name
		chartRef := fmt.Sprintf("%s/%s", strings.TrimPrefix(chartRepoURL, "oci://"), chartName)
		if chartVersion != "" {
			chartRef = fmt.Sprintf("%s:%s", chartRef, chartVersion)
		}
		
		// Use registry client for OCI
		pullAction.SetRegistryClient(registryClient)
		
		// Pull the chart
		_, err = pullAction.Run(chartRef)
	} else {
		// For classic repositories, add the repository first
		repoEntry := &action.ChartPathOptions{
			RepoURL: chartRepoURL,
		}
		
		// Configure getters
		pullAction.Getters = getter.All(env)
		
		// Build chart reference
		chartRef := chartName
		if chartRepoURL != "" {
			chartRef = fmt.Sprintf("%s/%s", chartRepoURL, chartName)
		}
		
		// Set chart path options
		pullAction.ChartPathOptions = *repoEntry
		
		// Pull the chart
		_, err = pullAction.Run(chartRef)
	}

	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to pull chart: %w", err)
	}

	logger.Debug("successfully pulled Helm chart", "outPath", outPath)

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// getChartDetails determines the chart repository URL, name, and version from
// either the freight or the configuration.
func (h *helmPullRunner) getChartDetails(
	stepCtx *promotion.StepContext,
	cfg builtin.HelmPullConfig,
) (string, string, string, error) {
	// If chart details are explicitly provided in config, use them
	if cfg.Chart != nil {
		return cfg.Chart.RepoURL, cfg.Chart.Name, cfg.Chart.Version, nil
	}

	// Otherwise, try to find chart from freight
	if cfg.ChartFromFreight != nil {
		repoURL := cfg.ChartFromFreight.RepoURL
		
		// Find matching chart in freight
		for _, freightRef := range stepCtx.Freight.References() {
			for _, chart := range freightRef.Charts {
				if chart.RepoURL == repoURL {
					// If name is specified, it must match
					if cfg.ChartFromFreight.Name != "" && chart.Name != cfg.ChartFromFreight.Name {
						continue
					}
					return chart.RepoURL, chart.Name, chart.Version, nil
				}
			}
		}
		
		return "", "", "", fmt.Errorf("chart not found in freight for repository %q", repoURL)
	}

	return "", "", "", fmt.Errorf("either 'chart' or 'chartFromFreight' must be specified")
}

// setupCredentials configures authentication for the chart repository.
func (h *helmPullRunner) setupCredentials(
	ctx context.Context,
	project string,
	repoURL string,
	chartName string,
	registryClient *registry.Client,
) error {
	// Determine the credential URL based on repository type
	var credURL string
	if strings.HasPrefix(repoURL, "oci://") {
		// For OCI repositories, include the chart name in the credential URL
		credURL = fmt.Sprintf("oci://%s/%s", 
			strings.TrimPrefix(repoURL, "oci://"), 
			chartName)
	} else {
		credURL = repoURL
	}

	// Get credentials from the database
	creds, err := h.credsDB.Get(ctx, project, credentials.TypeHelm, credURL)
	if err != nil {
		return fmt.Errorf("failed to get credentials for %q: %w", credURL, err)
	}

	// If no credentials found, that's okay - the repository might be public
	if creds == nil {
		return nil
	}

	// Set up authentication for OCI repositories
	if strings.HasPrefix(repoURL, "oci://") && registryClient != nil {
		registryHost := strings.TrimPrefix(repoURL, "oci://")
		if err := registryClient.Login(
			registryHost,
			registry.LoginOptBasicAuth(creds.Username, creds.Password),
		); err != nil {
			return fmt.Errorf("failed to login to OCI registry %q: %w", registryHost, err)
		}
	}

	return nil
}
