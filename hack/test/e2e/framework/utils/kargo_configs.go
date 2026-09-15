package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

// GetProjectConfig retrieves the single ProjectConfig resource from a project's
// namespace and returns it as a kargoapi.ProjectConfig.
func GetProjectConfig(ctx context.Context, project string) (kargoapi.ProjectConfig, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	cfg, httpRes, err := kargoClient.CoreAPI.GetProjectConfig(ctx, project).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		return kargoapi.ProjectConfig{}, fmt.Errorf("error getting project config: %w", err)
	}

	var projectConfig kargoapi.ProjectConfig
	if err := convertGeneratedResource(cfg, &projectConfig); err != nil {
		return kargoapi.ProjectConfig{}, err
	}
	return projectConfig, nil
}

// UpdateProjectConfig updates the ProjectConfig resource in a project's
// namespace to match projectConfig.
func UpdateProjectConfig(
	ctx context.Context,
	projectConfig kargoapi.ProjectConfig,
) error {
	return updateKargoResource(ctx, &projectConfig)
}

// GetClusterConfig retrieves the single ClusterConfig resource and returns it as
// a kargoapi.ClusterConfig.
func GetClusterConfig(ctx context.Context) (kargoapi.ClusterConfig, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	cfg, httpRes, err := kargoClient.SystemAPI.GetClusterConfig(ctx).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		return kargoapi.ClusterConfig{}, fmt.Errorf("error getting cluster config: %w", err)
	}

	var clusterConfig kargoapi.ClusterConfig
	if err := convertGeneratedResource(cfg, &clusterConfig); err != nil {
		return kargoapi.ClusterConfig{}, err
	}
	return clusterConfig, nil
}

// UpdateClusterConfig updates the ClusterConfig resource to match clusterConfig.
func UpdateClusterConfig(ctx context.Context, clusterConfig kargoapi.ClusterConfig) error {
	return updateKargoResource(ctx, &clusterConfig)
}

// convertGeneratedResource converts a generated API model into the equivalent
// kargoapi type by round-tripping through JSON, which both representations
// serialize identically.
func convertGeneratedResource(from any, to any) error {
	data, err := json.Marshal(from)
	if err != nil {
		return fmt.Errorf("error encoding resource: %w", err)
	}
	if err := json.Unmarshal(data, to); err != nil {
		return fmt.Errorf("error decoding resource: %w", err)
	}
	return nil
}

// updateKargoResource marshals obj to a YAML manifest and applies it via the
// resources API, returning any per-resource errors reported by the server.
func updateKargoResource(ctx context.Context, obj any) error {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	manifest, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error encoding resource manifest: %w", err)
	}

	res, httpRes, err := kargoClient.ResourcesAPI.
		UpdateResource(ctx).
		Manifest(string(manifest)).
		Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("error updating kargo resource: %w", err)
	}

	updateErrs := make([]error, 0, len(res.Results))
	for _, r := range res.Results {
		if r.Error != nil {
			updateErrs = append(updateErrs, errors.New(*r.Error))
		}
	}
	return errors.Join(updateErrs...)
}
