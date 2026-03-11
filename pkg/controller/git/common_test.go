package git

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

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
