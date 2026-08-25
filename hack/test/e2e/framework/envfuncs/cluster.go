//nolint:forcetypeassert
package envfuncs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	fwenvfuncs "sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/utils"
	"sigs.k8s.io/e2e-framework/support"
	"sigs.k8s.io/e2e-framework/support/kind"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

// ClusterNameKey holds the name of the kind cluster created for the test run.
// Its presence signals that the run manages its own cluster; the install
// functions key off it and no-op when it is absent.
const ClusterNameKey ContextKey = "cluster_name"

// KargoHostKey holds a hostname used in KargoLogin. Populated by InstallKargo
const KargoHostKey ContextKey = "kargo_host"

// KargoPasswordKey holds a password used in KargoLogin. Populated by InstallKargo
const KargoPasswordKey ContextKey = "kargo_password"

const defaultClusterName = "kargo-e2e"

// ClusterSetupFuncs returns the ordered env setup functions that create a kind
// cluster and install cert-manager, Argo CD and Kargo via Helm. Each function
// no-ops when the env file does not configure a `cluster` section, so the slice
// is safe to include in a funcsloader unconditionally.
func ClusterSetupFuncs() []env.Func {
	return []env.Func{
		CreateKindCluster,
		InstallCertManager,
		InstallArgoCD,
		InstallArgoRollouts,
		InstallKargo,
	}
}

// ClusterTeardownFuncs returns the env finish functions that tear down the
// cluster created by ClusterSetupFuncs.
func ClusterTeardownFuncs() []env.Func {
	return []env.Func{DestroyKindCluster}
}

// CreateKindCluster creates a kind cluster named after the `cluster.name` env
// value (default "kargo-e2e"), optionally using a node image (`cluster.image`)
// and a kind config file (`cluster.config_file`). It no-ops when the env file
// has no `cluster` section, leaving runs that target an external cluster
// untouched. The cluster name is stored in the context under ClusterNameKey.
func CreateKindCluster(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if _, err := GetEnvMap(ctx, []string{"cluster"}); err != nil {
		fmt.Println("No `cluster` section in env; skipping kind cluster creation")
		return ctx, nil
	}

	name := optionalString(ctx, []string{"cluster", "name"}, defaultClusterName)
	image := optionalString(ctx, []string{"cluster", "image"}, "")
	configFile := optionalString(ctx, []string{"cluster", "config_file"}, "")

	var opts []support.ClusterOpts
	if image != "" {
		opts = append(opts, kind.WithImage(image))
	}

	provider := kind.NewProvider()

	tempdir := ctx.Value(TmpDirKey)
	if tempdir == nil {
		return ctx, fmt.Errorf("Temp dir is not set up. Cannot create kubeconfig")
	}

	kubeconfig := filepath.Join(tempdir.(string), "kubeconfig.yaml")
	// NOTE: e2e framework does not support configuring --kubeconfig for kind in its helpers.
	// We set KUBECONFIG here for kind command to pick up.
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfig)
	defer func() {
		if oldKubeconfig != "" {
			os.Setenv("KUBECONFIG", oldKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}()

	create := fwenvfuncs.CreateClusterWithOpts(provider, name, opts...)
	if configFile != "" {
		create = fwenvfuncs.CreateClusterWithConfig(provider, name, expandHome(configFile), opts...)
	}

	fmt.Printf("Creating kind cluster %q with config %v\n", name, configFile)
	ctx, err := create(ctx, cfg)
	if err != nil {
		return ctx, fmt.Errorf("creating kind cluster %q: %w", name, err)
	}

	fmt.Printf("Kubeconfig file for tests: %v\n", cfg.KubeconfigFile())
	return context.WithValue(ctx, ClusterNameKey, name), nil
}

// DestroyKindCluster deletes the cluster created by CreateKindCluster. It
// no-ops when no managed cluster is present in the context.
func DestroyKindCluster(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	name, ok := managedClusterName(ctx)
	if !ok {
		return ctx, nil
	}
	fmt.Printf("Destroying kind cluster %q\n", name)
	return fwenvfuncs.DestroyCluster(name)(ctx, cfg)
}

// InstallCertManager installs cert-manager (a prerequisite for Kargo's
// self-signed certificates) via Helm. It no-ops when no managed cluster is
// present in the context.
func InstallCertManager(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if _, ok := managedClusterName(ctx); !ok {
		return ctx, nil
	}
	if _, err := GetEnvMap(ctx, []string{"cert_manager"}); err != nil {
		fmt.Println("No `cert_manager` section in env; skipping cert_manager installation")
		return ctx, nil
	}
	chart := helmChart{
		releaseName: optionalString(ctx, []string{"cert_manager", "release_name"}, "cert-manager"),
		chart:       optionalString(ctx, []string{"cert_manager", "chart"}, "jetstack/cert-manager"),
		namespace:   optionalString(ctx, []string{"cert_manager", "namespace"}, "cert-manager"),
		version:     optionalString(ctx, []string{"cert_manager", "version"}, ""),
		repoName:    optionalString(ctx, []string{"cert_manager", "chart_repo_name"}, "jetstack"),
		repoURL:     optionalString(ctx, []string{"cert_manager", "chart_repo_url"}, "https://charts.jetstack.io"),
		timeout:     optionalString(ctx, []string{"cert_manager", "timeout"}, "10m"),
		valuesFiles: expandHomeAll(optionalStringSlice(ctx, []string{"cert_manager", "values_files"})),
		setValues:   defaultStrings(optionalStringSlice(ctx, []string{"cert_manager", "set"}), "crds.enabled=true"),
	}
	fmt.Println("Installing cert-manager")
	if err := chart.install(cfg.KubeconfigFile()); err != nil {
		return ctx, fmt.Errorf("installing cert-manager: %w", err)
	}
	return ctx, nil
}

// InstallArgoCD installs the Argo CD Helm chart. It no-ops when no managed
// cluster is present in the context.
func InstallArgoCD(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if _, ok := managedClusterName(ctx); !ok {
		return ctx, nil
	}
	if _, err := GetEnvMap(ctx, []string{"argocd"}); err != nil {
		fmt.Println("No `argocd` section in env; skipping argocd installation")
		return ctx, nil
	}
	chart := helmChart{
		releaseName: optionalString(ctx, []string{"argocd", "release_name"}, "argocd"),
		chart:       optionalString(ctx, []string{"argocd", "chart"}, "argo/argo-cd"),
		namespace:   optionalString(ctx, []string{"argocd", "namespace"}, "argocd"),
		version:     optionalString(ctx, []string{"argocd", "version"}, ""),
		repoName:    optionalString(ctx, []string{"argocd", "chart_repo_name"}, "argo"),
		repoURL:     optionalString(ctx, []string{"argocd", "chart_repo_url"}, "https://argoproj.github.io/argo-helm"),
		timeout:     optionalString(ctx, []string{"argocd", "timeout"}, "10m"),
		valuesFiles: expandHomeAll(optionalStringSlice(ctx, []string{"argocd", "values_files"})),
		setValues:   optionalStringSlice(ctx, []string{"argocd", "set"}),
	}
	fmt.Println("Installing Argo CD")
	if err := chart.install(cfg.KubeconfigFile()); err != nil {
		return ctx, fmt.Errorf("installing Argo CD: %w", err)
	}

	// kubectl port-forward svc/argocd-server -n argocd 8080:443
	// TODO: make the port configurable
	err := portForward(ctx, cfg.KubeconfigFile(), chart.namespace, "svc/argocd-server", 8080, 443)
	if err != nil {
		return ctx, fmt.Errorf("port-forwarding Argocd: %w", err)
	}
	// Port from portForward above
	ctx = context.WithValue(ctx, ArgocdHostKey, "localhost:8080")
	// Auth info.
	// FIXME: the values require setValues to have configs.secret.argocdServerAdminPassword set
	// Currently set in values.argocd.test.yaml
	ctx = context.WithValue(ctx, ArgocdUsernameKey, "admin")
	ctx = context.WithValue(ctx, ArgocdPasswordKey, "admin")

	return ctx, nil
}

// InstallKargo installs the Kargo Helm chart. When `kargo.image` is set, that
// image is first loaded into the kind cluster (useful for testing a locally
// built image). It no-ops when no managed cluster is present in the context.
func InstallKargo(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	name, ok := managedClusterName(ctx)
	if !ok {
		return ctx, nil
	}
	if _, err := GetEnvMap(ctx, []string{"kargo"}); err != nil {
		fmt.Println("No `kargo` section in env; skipping kargo installation")
		return ctx, nil
	}

	if image := optionalString(ctx, []string{"kargo", "image"}, ""); image != "" {
		fmt.Printf("Loading Kargo image %q into cluster %q\n", image, name)
		var err error
		if ctx, err = fwenvfuncs.LoadImageToCluster(name, image)(ctx, cfg); err != nil {
			return ctx, fmt.Errorf("loading Kargo image %q: %w", image, err)
		}
	}

	chart := helmChart{
		releaseName: optionalString(ctx, []string{"kargo", "release_name"}, "kargo"),
		chart:       optionalString(ctx, []string{"kargo", "chart"}, "oci://ghcr.io/akuity/kargo-charts/kargo"),
		namespace:   optionalString(ctx, []string{"kargo", "namespace"}, "kargo"),
		version:     optionalString(ctx, []string{"kargo", "version"}, ""),
		repoName:    optionalString(ctx, []string{"kargo", "chart_repo_name"}, ""),
		repoURL:     optionalString(ctx, []string{"kargo", "chart_repo_url"}, ""),
		timeout:     optionalString(ctx, []string{"kargo", "timeout"}, "10m"),
		valuesFiles: expandHomeAll(optionalStringSlice(ctx, []string{"kargo", "values_files"})),
		setValues:   optionalStringSlice(ctx, []string{"kargo", "set"}),
	}

	fmt.Println("Installing Kargo")
	if err := chart.install(cfg.KubeconfigFile()); err != nil {
		return ctx, fmt.Errorf("installing Kargo: %w", err)
	}
	// kubectl port-forward --namespace kargo svc/kargo-api 3000:80
	// TODO: make the port configurable
	err := portForward(ctx, cfg.KubeconfigFile(), chart.namespace, "svc/kargo-api", 3000, 80)
	if err != nil {
		return ctx, fmt.Errorf("port-forwarding Kargo API: %w", err)
	}
	// Port from portForward above
	ctx = context.WithValue(ctx, KargoHostKey, "http://localhost:3000")
	// FIXME: "admin" value requires passwordHash set in values
	// Currently set in values.test.yaml
	ctx = context.WithValue(ctx, KargoPasswordKey, "admin")

	return ctx, nil
}

// InstallArgoRollouts installs the Argo Rollouts Helm chart.
// It no-ops when no managed cluster is present in the context.
func InstallArgoRollouts(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	if _, ok := managedClusterName(ctx); !ok {
		return ctx, nil
	}
	if _, err := GetEnvMap(ctx, []string{"argo-rollouts"}); err != nil {
		fmt.Println("No `argo-rollouts` section in env; skipping argo rollouts installation")
		return ctx, nil
	}
	chart := helmChart{
		releaseName: optionalString(ctx, []string{"argo-rollouts", "release_name"}, "argo-rollouts"),
		chart:       optionalString(ctx, []string{"argo-rollouts", "chart"}, "argo/argo-rollouts"),
		namespace:   optionalString(ctx, []string{"argo-rollouts", "namespace"}, "argo-rollouts"),
		version:     optionalString(ctx, []string{"argo-rollouts", "version"}, ""),
		repoName:    optionalString(ctx, []string{"argo-rollouts", "chart_repo_name"}, "argo"),
		repoURL:     optionalString(ctx, []string{"argo-rollouts", "chart_repo_url"}, "https://argoproj.github.io/argo-helm"),
		timeout:     optionalString(ctx, []string{"argo-rollouts", "timeout"}, "10m"),
		valuesFiles: expandHomeAll(optionalStringSlice(ctx, []string{"argo-rollouts", "values_files"})),
		setValues:   optionalStringSlice(ctx, []string{"argo-rollouts", "set"}),
	}
	fmt.Println("Installing Argo Rollouts")
	if err := chart.install(cfg.KubeconfigFile()); err != nil {
		return ctx, fmt.Errorf("installing Argo Rollouts: %w", err)
	}

	return ctx, nil
}

func portForward(ctx context.Context, kubeconfig, namespace, service string, outport, inport int) error {
	// Run port-forward in background.
	// This is a simplified approach when we just run a background shell.
	// There is no error handling here, it might fail silently.
	// FIXME: replace that with goroutine and error channels?
	// FIXME: implement forwarding to an non-predefined port
	cmd := fmt.Sprintf("sh -c \"kubectl port-forward --kubeconfig %s --namespace %s %s %d:%d > /dev/null 2>&1 &\"",
		kubeconfig, namespace, service, outport, inport)

	fmt.Printf("Port forwarding %s to %d\n", service, outport)

	p := utils.RunCommandContext(ctx, cmd)
	if p.Err() != nil {
		outBytes, outErr := io.ReadAll(p.Out())
		if outErr != nil {
			return fmt.Errorf("kubectl: failed to port-forward: %w %w", p.Err(), outErr)
		}
		return fmt.Errorf("kubectl: failed to port-forward: %w : %s", p.Err(), outBytes)
	}
	return nil
}

// helmChart describes a Helm release to install into the cluster.
type helmChart struct {
	releaseName string
	chart       string
	namespace   string
	version     string
	// repoName and repoURL, when both set, cause `helm repo add`/`update` to run
	// before install. Leave empty for OCI or local-path charts.
	repoName    string
	repoURL     string
	timeout     string
	valuesFiles []string
	setValues   []string
}

// install performs an idempotent `helm upgrade --install` of the chart, adding
// its repository first when one is configured.
func (h helmChart) install(kubeconfig string) error {
	m := helm.New(kubeconfig)

	if h.repoName != "" && h.repoURL != "" {
		if err := m.RunRepo(helm.WithArgs("add", h.repoName, h.repoURL, "--force-update")); err != nil {
			return fmt.Errorf("helm repo add %q: %w", h.repoName, err)
		}
		if err := m.RunRepo(helm.WithArgs("update")); err != nil {
			return fmt.Errorf("helm repo update: %w", err)
		}
	}

	args := []string{"--install", "--create-namespace"}
	for _, valuesFile := range h.valuesFiles {
		args = append(args, "--values", valuesFile)
	}
	for _, setValue := range h.setValues {
		args = append(args, "--set", setValue)
	}

	opts := []helm.Option{
		helm.WithName(h.releaseName),
		helm.WithChart(h.chart),
		helm.WithNamespace(h.namespace),
		helm.WithArgs(args...),
		helm.WithWait(),
	}
	if h.version != "" {
		opts = append(opts, helm.WithVersion(h.version))
	}
	if h.timeout != "" {
		opts = append(opts, helm.WithTimeout(h.timeout))
	}
	return m.RunUpgrade(opts...)
}

func managedClusterName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(ClusterNameKey).(string)
	return name, ok && name != ""
}

func optionalString(ctx context.Context, path []string, def string) string {
	val, err := GetEnv(ctx, path)
	if err != nil {
		return def
	}
	if s, ok := val.(string); ok && s != "" {
		return s
	}
	return def
}

func optionalStringSlice(ctx context.Context, path []string) []string {
	val, err := GetEnv(ctx, path)
	if err != nil {
		return nil
	}
	items, ok := val.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// defaultStrings returns values when non-empty, otherwise the provided defaults.
func defaultStrings(values []string, defaults ...string) []string {
	if len(values) > 0 {
		return values
	}
	return defaults
}

func expandHomeAll(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = expandHome(p)
	}
	return out
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		return filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(path, "~"))
	}
	return path
}
