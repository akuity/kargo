package utils

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	"github.com/akuity/kargo/pkg/client/generated"
	kargoresouces "github.com/akuity/kargo/pkg/client/generated/resources"
	"github.com/akuity/kargo/pkg/client/watch"
)

const groupKargo = "kargo"
const KargoCLIKey envfuncs.ContextKey = "kargo_cli"
const KargoCLIWatchKey envfuncs.ContextKey = "kargo_watch"

func testFiles(testdata fs.FS) ([]string, error) {
	pattern := "*"
	files, err := fs.Glob(testdata, pattern)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func SetupKargoClients(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	ctx = SetupKargoApiClient(ctx, t, cfg)
	return SetupKargoWatchClient(ctx, t, cfg)
}

func SetupKargoApiClient(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
	if _, ok := ctx.Value(KargoCLIKey).(generated.KargoAPI); ok {
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

func scanFixtures(
	ctx context.Context,
	group string,
	sortFun func([]string) []string,
	handlerFun decoder.HandlerFunc,
	options ...decoder.DecodeOption) error {

	testdata := os.DirFS(filepath.Join("testdata", group))
	files, err := testFiles(testdata)
	if err != nil {
		return err
	}

	files = sortFun(files)

	for _, file := range files {
		err := scanFile(ctx, testdata, file, handlerFun, options...)
		if err != nil {
			return err
		}
	}

	return nil
}

func scanFile(
	ctx context.Context,
	testdata fs.FS,
	fileName string,
	handlerFun decoder.HandlerFunc,
	options ...decoder.DecodeOption,
) error {
	f, err := testdata.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	err = decoder.DecodeEach(ctx, f, handlerFun, options...)
	if err != nil {
		return err
	}
	return f.Close()
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
		kargoClient, ok := ctx.Value(KargoCLIKey).(generated.KargoAPI)
		if !ok {
			return fmt.Errorf("kargo_cli is required in context")
		}

		manifest, err := yaml.Marshal(obj)
		fmt.Printf("Creating resource: %v\n", obj.GetObjectKind())
		if err != nil {
			return fmt.Errorf("error encoding kargo resource manifest: %w", err)
		}
		res, err := kargoClient.Resources.CreateResource(
			kargoresouces.NewCreateResourceParams().
				WithManifest(string(manifest)),
			nil,
		)
		if err != nil {
			return fmt.Errorf("error creating kargo resource: %w", err)
		}
		createErrs := make([]error, 0, len(res.Payload.Results))
		for _, r := range res.Payload.Results {
			if r.Error != "" {
				createErrs = append(createErrs, errors.New(r.Error))
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
		kargoClient, ok := ctx.Value(KargoCLIKey).(generated.KargoAPI)
		if !ok {
			return fmt.Errorf("kargo_cli is required in context")
		}

		manifest, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("error encoding kargo resource manifest: %w", err)
		}
		res, err := kargoClient.Resources.DeleteResource(
			kargoresouces.NewDeleteResourceParams().
				WithManifest(string(manifest)),
			nil,
		)
		if err != nil {
			// Don't fail decode sequence on error
			fmt.Printf("error deleting kargo resource: %v", err)
			return nil
		}
		createErrs := make([]error, 0, len(res.Payload.Results))
		for _, r := range res.Payload.Results {
			if r.Error != "" {
				createErrs = append(createErrs, errors.New(r.Error))
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
