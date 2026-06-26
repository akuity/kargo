package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/yaml"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
)

const groupArgocd = "argocd"
const ArgoCDClientKey envfuncs.ContextKey = "argocd_client"

type ArgocdE2EClient struct {
	Path       string
	ConfigFile string
}

func (client ArgocdE2EClient) Create(obj k8s.Object) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	name := obj.GetName()
	unstr, ok := obj.(runtime.Unstructured)
	if !ok {
		return errors.New("object is not Unstructured")
	}
	switch kind {
	case "ApplicationSet":
		return client.CreateResource(unstr, "appset")
	case "Application":
		return client.CreateResource(unstr, "app", name, "--file")
	default:
		return fmt.Errorf("unsupported kind: %v", kind)
	}
}

func (client ArgocdE2EClient) Delete(kind, name string) error {
	switch kind {
	case "ApplicationSet":
		return client.DeleteResource("appset", name)
	case "Application":
		return client.DeleteResource("app", name)
	default:
		return fmt.Errorf("unsupported kind: %v", kind)
	}
}

func (client ArgocdE2EClient) CreateAppSet(obj runtime.Unstructured) error {
	return client.CreateResource(obj, "appset")
}

func (client ArgocdE2EClient) CreateResource(obj runtime.Unstructured, resourceType string, args ...string) error {

	configFile := client.ConfigFile

	// FIXME: use testing tempfile tools here
	tmpfile := "/tmp/manifest.yaml"

	manifest, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error encoding resource manifest: %w", err)
	}

	err = os.WriteFile(tmpfile, manifest, 0600)
	if err != nil {
		return fmt.Errorf("error writing manifest tempfile: %w", err)
	}

	// TODO we could use command builder, but it's not necessary now
	cmdArgs := []string{resourceType, "create"}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, tmpfile, fmt.Sprintf("--config=%v", configFile))

	//nolint:gosec
	cmd := exec.Command(client.Path, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Command output: %v", string(output))
		return fmt.Errorf("error running argocd command: %w", err)
	}
	fmt.Printf("argocd create output: %v", string(output))
	return nil
}

func (client ArgocdE2EClient) DeleteResource(resourceType, name string) error {
	//nolint:gosec
	cmd := exec.Command(client.Path, resourceType, "delete", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Command output: %v", string(output))
		return fmt.Errorf("error running argocd command: %w", err)
	}
	fmt.Printf("argocd create output: %v", string(output))
	return nil
}

func RequireArgoCDCLI(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
	argocdCLIPath := "argocd" // FIXME: configurable?
	_, err := exec.LookPath(argocdCLIPath)
	if err != nil {
		t.Fatalf("error locating argocd executable %s: %v", argocdCLIPath, err)
	}
	return ctx
}

func SetupArgocdClient(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	if _, ok := ctx.Value(ArgoCDClientKey).(ArgocdE2EClient); ok {
		return ctx
	}

	if config, ok := ctx.Value(envfuncs.ArgoCDConfigFile).(string); ok {
		ctx = RequireArgoCDCLI(ctx, t, cfg)
		argocdCLIPath := "argocd" // FIXME: configurable?
		return context.WithValue(ctx, ArgoCDClientKey, ArgocdE2EClient{Path: argocdCLIPath, ConfigFile: config})
	}

	t.Fatalf("error getting argocd config from the context %v", ctx)
	return ctx
}

func NewSetupArgoCDFixtures(options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return SetupArgoCDFixturesWithOptions(ctx, t, cfg, options...)
	}
}

func NewTeardownArgoCDFixtures(options ...decoder.DecodeOption) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return TeardownArgoCDFixturesWithOptions(ctx, t, cfg, options...)
	}
}

func SetupArgoCDFixturesWithOptions(
	ctx context.Context,
	t *testing.T,
	_ *envconf.Config,
	options ...decoder.DecodeOption,
) context.Context {
	err := scanFixtures(ctx, groupArgocd, sortAsc, ArgoCDCreateHandler(), options...)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func SetupArgoCDFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return SetupArgoCDFixturesWithOptions(ctx, t, cfg)
}

func TeardownArgoCDFixturesWithOptions(
	ctx context.Context,
	t *testing.T,
	_ *envconf.Config,
	options ...decoder.DecodeOption,
) context.Context {
	// FIXME: test failure scenarios to assure cleanup
	err := scanFixtures(ctx, groupArgocd, sortDesc, ArgoCDDeleteHandler(), options...)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TeardownArgoCDFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return TeardownArgoCDFixturesWithOptions(ctx, t, cfg)
}

func ArgoCDCreateHandler() decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		argoCDClient, ok := ctx.Value(ArgoCDClientKey).(ArgocdE2EClient)
		if !ok {
			return fmt.Errorf("argocd_client is required in context")
		}

		return argoCDClient.Create(obj)
	}
}

func ArgoCDDeleteHandler() decoder.HandlerFunc {
	return func(ctx context.Context, obj k8s.Object) error {
		argoCDClient, ok := ctx.Value(ArgoCDClientKey).(ArgocdE2EClient)
		if !ok {
			return fmt.Errorf("argocd_client is required in context")
		}
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return argoCDClient.Delete(kind, obj.GetName())
	}
}
