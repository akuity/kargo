//go:build e2e && examples
//nolint:forcetypeassert
package kargo_example

// This test shows an example of using YAML files to define Kargo fixtures to use in tests.
// It sets up fixtures and verifies that they exist.

import (
	"context"
	"slices"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestKargoFixtures(t *testing.T) {
	feature := features.New("Example kargo fixtures")
	project := "kargo-fixtures"

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

	utils.TestEnv.Test(t, feature.Feature())
}
