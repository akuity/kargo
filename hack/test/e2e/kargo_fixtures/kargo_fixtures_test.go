//go:build e2e
//nolint:forcetypeassert
package kargo_example

import (
	"context"
	"slices"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/akuity/kargo/hack/test/e2e/utils"
	"github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
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
			kargoClient := ctx.Value(utils.KargoCLIKey).(generated.KargoAPI)
			// FIXME: do we need to pass client.Options?
			res, err := kargoClient.Core.ListProjects(
				core.NewListProjectsParams(),
				nil,
			)
			if err != nil {
				t.Fatalf("list projects: %v", err)
			}
			projects := res.Payload.Items
			index := slices.IndexFunc(projects, func(proj *models.Project) bool {
				return proj.Metadata.Name == project
			})
			if index < 0 {
				t.Fatalf("cannot find project `%s`", project)
			}
			return ctx
		})

	feature.Assess("fixture warehouse is created",
		func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			kargoClient := ctx.Value(utils.KargoCLIKey).(generated.KargoAPI)
			// FIXME: do we need to pass client.Options?
			res, err := kargoClient.Core.ListWarehouses(
				core.NewListWarehousesParams().WithProject(project),
				nil,
			)
			if err != nil {
				t.Fatalf("list warehouses: %v", err)
			}
			warehouses := res.Payload.Items
			index := slices.IndexFunc(warehouses, func(warehouse *models.Warehouse) bool {
				return warehouse.Metadata.Name == "images"
			})
			if index < 0 {
				t.Fatalf("cannot find warehouse `images`")
			}
			return ctx
		})

	utils.TestEnv.Test(t, feature.Feature())
}
