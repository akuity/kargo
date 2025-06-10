package builtin

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// helmChartUpdater is an implementation of the promotion.StepRunner interface
// that updates the dependencies of a Helm chart.
type helmChartUpdater struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmChartUpdater returns an implementation of the promotion.StepRunner
// interface that updates the dependencies of a Helm chart.
func newHelmChartUpdater(credsDB credentials.Database) promotion.StepRunner {
	r := &helmChartUpdater{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *helmChartUpdater) Name() string {
	return "helm-update-chart"
}

// Run implements the promotion.StepRunner interface.
func (h *helmChartUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if err := h.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.HelmUpdateChartConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.run(ctx, stepCtx, cfg)
}

// validate validates helmChartUpdater configuration against a JSON schema.
func (h *helmChartUpdater) validate(cfg promotion.Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *helmChartUpdater) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmUpdateChartConfig,
) (promotion.StepResult, error) {
	manager, err := helm.NewEphemeralDependencyManager(h.credsDB, stepCtx.Project, stepCtx.WorkDir)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create Helm dependency manager: %w", err)
	}

	updates := make([]helm.ChartDependency, 0, len(cfg.Charts))
	for _, chart := range cfg.Charts {
		updates = append(updates, helm.ChartDependency{
			Repository: chart.Repository,
			Name:       chart.Name,
			Version:    chart.Version,
		})
	}
	newVersions, err := manager.Update(ctx, cfg.Path, updates...)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if commitMsg := h.generateCommitMessage(cfg.Path, newVersions); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (h *helmChartUpdater) generateCommitMessage(chartPath string, newVersions map[string]string) string {
	if len(newVersions) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString("Updated chart dependencies for ")
	_, _ = commitMsg.WriteString(chartPath)
	_, _ = commitMsg.WriteString("\n")
	for name, change := range newVersions {
		if change == "" {
			change = "removed"
		}
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %s", name, change))
	}
	return commitMsg.String()
}

func compareChartVersions(before, after []helm.ChartDependency) map[string]string {
	beforeMap := make(map[string]string, len(before))
	for _, dep := range before {
		beforeMap[dep.Name] = dep.Version
	}

	changes := make(map[string]string)
	for _, dep := range after {
		if oldVersion, exists := beforeMap[dep.Name]; exists {
			if oldVersion != dep.Version {
				changes[dep.Name] = oldVersion + " -> " + dep.Version
			}
			// Remove the dependency from before map to track allow remaining
			// items to be counted as removed
			delete(beforeMap, dep.Name)
		} else {
			changes[dep.Name] = dep.Version
		}
	}

	// Handle any removed dependencies which are still listed in before map
	for name := range beforeMap {
		changes[name] = ""
	}
	return changes
}

// nameForRepositoryURL generates an SHA-256 hash of the repository URL to use
// as the name for the repository in the Helm repository cache.
//
// The repository URL is normalized before hashing using the same logic as
// urlutil.Equal from Helm, which is used to compare repository URLs in the
// download manager when looking at cached repository indexes to find the
// correct chart URL.
func nameForRepositoryURL(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		repoURL = filepath.Clean(repoURL)
	}

	if u != nil {
		if u.Path == "" {
			u.Path = "/"
		}
		u.Path = filepath.Clean(u.Path)
		repoURL = u.String()
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(repoURL)))
}
