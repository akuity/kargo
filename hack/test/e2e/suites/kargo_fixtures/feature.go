//nolint:forcetypeassert
package kargo_fixtures

import (
	"context"
	"embed"
	"slices"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

func init() {
	utils.TestFeatures = append(utils.TestFeatures, feature())
}

var (
	//go:embed testdata/*
	TestData embed.FS
)

func feature() features.Feature {
	feature := features.New("Example kargo fixtures")
	project := "kargo-fixtures"

	// This setup step is necessary to use this feature as a part of shared package test
	// It sets the path to look up the fixtures files.
	feature.Setup(utils.TestData(TestData))
	feature.Setup(utils.SetupKargoClients)
	feature.Setup(utils.SetupKargoFixtures)
	feature.Teardown(utils.TeardownKargoFixtures)

	feature.Assess("fixture project is created",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			kargoClient := ctx.Value(utils.KargoCLIKey).(generated.APIClient)
			// FIXME: do we need to pass client.Options?
			// FIXME: move this to helper functions package?
			res, httpRes, err := kargoClient.CoreAPI.ListProjects(ctx).Execute()
			if httpRes != nil {
				_ = httpRes.Body.Close()
			}
			if err != nil {
				t.Fatalf("list projects: %v", err)
			}
			projects := res.Items
			index := slices.IndexFunc(projects, func(proj generated.Project) bool {
				return *proj.Metadata.Name == project
			})
			if index < 0 {
				t.Fatalf("cannot find project `%s`", project)
			}
			return ctx
		})

	feature.Assess("fixture warehouse is created",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			kargoClient := ctx.Value(utils.KargoCLIKey).(generated.APIClient)
			// FIXME: do we need to pass client.Options?
			// FIXME: move this to helper functions package?
			res, httpRes, err := kargoClient.CoreAPI.ListWarehouses(ctx, project).Execute()
			if httpRes != nil {
				_ = httpRes.Body.Close()
			}
			if err != nil {
				t.Fatalf("list warehouses: %v", err)
			}
			warehouses := res.Items
			index := slices.IndexFunc(warehouses, func(warehouse generated.Warehouse) bool {
				return *warehouse.Metadata.Name == "images"
			})
			if index < 0 {
				t.Fatalf("cannot find warehouse `images`")
			}
			return ctx
		})

	return feature.Feature()
}
