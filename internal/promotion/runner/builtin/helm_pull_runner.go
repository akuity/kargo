package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// helmPullRunner is an implementation of the promotion.StepRunner interface
// that pulls Helm charts from repositories and extracts them to specified paths.
type helmPullRunner struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmPullRunner returns an implementation of the promotion.StepRunner
// interface that pulls Helm charts from repositories.
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
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	// Ensure the base path exists
	absBasePath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, fmt.Errorf("failed to join path %q: %w", cfg.Path, err)
	}

	if err := os.MkdirAll(absBasePath, 0o755); err != nil {
		return failure, fmt.Errorf("failed to create directory %q: %w", absBasePath, err)
	}

	// Create a temporary directory for Helm operations
	tempDir, err := os.MkdirTemp("", "helm-pull-*")
	if err != nil {
		return failure, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up Helm environment
	settings := cli.New()
	settings.RepositoryCache = filepath.Join(tempDir, "cache")
	settings.RepositoryConfig = filepath.Join(tempDir, "repositories.yaml")

	// Create registry client
	registryClient, err := helm.NewRegistryClient(tempDir)
	if err != nil {
		return failure, fmt.Errorf("failed to create registry client: %w", err)
	}

	// Process each chart
	for _, chart := range cfg.Charts {
		if err := h.pullChart(ctx, stepCtx, settings, registryClient, absBasePath, chart); err != nil {
			return failure, fmt.Errorf("failed to pull chart %q: %w", chart.Name, err)
		}
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (h *helmPullRunner) pullChart(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	settings *cli.EnvSettings,
	registryClient *action.RegistryClient,
	basePath string,
	chart builtin.HelmPullChart,
) error {
	// Get credentials for the repository
	creds, err := h.getCredentials(ctx, stepCtx.Project, chart.Repository)
	if err != nil {
		return fmt.Errorf("failed to get credentials for repository %q: %w", chart.Repository, err)
	}

	// Create the output directory for this chart
	chartOutPath, err := securejoin.SecureJoin(basePath, chart.OutPath)
	if err != nil {
		return fmt.Errorf("failed to join chart output path %q: %w", chart.OutPath, err)
	}

	if err := os.MkdirAll(chartOutPath, 0o755); err != nil {
		return fmt.Errorf("failed to create chart output directory %q: %w", chartOutPath, err)
	}

	// Create pull action
	client := action.NewPullWithOpts(action.PullOpts{
		Settings: settings,
	})
	client.DestDir = chartOutPath
	client.Untar = true
	client.UntarDir = chartOutPath
	client.Version = chart.Version

	// Set up authentication if credentials are available
	if creds != nil {
		if err := registryClient.Login(
			chart.Repository,
			action.LoginOpts{
				Username: creds.Username,
				Password: creds.Password,
			},
		); err != nil {
			return fmt.Errorf("failed to login to registry %q: %w", chart.Repository, err)
		}
	}

	// Construct the chart reference
	chartRef := chart.Repository
	if !isOCIRepository(chart.Repository) {
		// For classic repositories, append the chart name
		chartRef = fmt.Sprintf("%s/%s", chart.Repository, chart.Name)
	}

	// Pull the chart
	_, err = client.Run(chartRef)
	if err != nil {
		return fmt.Errorf("failed to pull chart %q from %q: %w", chart.Name, chart.Repository, err)
	}

	return nil
}

func (h *helmPullRunner) getCredentials(
	ctx context.Context,
	project string,
	repoURL string,
) (*helm.Credentials, error) {
	if h.credsDB == nil {
		return nil, nil
	}

	creds, found, err := h.credsDB.Get(
		ctx,
		project,
		credentials.TypeHelmChart,
		helm.NormalizeChartRepositoryURL(repoURL),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting credentials for chart repository %q: %w", repoURL, err)
	}
	if !found {
		return nil, nil
	}

	return &helm.Credentials{
		Username: creds.Username,
		Password: creds.Password,
	}, nil
}

// isOCIRepository returns true if the repository URL is an OCI repository.
func isOCIRepository(repoURL string) bool {
	return len(repoURL) > 6 && repoURL[:6] == "oci://"
}
