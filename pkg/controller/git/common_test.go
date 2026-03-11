package git

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/types"
)

// setupRemoteRepo creates a gitkit-backed remote repository accessible via an
// http server, returning the server, repo URL, and credentials, if applicable.
// The caller must close the server when done with it.
func setupRemoteRepo(
	t *testing.T,
	initFns ...func(t *testing.T, repo WorkTree),
) (*httptest.Server, string, RepoCredentials) {
	t.Helper()

	creds := RepoCredentials{
		Username: "test-user",
		Password: "test-pass",
	}

	var useAuth bool
	if s := os.Getenv("TEST_GIT_CLIENT_WITH_AUTH"); s != "" {
		useAuth = types.MustParseBool(s)
	}
	service := gitkit.New(gitkit.Config{
		Dir:        t.TempDir(),
		AutoCreate: true,
		Auth:       useAuth,
	})
	require.NoError(t, service.Setup())
	service.AuthFunc = func(
		c gitkit.Credential, _ *gitkit.Request,
	) (bool, error) {
		return c.Username == creds.Username &&
			c.Password == creds.Password, nil
	}
	server := httptest.NewServer(service)
	// server must be closed by the caller

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &creds}, nil)
	require.NoError(t, err)
	require.NotNil(t, repo)
	defer repo.Close()

	for _, initFn := range initFns {
		initFn(t, repo)
	}

	return server, testRepoURL, creds
}

func initialMainCommit(t *testing.T, rep WorkTree) {
	err := os.WriteFile(
		fmt.Sprintf("%s/%s", rep.Dir(), "test.txt"),
		[]byte("foo"),
		0600,
	)
	require.NoError(t, err)
	err = rep.AddAllAndCommit(fmt.Sprintf("initial commit %s", uuid.NewString()), nil)
	require.NoError(t, err)
	err = rep.Push(nil)
	require.NoError(t, err)
}

func commitAhead(t *testing.T, rep WorkTree) {
	err := os.WriteFile(
		fmt.Sprintf("%s/remote.txt", rep.Dir()),
		[]byte("from-remote"),
		0o600,
	)
	require.NoError(t, err)
	err = rep.AddAllAndCommit("remote commit", nil)
	require.NoError(t, err)
	err = rep.Push(&PushOptions{TargetBranch: "ahead"})
	require.NoError(t, err)
}

// internalWorkTree extracts the unexported *workTree from a Repo so tests
// can call unexported methods.
func internalWorkTree(t *testing.T, r Repo) *workTree {
	t.Helper()
	rr, ok := r.(*repo)
	require.True(t, ok, "expected *repo, got %T", r)
	return rr.workTree
}

// enableFakeCommitSigning installs a fake GPG program into the work tree's home
// directory and configures git to use it. The script handles both signing
// and verification. When trusted is true, verification reports
// TRUST_ULTIMATE (git %G? = G). When false, verification reports
// TRUST_UNDEFINED (git %G? = U).
func enableFakeCommitSigning(t *testing.T, wt *workTree, trusted bool) {
	t.Helper()
	trustLine := "echo '[GNUPG:] TRUST_UNDEFINED'"
	if trusted {
		trustLine = "echo '[GNUPG:] TRUST_ULTIMATE'"
	}
	// The script inspects its arguments to distinguish signing from
	// verification. Git passes --verify when checking a signature and
	// -bsau when creating one. The signing output must be a valid PGP
	// armor block; git validates the format before calling GPG to verify.
	script := fmt.Sprintf(
		"#!/bin/sh\n"+
			"verify=0\n"+
			"for arg in \"$@\"; do\n"+
			"  case \"$arg\" in --verify) verify=1 ;; esac\n"+
			"done\n"+
			"if [ \"$verify\" = 1 ]; then\n"+
			"  echo '[GNUPG:] GOODSIG ABCDEF1234567890 Test User'\n"+
			"  %s\n"+
			"else\n"+
			"  cat >/dev/null\n"+
			"  echo '' >&2\n"+
			"  echo '[GNUPG:] SIG_CREATED D 1 2 00 0 ABCDEF' >&2\n"+
			"  echo '-----BEGIN PGP SIGNATURE-----'\n"+
			"  echo ''\n"+
			"  echo 'iQEzBAABCAAdFiEEfake'\n"+
			"  echo '=fake'\n"+
			"  echo '-----END PGP SIGNATURE-----'\n"+
			"fi\n",
		trustLine,
	)
	fakeGPG := filepath.Join(wt.HomeDir(), "fake-gpg")
	err := os.WriteFile(fakeGPG, []byte(script), 0o755)
	require.NoError(t, err)
	_, err = libExec.Exec(wt.buildGitCommand(
		"config", "--global", "gpg.program", fakeGPG,
	))
	require.NoError(t, err)
	_, err = libExec.Exec(wt.buildGitCommand(
		"config", "--global", "commit.gpgSign", "true",
	))
	require.NoError(t, err)
}

// disableFakeCommitSigning turns off commit.gpgSign in the work tree's global
// git config.
func disableFakeCommitSigning(t *testing.T, wt *workTree) {
	t.Helper()
	_, err := libExec.Exec(wt.buildGitCommand(
		"config", "--unset", "--global", "commit.gpgSign",
	))
	require.NoError(t, err)
	_, err = libExec.Exec(wt.buildGitCommand(
		"config", "--global", "commit.gpgSign", "false",
	))
	require.NoError(t, err)
}
