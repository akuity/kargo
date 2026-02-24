package git

import (
	"os/exec"
	"strings"
	"testing"
)

// this is a bit of a hack to get the default branch name of the repository,
// which is necessary because it can differ based on git version and user configuration
// and addresses differences between running tests in this package locally vs in CI.
// Git 2.28+ allows configuring the default branch name via the init.defaultBranch config,
// and gitkit doesn't appear to provide a way to specify the initial branch name when initializing a repository,
// so we have to query git directly here.
func defaultInitBranch(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "config", "init.defaultBranch")
	cmd.Dir = t.TempDir()
	out, err := cmd.Output()
	if err != nil {
		// if the command fails, it's likely because the git version is older than 2.28 and
		// doesn't support init.defaultBranch, in which case we can assume the default branch name is "master".
		return "master"
	}
	return strings.TrimSpace(string(out))
}
