package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/helm"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindHelmUpdateChart = "helm-update-chart"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindHelmUpdateChart,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newHelmChartUpdater,
		},
	)
}

// helmChartUpdater is an implementation of the promotion.StepRunner interface
// that updates the dependencies of a Helm chart.
type helmChartUpdater struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmChartUpdater returns an implementation of the promotion.StepRunner
// interface that updates the dependencies of a Helm chart.
func newHelmChartUpdater(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &helmChartUpdater{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindHelmUpdateChart),
	}
}

// Run implements the promotion.StepRunner interface.
func (h *helmChartUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := h.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return h.run(ctx, stepCtx, cfg)
}

// convert validates helmChartUpdater configuration against a JSON schema and
// converts it into a builtin.HelmUpdateChartConfig struct.
func (h *helmChartUpdater) convert(cfg promotion.Config) (builtin.HelmUpdateChartConfig, error) {
	return validateAndConvert[builtin.HelmUpdateChartConfig](h.schemaLoader, cfg, stepKindHelmUpdateChart)
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
