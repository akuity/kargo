package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
)

// RemoteRef is a single ref reported by LsRemote.
type RemoteRef struct {
	// Name is the full ref name, e.g. "refs/heads/main" or "refs/tags/v1.2.3".
	Name string
	// ID is the object ID the ref points to. For an annotated tag this is the
	// tag object's ID, not the commit it dereferences to (the peeled entry is
	// dropped during parsing).
	ID string
}

// LsRemote lists refs in a remote Git repository using git ls-remote, WITHOUT
// cloning. It performs only the authentication setup required to reach the
// remote, issues a single round-trip, and cleans up after itself. The optional
// patterns restrict the listing to matching refs (e.g. "refs/heads/main",
// "refs/tags/*", or "HEAD" to resolve the default branch tip); when omitted,
// all refs are listed.
//
// For an annotated tag, the reported ID is the tag object's ID, not the commit
// it dereferences to: the peeled "^{}" entries that git emits for annotated
// tags are dropped (see parseLsRemoteOutput). This is immaterial to the
// intended use -- change detection -- because the tag object ID moves whenever
// the tag is re-pointed, and results are only ever compared against other
// LsRemote output produced the same way.
func LsRemote(
	repoURL string,
	clientOpts *ClientOptions,
	patterns ...string,
) ([]RemoteRef, error) {
	if clientOpts == nil {
		clientOpts = &ClientOptions{}
	}
	b := &baseRepo{
		creds:       clientOpts.Credentials,
		originalURL: repoURL,
		accessURL:   repoURL,
	}
	// setupDirs creates a temporary home directory that holds the ephemeral
	// client configuration (and any SSH key material) needed to authenticate.
	// There is no working tree or clone -- only the home directory -- so cleanup
	// is a single RemoveAll.
	if err := b.setupDirs(""); err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(b.homeDir)
	}()
	if err := b.setupClient(clientOpts); err != nil {
		return nil, err
	}

	// Note: --refs is intentionally NOT passed. It would exclude the symbolic
	// HEAD, which ListRefs queries to resolve a subscription's implicit default
	// branch. Annotated tags' peeled entries are instead dropped in
	// parseLsRemoteOutput.
	args := append([]string{"ls-remote", b.accessURL}, patterns...)
	cmd := b.buildGitCommand(args...)
	// Override the cmd.Dir that's set by buildGitCommand(). It's normally the
	// repository's path, which does not exist here because nothing was cloned.
	cmd.Dir = b.homeDir
	out, err := libExec.Exec(cmd)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing refs in remote repo %q: %w", b.originalURL, err,
		)
	}
	return parseLsRemoteOutput(out)
}

// parseLsRemoteOutput parses the output of git ls-remote, whose lines take the
// form "<object-id>\t<ref-name>".
func parseLsRemoteOutput(output []byte) ([]RemoteRef, error) {
	var refs []RemoteRef
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		id, name, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		// Drop the peeled "^{}" entry git emits for an annotated tag: it carries
		// the dereferenced commit, while the preceding line already carries the
		// tag object ID we record. Keeping both would yield two entries for one
		// tag name.
		if strings.HasSuffix(name, "^{}") {
			continue
		}
		refs = append(refs, RemoteRef{Name: name, ID: id})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning ls-remote output: %w", err)
	}
	return refs, nil
}
