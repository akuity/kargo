package builtin

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// kustomizeValidatorFS is a filesys.FileSystem that inspects every
// kustomization file read through it and rejects entries with a query
// parameter value that git could interpret as an option, as well as Helm chart
// names and release names that helm could interpret as an option.
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
// Kustomize's Helm chart inflation runs the helm binary, passing the chart
// name and release name as bare arguments. Those settings are validated too,
// both in kustomization files and in HelmChartInflationGenerator
// configurations, which Kustomize also reads through this filesys.FileSystem.
// See validateHelmCharts.
//
// Kustomize treats any error reading a kustomization file as the file being
// absent, which would obscure the reason for the failure. The first validation
// error is therefore also recorded, so that callers can report it via Err.
type kustomizeValidatorFS struct {
	filesys.FileSystem
	err error
}

// newKustomizeValidatorFS wraps the given filesys.FileSystem so that
// kustomization files containing entries with unsafe query parameter values,
// or Helm chart settings that helm could interpret as an option, cannot be
// read through it.
func newKustomizeValidatorFS(fs filesys.FileSystem) *kustomizeValidatorFS {
	return &kustomizeValidatorFS{FileSystem: fs}
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
		// like the equivalent kustomization fields. The file may hold several
		// documents.
		if err = validateHelmInflationGenerators(path, content); err != nil {
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
	// FixKustomization has merged the deprecated helmChartInflationGenerator
	// field into HelmCharts.
	if err = validateHelmCharts(path, k.HelmCharts); err != nil {
		return nil, r.record(err)
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

// helmInflationGeneratorKind is the kind of a Kustomize generator configuration
// that inflates a Helm chart.
const helmInflationGeneratorKind = "HelmChartInflationGenerator"

// helmInflationGeneratorConfig is the subset of a HelmChartInflationGenerator
// configuration that is passed to helm as a bare argument.
type helmInflationGeneratorConfig struct {
	Kind string `json:"kind"`
	kustypes.HelmChart
}

// validateHelmInflationGenerators validates every HelmChartInflationGenerator
// configuration among the YAML documents in content, read from the file at
// path. Documents that are not YAML or not of that kind are ignored.
func validateHelmInflationGenerators(path string, content []byte) error {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(content)))
	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return nil
		}
		var cfg helmInflationGeneratorConfig
		if err = yaml.Unmarshal(doc, &cfg); err != nil || cfg.Kind != helmInflationGeneratorKind {
			continue
		}
		if err = validateHelmCharts(path, []kustypes.HelmChart{cfg.HelmChart}); err != nil {
			return err
		}
	}
}

// validateHelmCharts returns an error if any of the given Helm charts, read
// from the file at path, has a chart name or release name that helm could
// interpret as an option. Kustomize rejects these itself from
// sigs.k8s.io/kustomize/api v0.21.2, but Kargo pins an earlier version.
func validateHelmCharts(path string, charts []kustypes.HelmChart) error {
	for _, chart := range charts {
		// Kustomize runs helm via exec, not a shell, passing the chart name and
		// release name each as a single bare argument. Helm's flag parser
		// treats an argument as a flag only if it starts with '-', so a value
		// starting with '-' (e.g. --post-renderer=sh) can make helm run
		// arbitrary commands. A value with '-' anywhere else (e.g.
		// "foo --post-renderer=sh") stays a single positional argument and
		// cannot inject a flag. Leading whitespace is trimmed before the check
		// so that its safety does not depend on how helm treats such values.
		if strings.HasPrefix(strings.TrimSpace(chart.Name), "-") {
			return fmt.Errorf("error validating %s: chart name %q must not start with '-'", path, chart.Name)
		}
		if strings.HasPrefix(strings.TrimSpace(chart.ReleaseName), "-") {
			return fmt.Errorf(
				"error validating %s: release name %q must not start with '-'",
				path, chart.ReleaseName,
			)
		}
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
