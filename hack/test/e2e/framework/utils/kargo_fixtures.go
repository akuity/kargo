package utils

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/yaml"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

const groupKargo = "kargo"
const KargoCLIKey envfuncs.ContextKey = "kargo_cli"
const KargoCLIWatchKey envfuncs.ContextKey = "kargo_watch"
const TestDataPath = "testdata"
const TestDataKey envfuncs.ContextKey = "test_data"

func SetupKargoClients(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	ctx = SetupKargoApiClient(ctx, t, cfg)
	return SetupKargoWatchClient(ctx, t, cfg)
}

func SetupKargoApiClient(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
	if _, ok := ctx.Value(KargoCLIKey).(generated.APIClient); ok {
		return ctx
	}

	if kargoConfig, ok := ctx.Value(envfuncs.KargoConfigKey).(config.CLIConfig); ok {
		kargoClient, err := client.GetClientFromConfig(ctx, kargoConfig, client.Options{})
		if err != nil {
			t.Fatalf("error loading kargo client: %v", err)
		}
		return context.WithValue(ctx, KargoCLIKey, *kargoClient)
	}

	t.Fatalf("error getting kargo_config from the context %v", ctx)
	return ctx
}

func SetupKargoWatchClient(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
	if _, ok := ctx.Value(KargoCLIWatchKey).(watch.Client); ok {
		return ctx
	}

	if kargoConfig, ok := ctx.Value(envfuncs.KargoConfigKey).(config.CLIConfig); ok {
		watchClient, err := client.GetWatchClientFromConfig(ctx, kargoConfig, client.Options{})
		if err != nil {
			t.Fatalf("error loading kargo watch client: %v", err)
		}
		return context.WithValue(ctx, KargoCLIWatchKey, *watchClient)
	}

	t.Fatalf("error getting kargo_config from the context %v", ctx)
	return ctx
}

func NewSetupKargoFixtures(options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return SetupKargoFixturesWithOptions(ctx, t, cfg, options...)
	}
}

func NewTeardownKargoFixtures(options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return TeardownKargoFixturesWithOptions(ctx, t, cfg, options...)
	}
}

func SetupKargoFixturesWithOptions(
	ctx context.Context,
	t *testing.T,
	_ *envconf.Config,
	options ...decoder.DecodeOption,
) context.Context {
	err := scanFixtures(ctx, groupKargo, sortAsc, KargoCreateHandler(), options...)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func SetupKargoFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return SetupKargoFixturesWithOptions(ctx, t, cfg)
}

func TeardownKargoFixturesWithOptions(
	ctx context.Context,
	t *testing.T,
	_ *envconf.Config,
	options ...decoder.DecodeOption,
) context.Context {
	// FIXME: test failure scenarios to assure cleanup
	err := scanFixtures(ctx, groupKargo, sortDesc, KargoDeleteHandler(), options...)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TeardownKargoFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return TeardownKargoFixturesWithOptions(ctx, t, cfg)
}

func TestData(testData fs.FS) features.Func {
	return func(ctx context.Context, _ *testing.T, _ *envconf.Config) context.Context {
		return context.WithValue(ctx, TestDataKey, testData)
	}
}

func scanFixtures(
	ctx context.Context,
	group string,
	sortFun func([]string) []string,
	handlerFun decoder.HandlerFunc,
	options ...decoder.DecodeOption,
) error {
	testData, ok := ctx.Value(TestDataKey).(fs.FS)
	if !ok {
		return fmt.Errorf("unable to get testdata from context")
	}

	fixturesDir := filepath.Join(TestDataPath, group)
	files, err := fs.Glob(testData, filepath.Join(fixturesDir, "*.yaml"))
	if err != nil {
		return err
	}

	files = sortFun(files)

	for _, file := range files {
		err := scanFile(ctx, testData, file, handlerFun, options...)
		if err != nil {
			return err
		}
	}

	return nil
}

func scanFile(
	ctx context.Context,
	testData fs.FS,
	fileName string,
	handlerFun decoder.HandlerFunc,
	options ...decoder.DecodeOption,
) error {
	f, err := testData.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	err = decoder.DecodeEach(ctx, f, handlerFun, options...)
	if err != nil {
		return err
	}
	return nil
}

func sortDesc(sorted []string) []string {
	slices.SortFunc(sorted, func(a, b string) int {
		// Descending order
		return -strings.Compare(a, b)
	})
	return sorted
}

func sortAsc(sorted []string) []string {
	slices.SortFunc(sorted, func(a, b string) int {
		// Ascending order
		return strings.Compare(a, b)
	})
	return sorted
}

func KargoCreateHandler() decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		kargoClient, ok := ctx.Value(KargoCLIKey).(generated.APIClient)
		if !ok {
			return fmt.Errorf("kargo_cli is required in context")
		}

		fmt.Printf("Create kargo resource %v : %v\n", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())

		manifest, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("error encoding kargo resource manifest: %w", err)
		}

		res, httpRes, err := kargoClient.ResourcesAPI.
			CreateResource(ctx).
			Manifest(string(manifest)).
			Execute()
		if httpRes != nil {
			defer httpRes.Body.Close()
		}

		if err != nil {
			return fmt.Errorf("error creating kargo resource %v-%v: %w",
				obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
		createErrs := make([]error, 0, len(res.Results))
		for _, r := range res.Results {
			if r.Error != nil {
				createErrs = append(createErrs, errors.New(*r.Error))
			}
		}
		if len(createErrs) > 0 {
			return errors.Join(createErrs...)
		}
		return nil
	}
}

func KargoDeleteHandler() decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		kargoClient, ok := ctx.Value(KargoCLIKey).(generated.APIClient)
		if !ok {
			return fmt.Errorf("kargo_cli is required in context")
		}

		manifest, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("error encoding kargo resource manifest: %w", err)
		}

		res, httpRes, err := kargoClient.ResourcesAPI.
			DeleteResource(ctx).
			Manifest(string(manifest)).
			Execute()
		if httpRes != nil {
			defer httpRes.Body.Close()
		}
		if err != nil {
			// Don't fail decode sequence on error
			fmt.Printf("error deleting kargo resource: %v", err)
			return nil
		}
		createErrs := make([]error, 0, len(res.Results))
		for _, r := range res.Results {
			if r.Error != nil {
				createErrs = append(createErrs, errors.New(*r.Error))
			}
		}
		if len(createErrs) > 0 {
			// Don't fail decode sequence on error
			fmt.Printf("errors deleting kargo resource: %v", errors.Join(createErrs...))
			return nil
		}
		return nil
	}
}
