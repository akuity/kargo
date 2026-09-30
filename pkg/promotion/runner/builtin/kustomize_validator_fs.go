package builtin

import (
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// kustomizeValidatorFS is a filesys.FileSystem that inspects every
// kustomization file read through it and rejects entries with a query
// parameter value that git could interpret as an option.
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
// Kustomize treats any error reading a kustomization file as the file being
// absent, which would obscure the reason for the failure. The first validation
// error is therefore also recorded, so that callers can report it via Err.
type kustomizeValidatorFS struct {
	filesys.FileSystem
	err error
}

// newKustomizeValidatorFS wraps the given filesys.FileSystem so that
// kustomization files containing entries with unsafe query parameter values
// cannot be read through it.
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
				err = fmt.Errorf("error validating %s: %w", path, err)
				if r.err == nil {
					r.err = err
				}
				return nil, err
			}
		}
	}
	return content, nil
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
