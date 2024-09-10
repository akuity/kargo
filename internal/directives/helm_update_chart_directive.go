package directives

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

func init() {
	// Register the helm-update-chart directive with the builtins registry.
	builtins.RegisterDirective(newHelmUpdateChartDirective(), &DirectivePermissions{
		AllowArgoCDClient:  true,
		AllowCredentialsDB: true,
	})
}

type helmUpdateChartDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmUpdateChartDirective creates a new helm-update-image directive.
func newHelmUpdateChartDirective() Directive {
	d := &helmUpdateChartDirective{}
	d.schemaLoader = getConfigSchemaLoader(d.Name())
	return d
}

// Name implements the Directive interface.
func (d *helmUpdateChartDirective) Name() string {
	return "helm-update-chart"
}

// Run implements the Directive interface.
func (d *helmUpdateChartDirective) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	failure := Result{Status: StatusFailure}

	// Validate the configuration against the JSON Schema
	if err := validate(
		d.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		d.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := configToStruct[HelmUpdateChartConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.run(ctx, stepCtx, cfg)
}

func (d *helmUpdateChartDirective) run(
	ctx context.Context,
	stepCtx *StepContext,
	cfg HelmUpdateChartConfig,
) (Result, error) {
	absChartPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("failed to join path %q: %w", cfg.Path, err)
	}

	chartFilePath := filepath.Join(absChartPath, "Chart.yaml")
	chartDependencies, err := loadChartDependencies(chartFilePath)
	if err != nil {
		return Result{
			Status: StatusFailure,
		}, fmt.Errorf("failed to load chart dependencies from %q: %w", chartFilePath, err)
	}

	changes, err := d.processChartUpdates(ctx, stepCtx, cfg, chartDependencies)
	if err != nil {
		return Result{Status: StatusFailure}, err
	}

	if err = intyaml.SetStringsInFile(chartFilePath, changes); err != nil {
		return Result{
			Status: StatusFailure,
		}, fmt.Errorf("failed to update chart dependencies in %q: %w", chartFilePath, err)
	}

	helmHome, err := os.MkdirTemp("", "helm-chart-update-")
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("failed to create temporary Helm home directory: %w", err)
	}
	defer os.RemoveAll(helmHome)

	if err := d.updateDependencies(ctx, stepCtx, helmHome, absChartPath, chartDependencies); err != nil {
		return Result{Status: StatusFailure}, err
	}

	return Result{Status: StatusSuccess}, nil
}

func (d *helmUpdateChartDirective) processChartUpdates(
	ctx context.Context,
	stepCtx *StepContext,
	cfg HelmUpdateChartConfig,
	chartDependencies []chartDependency,
) (map[string]string, error) {
	changes := make(map[string]string)
	for _, update := range cfg.Charts {
		repoURL, chartName := normalizeChartReference(update.Repository, update.Name)

		var desiredOrigin *kargoapi.FreightOrigin
		if update.FromOrigin != nil {
			desiredOrigin = &kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKind(update.FromOrigin.Kind),
				Name: update.FromOrigin.Name,
			}
		}

		chart, err := freight.FindChart(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			repoURL,
			chartName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find chart: %w", err)
		}
		if chart == nil {
			continue
		}

		for i, dep := range chartDependencies {
			if dep.Repository == update.Repository && dep.Name == update.Name {
				changes[fmt.Sprintf("dependencies.%d.version", i)] = chart.Version
				break
			}
		}
	}
	return changes, nil
}

func (d *helmUpdateChartDirective) updateDependencies(
	ctx context.Context,
	stepCtx *StepContext,
	helmHome, chartPath string,
	chartDependencies []chartDependency,
) error {
	registryClient, err := helm.NewRegistryClient(helmHome)
	if err != nil {
		return fmt.Errorf("failed to create Helm registry client: %w", err)
	}

	repositoryFile := repo.NewFile()

	if err = d.loadDependencyCredentials(
		ctx,
		stepCtx.CredentialsDB,
		registryClient,
		repositoryFile,
		stepCtx.Project,
		chartDependencies,
	); err != nil {
		return err
	}

	repositoryConfig := filepath.Join(helmHome, "repositories.yaml")
	if err = repositoryFile.WriteFile(repositoryConfig, 0o600); err != nil {
		return fmt.Errorf("failed to write Helm repositories file: %w", err)
	}

	manager := downloader.Manager{
		Out:              io.Discard,
		ChartPath:        chartPath,
		Verify:           downloader.VerifyNever,
		SkipUpdate:       false,
		Getters:          getter.All(&cli.EnvSettings{}),
		RegistryClient:   registryClient,
		RepositoryConfig: repositoryConfig,
		RepositoryCache:  filepath.Join(helmHome, "cache"),
	}
	if err = manager.Update(); err != nil {
		return fmt.Errorf("failed to update chart dependencies: %w", err)
	}

	return nil
}

func (d *helmUpdateChartDirective) loadDependencyCredentials(
	ctx context.Context,
	credentialsDB credentials.Database,
	registryClient *registry.Client,
	repositoryFile *repo.File,
	project string,
	dependencies []chartDependency,
) error {
	for _, dep := range dependencies {
		var credType credentials.Type
		var credURL string

		switch {
		case strings.HasPrefix(dep.Repository, "https://"):
			credType = credentials.TypeHelm
			credURL = dep.Repository
		case strings.HasPrefix(dep.Repository, "oci://"):
			credType = credentials.TypeHelm
			credURL = "oci://" + path.Join(helm.NormalizeChartRepositoryURL(dep.Repository), dep.Name)
		default:
			continue
		}

		creds, ok, err := credentialsDB.Get(ctx, project, credType, credURL)
		if err != nil {
			return fmt.Errorf("failed to obtain credentials for chart repository %q: %w", dep.Repository, err)
		}
		if !ok {
			continue
		}

		if strings.HasPrefix(dep.Repository, "https://") {
			repositoryFile.Add(&repo.Entry{
				Name:     dep.Name,
				URL:      dep.Repository,
				Username: creds.Username,
				Password: creds.Password,
			})
		} else {
			if err = registryClient.Login(
				strings.TrimPrefix(dep.Repository, "oci://"),
				registry.LoginOptBasicAuth(creds.Username, creds.Password),
			); err != nil {
				return fmt.Errorf("failed to authenticate with chart repository %q: %w", dep.Repository, err)
			}
		}
	}
	return nil
}

func normalizeChartReference(repoURL, chartName string) (string, string) {
	if strings.HasPrefix(repoURL, "oci://") {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(repoURL, "/"), chartName), ""
	}
	return repoURL, chartName
}

type chartDependency struct {
	Repository string `json:"repository,omitempty"`
	Name       string `json:"name,omitempty"`
}

func loadChartDependencies(chartPath string) ([]chartDependency, error) {
	b, err := os.ReadFile(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", chartPath, err)
	}

	chartObj := struct {
		Dependencies []chartDependency `json:"dependencies,omitempty"`
	}{}
	if err := yaml.Unmarshal(b, &chartObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q: %w", chartPath, err)
	}

	return chartObj.Dependencies, nil
}
