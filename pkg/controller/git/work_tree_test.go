package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testingPkg "github.com/akuity/kargo/api/testing"
	libExec "github.com/akuity/kargo/pkg/exec"
)

func TestNonFastForwardRegex(t *testing.T) {
	testCases := map[string]bool{
		// source: https://regex101.com/r/aNYjHP/1
		" ! [rejected]        krancour/foo -> krancour/foo (non-fast-forward)": true,
		" ! [rejected]        main -> main (fetch first)":                      true,
		" ! [remote rejected] HEAD -> experiment (cannot lock ref 'refs/heads/experiment': is at " +
			"7dc98ee9c0b75be429e300bb59b3cf6d091ca9ed but expected 1bdf96c8c868981a0e24c43c98aef09a8970a1b8)": true,
		" ! [rejected]        HEAD -> experiment (fetch first)":            true,
		" ! [remote rejected] HEAD -> main (incorrect old value provided)": true,
	}

	testingPkg.ValidateRegularExpression(t, nonFastForwardRegex, testCases)
}

func TestWorkTree(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t, initialMainCommit)
	defer testServer.Close()

	rep, err := CloneBare(
		t.Context(),
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(t.Context())

	workingTreePath := filepath.Join(rep.HomeDir(), "working-tree")
	workTree, err := rep.AddWorkTree(
		t.Context(),
		workingTreePath,
		&AddWorkTreeOptions{Ref: "main"},
	)
	require.NoError(t, err)
	defer workTree.Close(t.Context())

	t.Run("can load an existing working tree", func(t *testing.T) {
		existingWorkTree, err := LoadWorkTree(
			t.Context(),
			workTree.Dir(),
			&LoadWorkTreeOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, workTree, existingWorkTree)
	})

	t.Run("can close working tree", func(t *testing.T) {
		require.NoError(t, workTree.Close(t.Context()))
		_, err := os.Stat(workTree.Dir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

}

func Test_workTree_Pull(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t, initialMainCommit, commitAhead,
	)
	defer testServer.Close()

	t.Run("force pull resets to remote", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Make a local commit that diverges from "ahead".
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local divergent commit", nil)
		require.NoError(t, err)

		localCommit, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)

		// Force pull from "ahead" should reset to the remote state.
		err = repo.Pull(t.Context(), &PullOptions{Branch: "ahead", Force: true})
		require.NoError(t, err)

		newCommit, err := repo.LastCommitID(t.Context())
		require.NoError(t, err)
		require.NotEqual(t, localCommit, newCommit)

		// The remote-only file should be present.
		_, err = os.Stat(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
		)
		require.NoError(t, err)

		// The local-only file should be gone (hard reset).
		_, err = os.Stat(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
		)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("non-force pull merges remote", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Make a local commit on a different file to avoid conflicts.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local non-conflicting commit", nil)
		require.NoError(t, err)

		// Non-force pull from "ahead" should merge.
		err = repo.Pull(t.Context(), &PullOptions{Branch: "ahead"})
		require.NoError(t, err)

		// Both files should be present after merge.
		_, err = os.Stat(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
		)
		require.NoError(t, err)
		_, err = os.Stat(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
		)
		require.NoError(t, err)

		// Should have a merge commit.
		msg, err := repo.CommitMessage(t.Context(), "HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})

	t.Run("nil opts defaults to current branch", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Pull with nil opts should not error (fetches current branch).
		err = repo.Pull(t.Context(), nil)
		require.NoError(t, err)
	})
}

func Test_workTree_Fetch(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initMultipleBranches,
	)
	t.Cleanup(testServer.Close)

	testCases := []struct {
		name   string
		opts   *FetchOptions
		assert func(*testing.T, *repo, error)
	}{
		{
			name: "fetched branch can be checked out by name",
			opts: &FetchOptions{Branch: "a"},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				require.NoError(t, r.Checkout(t.Context(), "a"))
				requireCurrentBranch(t, r, "a")
			},
		},
		{
			name: "fetching a branch again does not track it twice",
			opts: &FetchOptions{Branch: "a"},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				require.NoError(t, r.Fetch(t.Context(), &FetchOptions{Branch: "a"}))
				refspecs, err := r.fetchRefspecs(t.Context())
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{
						"+refs/heads/main:refs/remotes/origin/main",
						"+refs/heads/a:refs/remotes/origin/a",
					},
					refspecs,
				)
			},
		},
		{
			name: "all fetched branches can be checked out by name",
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				require.NoError(t, r.Checkout(t.Context(), "b"))
				requireCurrentBranch(t, r, "b")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// A single-branch clone, whose configured fetch refspecs cover only
			// the default branch
			rep, err := Clone(
				t.Context(),
				testRepoURL,
				&ClientOptions{Credentials: &testRepoCreds},
				&CloneOptions{Refs: []string{headRef}},
			)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, rep.Close(context.Background())) })
			r, ok := rep.(*repo)
			require.True(t, ok)
			testCase.assert(t, r, r.Fetch(t.Context(), testCase.opts))
		})
	}
}

func Test_isTracked(t *testing.T) {
	testCases := []struct {
		name     string
		refspecs []string
		branch   string
		tracked  bool
	}{
		{
			name:   "no refspecs",
			branch: "main",
		},
		{
			name:     "refspec for this branch",
			refspecs: []string{"+refs/heads/main:refs/remotes/origin/main"},
			branch:   "main",
			tracked:  true,
		},
		{
			name:     "refspec for this branch without +",
			refspecs: []string{"refs/heads/main:refs/remotes/origin/main"},
			branch:   "main",
			tracked:  true,
		},
		{
			name:     "refspec for all branches",
			refspecs: []string{"+refs/heads/*:refs/remotes/origin/*"},
			branch:   "main",
			tracked:  true,
		},
		{
			name:     "refspec for another branch",
			refspecs: []string{"+refs/heads/main:refs/remotes/origin/main"},
			branch:   "a",
		},
		{
			name:     "all branches requested but only one tracked",
			refspecs: []string{"+refs/heads/main:refs/remotes/origin/main"},
			branch:   "*",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.tracked,
				isTracked(testCase.refspecs, testCase.branch),
			)
		})
	}
}

func Test_parseTagMetadataLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    TagMetadata
		wantErr bool
	}{
		{
			name:    "incorrect number of fields",
			line:    "tag3|*|commitid3|*|subject3|*|author3",
			wantErr: true,
		},
		{
			name: "lightweight tag",
			line: "tag1|*|commitid1|*|subject1|*|author1|*|committer1|*|2024-01-01 12:00:00 -0500",
			want: TagMetadata{
				Tag:         "tag1",
				CommitID:    "commitid1",
				Subject:     "subject1",
				Author:      "author1",
				Committer:   "committer1",
				CreatorDate: mustParseTime("2024-01-01 12:00:00 -0500"),
			},
		},
		{
			name: "annotated tag with extra |*| in annotation",
			line: "tag2|*|commitid2|*|subject2|*|author2|*|committer2|*|" +
				"2024-01-01 12:00:00 -0500|*|tagger2|*|annotation with |*| inside",
			want: TagMetadata{
				Tag:         "tag2",
				CommitID:    "commitid2",
				Subject:     "subject2",
				Author:      "author2",
				Committer:   "committer2",
				CreatorDate: mustParseTime("2024-01-01 12:00:00 -0500"),
				Tagger:      "tagger2",
				Annotation:  "annotation with |*| inside",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTagMetadataLine([]byte(tt.line))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05 -0700", s)
	return t
}

func TestListCommits(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		t.Context(),
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(t.Context())

	wt := internalWorkTree(t, rep)

	// Create initial commit on main
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/file.txt", rep.Dir()),
			[]byte("initial"),
			0o600,
		),
	)
	require.NoError(
		t,
		rep.AddAllAndCommit(t.Context(), "main: initial commit", nil),
	)

	// Create a feature branch from the initial commit
	require.NoError(t, rep.CreateChildBranch(t.Context(), "feature"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/feature.txt", rep.Dir()),
			[]byte("feature work 1"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit(t.Context(), "feature: work 1", nil))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/feature.txt", rep.Dir()),
			[]byte("feature work 2"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit(t.Context(), "feature: work 2", nil))

	// Back to main, add another commit
	require.NoError(t, rep.Checkout(t.Context(), "main"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/file.txt", rep.Dir()),
			[]byte("second"),
			0o600,
		),
	)
	require.NoError(
		t,
		rep.AddAllAndCommit(t.Context(), "main: second commit", nil),
	)

	// Merge the feature branch into main
	_, err = libExec.Exec(wt.buildGitCommand(
		t.Context(), "merge", "feature", "--no-ff", "-m", "main: merge feature",
	))
	require.NoError(t, err)

	// ListCommits should only return first-parent commits (main line),
	// not the individual feature branch commits.
	commits, err := rep.ListCommits(t.Context(), nil)
	require.NoError(t, err)

	subjects := make([]string, len(commits))
	for i, c := range commits {
		subjects[i] = c.Subject
	}
	require.Equal(
		t,
		[]string{
			"main: merge feature",
			"main: second commit",
			"main: initial commit",
		},
		subjects,
	)
}

func TestGetDiffPathsForMergeCommit(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		t.Context(),
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(t.Context())

	wt := internalWorkTree(t, rep)

	// Create initial commit on main with base files
	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/foo", rep.Dir()), 0o755))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file1.txt", rep.Dir()),
			[]byte("base"),
			0o600,
		),
	)
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file2.txt", rep.Dir()),
			[]byte("base"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit(t.Context(), "initial commit", nil))

	// Create branch-a and modify file1
	require.NoError(t, rep.CreateChildBranch(t.Context(), "branch-a"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file1.txt", rep.Dir()),
			[]byte("changed by branch-a"),
			0o600,
		),
	)
	require.NoError(
		t,
		rep.AddAllAndCommit(t.Context(), "branch-a: modify file1", nil),
	)

	// Back to main, create branch-b, modify file2
	require.NoError(t, rep.Checkout(t.Context(), "main"))
	require.NoError(t, rep.CreateChildBranch(t.Context(), "branch-b"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file2.txt", rep.Dir()),
			[]byte("changed by branch-b"),
			0o600,
		),
	)
	require.NoError(
		t,
		rep.AddAllAndCommit(t.Context(), "branch-b: modify file2", nil),
	)

	// Merge branch-b into main
	require.NoError(t, rep.Checkout(t.Context(), "main"))
	_, err = libExec.Exec(wt.buildGitCommand(
		t.Context(), "merge", "branch-b", "--no-ff", "-m", "merge branch-b",
	))
	require.NoError(t, err)

	// Merge branch-a into main
	_, err = libExec.Exec(wt.buildGitCommand(
		t.Context(), "merge", "branch-a", "--no-ff", "-m", "merge branch-a",
	))
	require.NoError(t, err)

	mergeCommitID, err := rep.LastCommitID(t.Context())
	require.NoError(t, err)

	// GetDiffPathsForCommitID on the merge commit should return only
	// the file introduced by that merge (file1, from branch-a), not
	// file2 which was already on main via the earlier merge of branch-b.
	paths, err := rep.GetDiffPathsForCommitID(t.Context(), mergeCommitID)
	require.NoError(t, err)
	require.Equal(t, []string{"foo/file1.txt"}, paths)
}

func TestGetDiffPathsForCommitWithSpecialCharacters(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		t.Context(),
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer func() {
		require.NoError(t, rep.Close(t.Context()))
	}()

	wt := internalWorkTree(t, rep)
	_, err = libExec.Exec(wt.buildGitCommand(
		t.Context(), "config", "core.quotePath", "true",
	))
	require.NoError(t, err)

	wantPaths := []string{
		"apps/café/values.yaml",
		`apps/with space/a"b.yaml`,
		"apps/plain/values.yaml",
	}
	for _, path := range wantPaths {
		fullPath := filepath.Join(rep.Dir(), path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte("value: test"), 0o600))
	}
	require.NoError(t, rep.AddAllAndCommit(t.Context(), "add special paths", nil))

	commitID, err := rep.LastCommitID(t.Context())
	require.NoError(t, err)

	paths, err := rep.GetDiffPathsForCommitID(t.Context(), commitID)
	require.NoError(t, err)
	require.ElementsMatch(t, wantPaths, paths)
}

func TestGetDiffPathsForMovedFile(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		t.Context(),
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(t.Context())

	wt := internalWorkTree(t, rep)

	// Create initial commit with a file in app/
	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/app", rep.Dir()), 0o755))
	require.NoError(t, os.WriteFile(
		fmt.Sprintf("%s/app/pod.yaml", rep.Dir()),
		[]byte("kind: Pod"),
		0o600,
	))
	require.NoError(t, rep.AddAllAndCommit(t.Context(), "initial commit", nil))

	// Move the file to a different directory
	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/different-app", rep.Dir()), 0o755))
	_, err = libExec.Exec(wt.buildGitCommand(
		t.Context(), "mv", "app/pod.yaml", "different-app/pod.yaml",
	))
	require.NoError(t, err)
	require.NoError(
		t,
		rep.AddAllAndCommit(t.Context(), "move pod.yaml to different-app/", nil),
	)

	moveCommitID, err := rep.LastCommitID(t.Context())
	require.NoError(t, err)

	// GetDiffPathsForCommitID should return both the old and new paths so
	// that a warehouse watching app/ detects the removal.
	paths, err := rep.GetDiffPathsForCommitID(t.Context(), moveCommitID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"app/pod.yaml", "different-app/pod.yaml"}, paths)
}

func TestGetDiffPathsForCommitWithManyModifiedRenames(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	ctx := t.Context()

	rep, err := Clone(
		ctx,
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(ctx)

	const fileCount = 2000

	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/old", rep.Dir()), 0o755))
	for i := range fileCount {
		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/old/file%d.txt", rep.Dir(), i),
			[]byte(fmt.Sprintf("content %d", i)),
			0o600,
		))
	}
	require.NoError(t, rep.AddAllAndCommit(ctx, "initial commit", nil))

	// Move every file to a new directory and slightly modify its content, so
	// git can't pair renames using a cheap exact-content match and instead
	// falls back to similarity-based rename detection. Once a commit changes
	// enough files at once, git skips that detection and prints a warning to
	// stderr rather than failing -- GetDiffPathsForCommitID must not treat
	// that warning as a fatal error.
	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/new", rep.Dir()), 0o755))
	for i := range fileCount {
		require.NoError(t, os.Remove(fmt.Sprintf("%s/old/file%d.txt", rep.Dir(), i)))
		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/new/file%d.txt", rep.Dir(), i),
			[]byte(fmt.Sprintf("content %d, modified", i)),
			0o600,
		))
	}
	require.NoError(t, rep.AddAllAndCommit(ctx, "move and modify many files", nil))

	commitID, err := rep.LastCommitID(ctx)
	require.NoError(t, err)

	paths, err := rep.GetDiffPathsForCommitID(ctx, commitID)
	require.NoError(t, err)
	require.Len(t, paths, fileCount*2)
}

func TestWorkTree_FetchCommits(t *testing.T) {
	// main: initial -> commit-0 -> commit-1 -> commit-2
	// other: initial -> other
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	commitIDs := make([]string, 3)
	for i := range commitIDs {
		commitFile(t, author, fmt.Sprintf("commit-%d", i))
		var err error
		commitIDs[i], err = author.LastCommitID(t.Context())
		require.NoError(t, err)
	}
	olderCommitID, middleCommitID := commitIDs[0], commitIDs[1]
	require.NoError(t, author.Push(t.Context(), nil))
	latestCommitID, err := author.LastCommitID(t.Context())
	require.NoError(t, err)
	require.NoError(t, author.CreateChildBranch(t.Context(), "other"))
	commitFile(t, author, "other")
	require.NoError(t, author.Push(t.Context(), nil))
	// A server that does not allow fetching arbitrary commits by ID will still
	// serve this one, because a branch points to it.
	otherCommitID, err := author.LastCommitID(t.Context())
	require.NoError(t, err)

	cloneMain := func(t *testing.T, depth uint) *workTree {
		return cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: depth, Refs: []string{"refs/heads/main"}},
			"main",
		)
	}

	testCases := []struct {
		name      string
		depth     uint
		commitIDs []string
		assert    func(*testing.T, *workTree, map[string]struct{}, error)
	}{
		{
			name:      "not a full commit ID",
			commitIDs: []string{"abc1234"},
			assert: func(t *testing.T, _ *workTree, _ map[string]struct{}, err error) {
				require.ErrorContains(t, err, `"abc1234" is not a full commit ID`)
			},
		},
		{
			name:      "commits already present in a full clone",
			commitIDs: []string{olderCommitID, latestCommitID},
			assert: func(t *testing.T, wt *workTree, _ map[string]struct{}, err error) {
				require.NoError(t, err)
				shallow, err := wt.isShallow(t.Context())
				require.NoError(t, err)
				require.False(t, shallow)
			},
		},
		{
			name:      "missing commit fetched into a full clone with its history",
			commitIDs: []string{otherCommitID},
			assert: func(t *testing.T, wt *workTree, _ map[string]struct{}, err error) {
				require.NoError(t, err)
				missing, err := wt.missingCommits(t.Context(), []string{otherCommitID})
				require.NoError(t, err)
				require.Empty(t, missing)
				shallow, err := wt.isShallow(t.Context())
				require.NoError(t, err)
				require.False(t, shallow)
			},
		},
		{
			// commit-1 is present but not a shallow boundary. Fetching it without its
			// history would wrongly make it one.
			name:      "commit already present in a shallow clone",
			depth:     2,
			commitIDs: []string{middleCommitID},
			assert: func(
				t *testing.T,
				wt *workTree,
				shallowBefore map[string]struct{},
				err error,
			) {
				require.NoError(t, err)
				shallowAfter, err := wt.shallowCommits(t.Context())
				require.NoError(t, err)
				require.Equal(t, shallowBefore, shallowAfter)
			},
		},
		{
			name:      "missing commit fetched into a shallow clone without its history",
			depth:     1,
			commitIDs: []string{latestCommitID, otherCommitID},
			assert: func(t *testing.T, wt *workTree, _ map[string]struct{}, err error) {
				require.NoError(t, err)
				sigInfo, err := wt.GetCommitSignatureInfo(t.Context(), otherCommitID)
				require.NoError(t, err)
				require.NotNil(t, sigInfo)
				shallowAfter, err := wt.shallowCommits(t.Context())
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]struct{}{latestCommitID: {}, otherCommitID: {}},
					shallowAfter,
				)
			},
		},
		{
			name:      "commit the server will not serve by ID",
			depth:     1,
			commitIDs: []string{olderCommitID},
			assert: func(t *testing.T, _ *workTree, _ map[string]struct{}, err error) {
				require.ErrorContains(
					t, err, "server does not support fetching commits by ID",
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wt := cloneMain(t, testCase.depth)
			shallowBefore, err := wt.shallowCommits(t.Context())
			require.NoError(t, err)
			testCase.assert(
				t,
				wt,
				shallowBefore,
				wt.FetchCommits(t.Context(), testCase.commitIDs),
			)
		})
	}
}

func TestWorkTree_deepen(t *testing.T) {
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	for i := range 3 {
		commitFile(t, author, fmt.Sprintf("commit-%d", i))
	}
	require.NoError(t, author.Push(t.Context(), nil))

	t.Run("no-op for a full clone", func(t *testing.T) {
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Refs: []string{"refs/heads/main"}},
			"main",
		)
		require.NoError(t, wt.deepen(t.Context(), 1, "refs/heads/main"))
		// deepen must never turn a full clone into a shallow one
		shallow, err := wt.isShallow(t.Context())
		require.NoError(t, err)
		require.False(t, shallow)
	})

	t.Run("deepens a shallow clone", func(t *testing.T) {
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: 1, Refs: []string{"refs/heads/main"}},
			"main",
		)
		require.Equal(t, 1, commitCount(t, wt.baseRepo, "HEAD"))
		require.NoError(t, wt.deepen(t.Context(), 2, "refs/heads/main"))
		require.Equal(t, 3, commitCount(t, wt.baseRepo, "HEAD"))
	})

	t.Run("no-op for a depth of zero", func(t *testing.T) {
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: 1, Refs: []string{"refs/heads/main"}},
			"main",
		)
		shallowBefore, err := wt.shallowCommits(t.Context())
		require.NoError(t, err)
		require.NoError(t, wt.deepen(t.Context(), 0, "refs/heads/main"))
		shallowAfter, err := wt.shallowCommits(t.Context())
		require.NoError(t, err)
		require.Equal(t, shallowBefore, shallowAfter)
	})
}
