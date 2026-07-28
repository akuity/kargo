package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LsRemote(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
		func(t *testing.T, rep WorkTree) {
			// Create and push an annotated tag so we can verify that its peeled
			// "^{}" entry is dropped and only the tag object is reported.
			require.NoError(t, rep.CreateTag("v1.0.0", "release v1.0.0", nil))
			require.NoError(t, rep.Push(&PushOptions{Tag: "v1.0.0"}))
		},
	)
	defer testServer.Close()

	clientOpts := &ClientOptions{Credentials: &testRepoCreds}

	// Note: HEAD resolution is exercised by the newestFromBranchSelector unit
	// tests rather than here. Whether a server advertises a HEAD symref depends
	// on how its bare repository was initialized (the test harness leaves it
	// pointing at a possibly-absent default branch), so it is not a stable
	// signal in this environment. Our handling of whatever ls-remote returns is
	// what these tests cover.

	t.Run("branch ref is listed", func(t *testing.T) {
		refs, err := LsRemote(testRepoURL, clientOpts, "refs/heads/main")
		require.NoError(t, err)
		require.Len(t, refs, 1)
		require.Equal(t, "refs/heads/main", refs[0].Name)
	})

	t.Run("annotated tag collapses to a single entry", func(t *testing.T) {
		refs, err := LsRemote(testRepoURL, clientOpts, "refs/tags/*")
		require.NoError(t, err)
		// Despite the annotated tag's peeled entry on the wire, exactly one ref
		// is reported for the tag.
		require.Len(t, refs, 1)
		require.Equal(t, "refs/tags/v1.0.0", refs[0].Name)
	})

	t.Run("absent ref yields no entries", func(t *testing.T) {
		refs, err := LsRemote(testRepoURL, clientOpts, "refs/heads/nonexistent")
		require.NoError(t, err)
		require.Empty(t, refs)
	})
}

func Test_parseLsRemoteOutput(t *testing.T) {
	testCases := []struct {
		name       string
		output     string
		assertions func(*testing.T, []RemoteRef, error)
	}{
		{
			name:   "empty output",
			output: "",
			assertions: func(t *testing.T, refs []RemoteRef, err error) {
				require.NoError(t, err)
				require.Empty(t, refs)
			},
		},
		{
			name: "branches and tags",
			output: "aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78\trefs/heads/main\n" +
				"4f7c1b2a9e3d8a6b5c2f1e0d9b8a7c6e5d4f3a2b\trefs/tags/v1.2.3\n",
			assertions: func(t *testing.T, refs []RemoteRef, err error) {
				require.NoError(t, err)
				require.Equal(t, []RemoteRef{
					{
						Name: "refs/heads/main",
						ID:   "aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78",
					},
					{
						Name: "refs/tags/v1.2.3",
						ID:   "4f7c1b2a9e3d8a6b5c2f1e0d9b8a7c6e5d4f3a2b",
					},
				}, refs)
			},
		},
		{
			name: "peeled tag entry is dropped",
			output: "1111111111111111111111111111111111111111\trefs/tags/v1.0.0\n" +
				"2222222222222222222222222222222222222222\trefs/tags/v1.0.0^{}\n",
			assertions: func(t *testing.T, refs []RemoteRef, err error) {
				require.NoError(t, err)
				// The peeled "^{}" line is dropped, leaving only the tag object
				// entry. Annotated tags thus collapse to a single ref.
				require.Equal(t, []RemoteRef{{
					Name: "refs/tags/v1.0.0",
					ID:   "1111111111111111111111111111111111111111",
				}}, refs)
			},
		},
		{
			name: "HEAD entry is retained",
			output: "aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78\tHEAD\n" +
				"aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78\trefs/heads/develop\n",
			assertions: func(t *testing.T, refs []RemoteRef, err error) {
				require.NoError(t, err)
				require.Equal(t, []RemoteRef{
					{
						Name: "HEAD",
						ID:   "aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78",
					},
					{
						Name: "refs/heads/develop",
						ID:   "aa12bb34cc56dd78ee90ff12aabb34ccdd56ee78",
					},
				}, refs)
			},
		},
		{
			name:   "malformed line without tab is skipped",
			output: "not-a-valid-line\n",
			assertions: func(t *testing.T, refs []RemoteRef, err error) {
				require.NoError(t, err)
				require.Empty(t, refs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			refs, err := parseLsRemoteOutput([]byte(testCase.output))
			testCase.assertions(t, refs, err)
		})
	}
}
