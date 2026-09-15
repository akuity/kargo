//nolint:forcetypeassert
package utils

import (
	"context"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

// GetProjectConfig retrieves the single ProjectConfig resource from a project's
// namespace and returns it as a kargoapi.ProjectConfig.
func GetProjectConfig(ctx context.Context, project string) (*kargoapi.ProjectConfig, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	cfg, httpRes, err := kargoClient.CoreAPI.GetProjectConfig(ctx, project).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil && httpRes.StatusCode == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting project config: %w", err)
	}

	projectConfig := &kargoapi.ProjectConfig{}
	if err := convertGeneratedResource(cfg, projectConfig); err != nil {
		return nil, err
	}
	return projectConfig, nil
}

// UpdateProjectConfig updates the ProjectConfig resource in a project's
// namespace to match projectConfig.
func UpdateProjectConfig(
	ctx context.Context,
	projectConfig *kargoapi.ProjectConfig,
) error {
	projectConfig.Kind = "ProjectConfig"
	projectConfig.APIVersion = kargoapi.GroupVersion.String()
	return updateKargoResource(ctx, projectConfig, 1)
}

// GetClusterConfig retrieves the single ClusterConfig resource and returns it as
// a kargoapi.ClusterConfig.
func GetClusterConfig(ctx context.Context) (*kargoapi.ClusterConfig, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	cfg, httpRes, err := kargoClient.SystemAPI.GetClusterConfig(ctx).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil && httpRes.StatusCode == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting cluster config: %w", err)
	}

	clusterConfig := &kargoapi.ClusterConfig{}
	if err := convertGeneratedResource(cfg, clusterConfig); err != nil {
		return nil, err
	}
	return clusterConfig, nil
}

// UpdateClusterConfig updates the ClusterConfig resource to match clusterConfig.
func UpdateClusterConfig(ctx context.Context, clusterConfig *kargoapi.ClusterConfig) error {
	clusterConfig.Kind = "ClusterConfig"
	clusterConfig.APIVersion = kargoapi.GroupVersion.String()
	return updateKargoResource(ctx, clusterConfig, 1)
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
func updateKargoResource(ctx context.Context, obj any, depth int) error {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)
	manifest, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error encoding resource manifest: %w", err)
	}

	_, httpRes, err := kargoClient.ResourcesAPI.
		UpdateResource(ctx).
		Manifest(string(manifest)).
		Upsert(true).
		Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if httpRes.StatusCode == 409 {
		// Retry on conflict
		if depth < 10 {
			fmt.Printf("Update resource retry")
			return updateKargoResource(ctx, obj, depth+1)
		}
	}
	if err != nil {
		return fmt.Errorf("error updating kargo resource: %w, %v", err, httpRes)
	}

	return nil
}
