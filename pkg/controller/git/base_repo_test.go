package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupUser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// createRepoDir controls whether b.dir exists before calling
		// setupAuthor. When false, simulates a clone-in-progress where
		// the repo directory hasn't been created yet.
		createRepoDir bool
		author        *User
		assert        func(*testing.T, string, string, error)
	}{
		{
			name:          "nil author uses defaults",
			createRepoDir: true,
			author:        nil,
			assert: func(t *testing.T, homeDir, _ string, err error) {
				require.NoError(t, err)
				assertGitConfig(t, homeDir, "user.name", defaultUsername)
				assertGitConfig(t, homeDir, "user.email", defaultEmail)
			},
		},
		{
			name:          "custom name and email",
			createRepoDir: true,
			author: &User{
				Name:  "Test User",
				Email: "test@example.com",
			},
			assert: func(t *testing.T, homeDir, _ string, err error) {
				require.NoError(t, err)
				assertGitConfig(t, homeDir, "user.name", "Test User")
				assertGitConfig(t, homeDir, "user.email", "test@example.com")
			},
		},
		{
			// Regression: setupAuthor must use homeDir (not b.homeDir) so
			// that Commit() can set up per-commit author config in a
			// temporary home directory.
			name:          "config written to homeDir not repo homeDir",
			createRepoDir: true,
			author: &User{
				Name:  "Per-Commit Author",
				Email: "per-commit@example.com",
			},
			assert: func(t *testing.T, homeDir, repoHomeDir string, err error) {
				require.NoError(t, err)
				assertGitConfig(t, homeDir, "user.name", "Per-Commit Author")
				assertNoGitConfig(t, repoHomeDir)
			},
		},
		{
			// During clone, b.dir doesn't exist yet. cmd.Dir is overridden
			// to homeDir so the command can still execute.
			name:          "succeeds when repo dir does not exist",
			createRepoDir: false,
			author:        nil,
			assert: func(t *testing.T, homeDir, _ string, err error) {
				require.NoError(t, err)
				assertGitConfig(t, homeDir, "user.name", defaultUsername)
			},
		},
		{
			name:          "with signing key path",
			createRepoDir: true,
			author: &User{
				Name:           "Test User",
				Email:          "test@example.com",
				SigningKeyPath: "/nonexistent/key.asc",
			},
			assert: func(t *testing.T, homeDir, _ string, err error) {
				// git config succeeds but gpg --import fails because the
				// key file doesn't exist. This exercises the setCmdHome
				// calls in the signing path.
				require.ErrorContains(t, err, "error importing gpg key")
				assertGitConfig(t, homeDir, "commit.gpgsign", "true")
			},
		},
		{
			name:          "with inline signing key",
			createRepoDir: true,
			author: &User{
				Name:       "Test User",
				Email:      "test@example.com",
				SigningKey: "not-a-real-key",
			},
			assert: func(t *testing.T, homeDir, _ string, err error) {
				// The key file is written and git config succeeds but gpg
				// --import fails because the key content is invalid.
				require.ErrorContains(t, err, "error importing gpg key")
				assertGitConfig(t, homeDir, "commit.gpgsign", "true")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			homeDir := t.TempDir()
			repoHomeDir := t.TempDir()

			repoDir := filepath.Join(t.TempDir(), "repo")
			if tc.createRepoDir {
				require.NoError(t, os.MkdirAll(repoDir, 0o700))
			}

			b := &baseRepo{
				dir:     repoDir,
				homeDir: repoHomeDir,
			}

			err := b.setupUser(homeDir, tc.author)
			tc.assert(t, homeDir, repoHomeDir, err)
		})
	}
}

// assertGitConfig verifies that a git config key has the expected value in
// the .gitconfig within the given home directory.
func assertGitConfig(t *testing.T, homeDir, key, expected string) {
	t.Helper()
	configPath := filepath.Join(homeDir, ".gitconfig")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "expected .gitconfig in %s", homeDir)
	require.Contains(t,
		string(content), expected,
		"expected %s = %s in %s", key, expected, configPath,
	)
}

// assertNoGitConfig verifies that no .gitconfig exists in the given directory.
func assertNoGitConfig(t *testing.T, homeDir string) {
	t.Helper()
	configPath := filepath.Join(homeDir, ".gitconfig")
	_, err := os.Stat(configPath)
	require.True(t,
		os.IsNotExist(err),
		"expected no .gitconfig in %s but found one", homeDir,
	)
}
