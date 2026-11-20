package git

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
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

			err := b.setupUser(t.Context(), homeDir, tc.author)
			tc.assert(t, homeDir, repoHomeDir, err)
		})
	}
}

func TestSetupDirs(t *testing.T) {
	t.Parallel()

	baseDir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	b := &baseRepo{}
	require.NoError(t, b.setupDirs(t.Context(), baseDir))
	require.Equal(t, baseDir, filepath.Dir(b.homeDir))
	require.Equal(t, filepath.Join(b.homeDir, "repo"), b.dir)
	assertGitConfig(t, b.homeDir, "init.defaultBranch", "main")
	assertGitConfig(t, b.homeDir, "maintenance.auto", "false")
}

func TestDisableAutoMaintenance(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	b := &baseRepo{homeDir: t.TempDir()}
	require.NoError(t, b.disableAutoMaintenance(t.Context(), homeDir))
	assertGitConfig(t, homeDir, "maintenance.auto", "false")
	assertNoGitConfig(t, b.homeDir)
}

func TestValidateRepoURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		repoURL string
		assert  func(*testing.T, error)
	}{
		{
			name:    "unparseable URL",
			repoURL: "https://user:secret@bad host/example/repo.git",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "only HTTP(S) URLs are supported")
				require.NotContains(t, err.Error(), "secret")
			},
		},
		{
			name:    "SSH URL",
			repoURL: "ssh://git:secret@github.com/example/repo.git",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "only HTTP(S) URLs are supported")
				require.ErrorContains(t, err, "ssh://git:xxxxx@github.com")
				require.NotContains(t, err.Error(), "secret")
			},
		},
		{
			name:    "SCP-style URL",
			repoURL: "git@github.com:example/repo.git",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "only HTTP(S) URLs are supported")
			},
		},
		{
			name:    "local path",
			repoURL: "/tmp/repo",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "only HTTP(S) URLs are supported")
			},
		},
		{
			name:    "HTTP URL",
			repoURL: "http://localhost:8080/example/repo.git",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "HTTPS URL",
			repoURL: "HTTPS://github.com/example/repo.git",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, validateRepoURL(testCase.repoURL))
		})
	}
}

func TestBuildGitCommandStallDetection(t *testing.T) {
	t.Parallel()

	b := &baseRepo{
		dir:     t.TempDir(),
		homeDir: t.TempDir(),
	}

	cmd := b.buildGitCommand(t.Context(), "status")
	require.Contains(t, cmd.Env, "GIT_HTTP_LOW_SPEED_LIMIT="+gitHTTPLowSpeedLimit)
	require.Contains(t, cmd.Env, "GIT_HTTP_LOW_SPEED_TIME="+gitHTTPLowSpeedTime)
}

func TestGitAbortsStalledTransfer(t *testing.T) {
	t.Parallel()

	// A server that accepts git's initial request (GET .../info/refs) and then
	// never sends a byte -- the same shape as a connection black-holed at the
	// network layer.
	blocked := make(chan struct{})
	testServer := httptest.NewServer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			<-blocked
		}),
	)
	t.Cleanup(func() {
		close(blocked)
		testServer.Close()
	})

	b := &baseRepo{
		dir:     t.TempDir(),
		homeDir: t.TempDir(),
	}

	cmd := b.buildGitCommand(
		t.Context(),
		"clone",
		testServer.URL+"/repo.git",
		filepath.Join(t.TempDir(), "repo"),
	)
	// Shrink the stall window so the test proves the mechanism -- git aborting
	// a transfer that is moving no data -- without waiting out the full
	// production window.
	for i, v := range cmd.Env {
		if strings.HasPrefix(v, "GIT_HTTP_LOW_SPEED_TIME=") {
			cmd.Env[i] = "GIT_HTTP_LOW_SPEED_TIME=2"
		}
	}

	start := time.Now()
	_, err := libExec.Exec(cmd)
	elapsed := time.Since(start)

	// Without stall detection, the clone would block on the silent server
	// indefinitely. With it, git aborts once throughput has stayed below the
	// floor for the configured window.
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "too slow")
	require.Less(t, elapsed, 30*time.Second)
}

func TestBuildCommandCancellation(t *testing.T) {
	t.Parallel()

	b := &baseRepo{
		dir:     t.TempDir(),
		homeDir: t.TempDir(),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	cmd := b.buildCommand(ctx, "sleep", "30")
	start := time.Now()
	_, err := libExec.Exec(cmd)
	elapsed := time.Since(start)

	require.Error(t, err)
	// The command must be terminated promptly once the context expires rather
	// than running to completion. The bound is generous to avoid flakes on
	// slow machines, but far below the command's 30-second natural duration.
	require.Less(t, elapsed, 10*time.Second)
}

func TestSetupAuth(t *testing.T) {
	const (
		testURLWithCreds    = "https://bob:secret@example.com/repo.git"
		testURLWithoutCreds = "https://example.com/repo.git"
	)
	testCreds := &RepoCredentials{
		Username: "user",
		Password: "pass",
	}
	testCases := []struct {
		name   string
		url    string
		creds  *RepoCredentials
		assert func(*testing.T, string, error)
	}{
		{
			name: "no credentials",
			url:  testURLWithCreds,
			assert: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				require.Equal(t, testURLWithCreds, url)
			},
		},
		{
			name:  "no password",
			url:   testURLWithCreds,
			creds: &RepoCredentials{Username: "user"},
			assert: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				require.Equal(t, testURLWithCreds, url)
			},
		},
		{
			name:  "no credentials in URL",
			url:   testURLWithoutCreds,
			creds: testCreds,
			assert: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				require.Equal(t, testURLWithoutCreds, url)
			},
		},
		{
			name:  "username in URL",
			url:   "https://bob@example.com/repo.git",
			creds: testCreds,
			assert: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				require.Equal(t, testURLWithoutCreds, url)
			},
		},
		{
			name:  "username and password in URL",
			url:   testURLWithCreds,
			creds: testCreds,
			assert: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				require.Equal(t, testURLWithoutCreds, url)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			b := &baseRepo{
				url:   testCase.url,
				creds: testCase.creds,
			}
			err := b.setupAuth()
			testCase.assert(t, b.url, err)
		})
	}
}

func TestGitHTTPAuthNeverSendsEmptyPassword(t *testing.T) {
	t.Parallel()

	const (
		testUsername = "test-user"
		testPassword = "test-pass"
	)

	// Build the real GIT_ASKPASS program rather than a stand-in for it.
	helperPath := filepath.Join(t.TempDir(), "credential-helper")
	out, err := exec.Command(
		"go", "build",
		"-o", helperPath,
		"github.com/akuity/kargo/cmd/credential-helper",
	).CombinedOutput()
	require.NoError(t, err, string(out))

	// A server that records the credentials on each request ("" for none) and
	// demands Basic auth until it gets the right ones.
	var (
		mu       sync.Mutex
		received []string
	)
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var creds string
			if username, password, ok := r.BasicAuth(); ok {
				creds = username + ":" + password
			}
			mu.Lock()
			received = append(received, creds)
			mu.Unlock()
			if creds != testUsername+":"+testPassword {
				w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}),
	)
	t.Cleanup(testServer.Close)

	b := &baseRepo{
		url: testServer.URL + "/repo.git",
		creds: &RepoCredentials{
			Username: testUsername,
			Password: testPassword,
		},
		dir:     t.TempDir(),
		homeDir: t.TempDir(),
	}
	require.NoError(t, b.setupAuth())

	cmd := b.buildGitCommand(t.Context(), "ls-remote", b.url)
	cmd.Env = append(
		cmd.Env,
		// When a key repeats in Env, the last value wins
		"GIT_ASKPASS="+helperPath,
		// Keep a credential helper in the system git config from answering
		"GIT_CONFIG_NOSYSTEM=1",
	)
	// The server answers the authenticated request with a 404, so git fails.
	// Only the requests it made matter.
	_, _ = libExec.Exec(cmd)

	mu.Lock()
	defer mu.Unlock()
	require.NotContains(t, received, testUsername+":")
	require.NotEmpty(t, received)
	require.Equal(t, testUsername+":"+testPassword, received[len(received)-1])
}

// assertGitConfig verifies that a git config key has the expected value in
// the .gitconfig within the given home directory.
func assertGitConfig(t *testing.T, homeDir, key, expected string) {
	t.Helper()
	configPath := filepath.Join(homeDir, ".gitconfig")
	out, err := libExec.Exec(
		exec.Command("git", "config", "--file", configPath, "--get", key),
	)
	require.NoError(t, err, "expected %s in %s", key, configPath)
	require.Equal(t, expected, strings.TrimSpace(string(out)), key)
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
