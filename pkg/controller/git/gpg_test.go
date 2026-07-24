package git

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
)

func Test_workTree_GetCommitSignatureInfo(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
	)
	defer testServer.Close()

	t.Run("unsigned commit is not trusted", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(
			fmt.Sprintf("%s/test.txt", repo.Dir()),
			[]byte("hello"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "unsigned commit", nil)
		require.NoError(t, err)

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		info, err := repo.GetCommitSignatureInfo(t.Context(), commitID)
		require.NoError(t, err)
		require.False(t, info.Trusted)
		require.Empty(t, info.SignerName)
		require.Empty(t, info.SignerEmail)
	})

	t.Run("commit signed by trusted key", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/test.txt", repo.Dir()),
			[]byte("signed"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "trusted commit", nil)
		require.NoError(t, err)

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		info, err := repo.GetCommitSignatureInfo(t.Context(), commitID)
		require.NoError(t, err)
		require.True(t, info.Trusted)
		// Note: the fake GPG script used by enableFakeCommitSigning does
		// not produce a real %GS signer identity, so we don't assert on
		// SignerName/SignerEmail here. parseSignerIdentity is tested
		// separately.
	})

	t.Run("commit signed by untrusted key is not trusted", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, false)

		err = os.WriteFile(
			fmt.Sprintf("%s/test.txt", repo.Dir()),
			[]byte("untrusted"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "untrusted commit", nil)
		require.NoError(t, err)

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		info, err := repo.GetCommitSignatureInfo(t.Context(), commitID)
		require.NoError(t, err)
		require.False(t, info.Trusted)
	})
}

func Test_workTree_verifyCommitSignature(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t, initialMainCommit,
	)
	defer testServer.Close()

	t.Run("unsigned commit", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		status, err := wt.verifyCommitSignature(t.Context(), commitID)
		require.NoError(t, err)
		require.Equal(t, signatureUnsigned, status)
	})

	t.Run("trusted signature", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/signed.txt", repo.Dir()),
			[]byte("signed"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "signed commit", nil)
		require.NoError(t, err)

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		status, err := wt.verifyCommitSignature(t.Context(), commitID)
		require.NoError(t, err)
		require.Equal(t, signatureTrusted, status)
	})

	t.Run("untrusted signature", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, false)

		err = os.WriteFile(
			fmt.Sprintf("%s/signed.txt", repo.Dir()),
			[]byte("signed"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "signed commit", nil)
		require.NoError(t, err)

		commitID, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		status, err := wt.verifyCommitSignature(t.Context(), commitID)
		require.NoError(t, err)
		require.Equal(t, signatureUntrusted, status)
	})
}

func Test_workTree_isSigningConfigured(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t, initialMainCommit,
	)
	defer testServer.Close()

	t.Run("not configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		configured, err := internalWorkTree(t, repo).isSigningConfigured(t.Context())
		require.NoError(t, err)
		require.False(t, configured)
	})

	t.Run("configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		_, err = libExec.Exec(wt.buildGitCommand(
			t.Context(), "config", "--global", "commit.gpgSign", "true",
		))
		require.NoError(t, err)

		configured, err := wt.isSigningConfigured(t.Context())
		require.NoError(t, err)
		require.True(t, configured)
	})
}

func Test_parseSignerIdentity(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedName  string
		expectedEmail string
	}{
		{
			name:          "standard format",
			input:         "Alice Smith <alice@example.com>",
			expectedName:  "Alice Smith",
			expectedEmail: "alice@example.com",
		},
		{
			name:          "no angle brackets",
			input:         "Alice Smith",
			expectedName:  "Alice Smith",
			expectedEmail: "",
		},
		{
			name:          "empty string",
			input:         "",
			expectedName:  "",
			expectedEmail: "",
		},
		{
			name:          "email only",
			input:         "<alice@example.com>",
			expectedName:  "",
			expectedEmail: "alice@example.com",
		},
		{
			name:          "name with angle bracket",
			input:         "Alice <Bot> Smith <alice@example.com>",
			expectedName:  "Alice <Bot> Smith",
			expectedEmail: "alice@example.com",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			name, email := parseSignerIdentity(testCase.input)
			require.Equal(t, testCase.expectedName, name)
			require.Equal(t, testCase.expectedEmail, email)
		})
	}
}
