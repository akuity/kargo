package builtin

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// kustomizeValidatorFS is a filesys.FileSystem that inspects every
// kustomization file read through it and rejects entries with a query
// parameter value that git could interpret as an option, as well as Helm
// chart inflation settings that would make Kustomize read or write outside
// the promotion's working directory.
//
// Kustomize resolves remote Git references by running the system git binary
// as `git fetch --depth=1 <repository> <ref>`, passing the ref (taken from the
// entry's ?ref= or ?version= query parameter) as a literal argument without an
// end-of-options separator. A ref such as --upload-pack=<cmd> therefore causes
// git to execute an arbitrary command.
//
// Kustomize reads every kustomization file -- including those inside
// repositories it has cloned to resolve remote entries -- through the
// filesys.FileSystem passed to it, and it always reads and parses a
// kustomization file before resolving any of its entries. Validating at this
// layer therefore rejects a malicious ref before git is ever invoked, and also
// covers remote entries referenced transitively by other remote entries. See
// validateQueryValues for which values are rejected.
//
// Kustomize's Helm chart inflation runs the helm binary on the chart home,
// which it derives from the kustomization file or generator configuration by
// path, outside the filesys.FileSystem it was given. Those settings are
// validated here too, for every kustomization and generator configuration
// Kustomize reads, so that the chart home and any configured config home
// resolve inside the working directory, or beneath the kustomization's own
// directory when Kustomize has cloned it from a remote. See
// validateHelmInflation.
//
// Kustomize treats any error reading a kustomization file as the file being
// absent, which would obscure the reason for the failure. The first validation
// error is therefore also recorded, so that callers can report it via Err.
type kustomizeValidatorFS struct {
	filesys.FileSystem
	root *os.Root
	// workDir is root's directory with symlinks resolved, which is how
	// Kustomize reports the paths it reads.
	workDir string
	err     error
}

// newKustomizeValidatorFS wraps the given filesys.FileSystem so that
// kustomization files containing entries with unsafe query parameter values or
// Helm chart inflation settings that reach outside root cannot be read through
// it.
func newKustomizeValidatorFS(fileSystem filesys.FileSystem, root *os.Root) (*kustomizeValidatorFS, error) {
	workDir, err := filepath.EvalSymlinks(root.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}
	return &kustomizeValidatorFS{FileSystem: fileSystem, root: root, workDir: workDir}, nil
}

// helmInflationGeneratorKind is the kind of a Kustomize generator configuration
// that inflates a Helm chart.
const helmInflationGeneratorKind = "HelmChartInflationGenerator"

// helmInflationGeneratorConfig is the subset of a HelmChartInflationGenerator
// configuration that names directories or files.
type helmInflationGeneratorConfig struct {
	Kind string `json:"kind"`
	kustypes.HelmGlobals
	kustypes.HelmChart
}

// Err returns the first validation error encountered by ReadFile, if any.
func (r *kustomizeValidatorFS) Err() error {
	return r.err
}

// ReadFile implements filesys.FileSystem.
func (r *kustomizeValidatorFS) ReadFile(path string) ([]byte, error) {
	content, err := r.FileSystem.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(
		konfig.RecognizedKustomizationFileNames(),
		filepath.Base(path),
	) {
		// A generator configuration that inflates a Helm chart is validated
		// like the equivalent kustomization fields.
		var cfg helmInflationGeneratorConfig
		if err = yaml.Unmarshal(content, &cfg); err != nil || cfg.Kind != helmInflationGeneratorKind {
			return content, nil
		}
		if err = r.validateHelmInflation(path, &cfg.HelmGlobals, []kustypes.HelmChart{cfg.HelmChart}); err != nil {
			return nil, r.record(err)
		}
		return content, nil
	}
	// Parse the kustomization exactly as Kustomize does. If it cannot be
	// parsed, Kustomize will fail to parse it as well and none of its entries
	// will be resolved, so it is safe to return the content as-is and let
	// Kustomize report the error.
	var k kustypes.Kustomization
	if err = k.Unmarshal(content); err != nil {
		return content, nil
	}
	// Merges deprecated fields (e.g. bases) into their replacements.
	k.FixKustomization()
	for _, entries := range [][]string{
		k.Resources,
		k.Components,
		k.Generators,
		k.Transformers,
		k.Validators,
	} {
		for _, entry := range entries {
			if err = validateQueryValues(entry); err != nil {
				return nil, r.record(fmt.Errorf("error validating %s: %w", path, err))
			}
		}
	}
	if len(k.HelmCharts) > 0 {
		if err = r.validateHelmInflation(path, k.HelmGlobals, k.HelmCharts); err != nil {
			return nil, r.record(err)
		}
	}
	return content, nil
}

// record records err as the first validation error, if none has been
// recorded yet, and returns it.
func (r *kustomizeValidatorFS) record(err error) error {
	if r.err == nil {
		r.err = err
	}
	return err
}

// validateHelmInflation returns an error if the Helm chart inflation settings
// read from the file at path would make Kustomize read or write outside the
// working directory: a chart home or config home that is absolute or resolves
// outside, a symlink beneath either that leads outside, or a chart name with a
// path separator, which Kustomize joins onto the chart home.
func (r *kustomizeValidatorFS) validateHelmInflation(
	path string,
	globals *kustypes.HelmGlobals,
	charts []kustypes.HelmChart,
) error {
	chartHome := kustypes.HelmDefaultHome
	var configHome string
	if globals != nil {
		if globals.ChartHome != "" {
			chartHome = globals.ChartHome
		}
		configHome = globals.ConfigHome
	}
	if err := r.validateHelmDir(filepath.Dir(path), chartHome); err != nil {
		return fmt.Errorf("error validating %s: chart home %w", path, err)
	}
	if configHome != "" {
		if err := r.validateHelmDir(filepath.Dir(path), configHome); err != nil {
			return fmt.Errorf("error validating %s: config home %w", path, err)
		}
	}
	for _, chart := range charts {
		if chart.Name == "" || chart.Name == "." || chart.Name == ".." || chart.Name != filepath.Base(chart.Name) {
			return fmt.Errorf("error validating %s: chart name %q must be a plain name", path, chart.Name)
		}
	}
	return nil
}

// validateHelmDir returns an error if dir, a directory a Helm chart inflation
// setting names relative to the directory kusDir of the kustomization, is
// absolute or does not resolve inside the working directory, or if anything
// beneath it does not. A kustomization outside the working directory is one
// Kustomize cloned from a remote, and may only name directories beneath its
// own.
func (r *kustomizeValidatorFS) validateHelmDir(kusDir, dir string) error {
	if filepath.IsAbs(dir) {
		return fmt.Errorf("%q must be relative", dir)
	}
	// Kustomize reports the directory with symlinks resolved, as workDir is.
	kusDir, err := filepath.EvalSymlinks(kusDir)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", kusDir, err)
	}
	base := r.workDir
	root := r.root
	if rel, relErr := filepath.Rel(r.workDir, kusDir); relErr != nil || !filepath.IsLocal(rel) {
		base = kusDir
		if root, err = os.OpenRoot(kusDir); err != nil {
			return fmt.Errorf("failed to open %q: %w", kusDir, err)
		}
		defer root.Close()
	}
	rel, err := filepath.Rel(base, filepath.Join(kusDir, dir))
	if err != nil || !filepath.IsLocal(rel) {
		return fmt.Errorf("%q resolves outside %s", dir, base)
	}
	if err = statTree(root, rel); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%q: %w", dir, err)
	}
	return nil
}

// validateQueryValues returns an error if the value of any query parameter of
// the given kustomization entry begins with "-".
//
// Kustomize currently passes only the ref (taken from the ref or version query
// parameter) to git, and a valid Git ref never begins with "-". Every
// parameter is checked nonetheless, so that a parameter Kustomize starts
// passing to git in the future cannot silently reintroduce option injection.
func validateQueryValues(entry string) error {
	_, query, ok := strings.Cut(entry, "?")
	if !ok {
		return nil
	}
	// A query that cannot be fully parsed still yields every parameter that
	// could be parsed. Those are checked too, rather than relying on Kustomize
	// ignoring malformed queries.
	values, _ := url.ParseQuery(query)
	for _, param := range slices.Sorted(maps.Keys(values)) {
		for _, value := range values[param] {
			if strings.HasPrefix(value, "-") {
				return fmt.Errorf(
					"entry %q specifies query parameter %s %q, which must not begin with %q",
					entry, param, value, "-",
				)
			}
		}
	}
	return nil
}
