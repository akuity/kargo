package git

import (
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
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()

	workingTreePath := filepath.Join(rep.HomeDir(), "working-tree")
	workTree, err := rep.AddWorkTree(
		workingTreePath,
		&AddWorkTreeOptions{Ref: "main"},
	)
	require.NoError(t, err)
	defer workTree.Close()

	t.Run("can load an existing working tree", func(t *testing.T) {
		existingWorkTree, err := LoadWorkTree(
			workTree.Dir(),
			&LoadWorkTreeOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, workTree, existingWorkTree)
	})

	t.Run("can close working tree", func(t *testing.T) {
		require.NoError(t, workTree.Close())
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
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close()

		// Make a local commit that diverges from "ahead".
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local divergent commit", nil)
		require.NoError(t, err)

		localCommit, err := repo.LastCommitID()
		require.NoError(t, err)

		// Force pull from "ahead" should reset to the remote state.
		err = repo.Pull(&PullOptions{Branch: "ahead", Force: true})
		require.NoError(t, err)

		newCommit, err := repo.LastCommitID()
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
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close()

		// Make a local commit on a different file to avoid conflicts.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local non-conflicting commit", nil)
		require.NoError(t, err)

		// Non-force pull from "ahead" should merge.
		err = repo.Pull(&PullOptions{Branch: "ahead"})
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
		msg, err := repo.CommitMessage("HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})

	t.Run("nil opts defaults to current branch", func(t *testing.T) {
		repo, err := Clone(
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close()

		// Pull with nil opts should not error (fetches current branch).
		err = repo.Pull(nil)
		require.NoError(t, err)
	})
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
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()

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
	require.NoError(t, rep.AddAllAndCommit("main: initial commit", nil))

	// Create a feature branch from the initial commit
	require.NoError(t, rep.CreateChildBranch("feature"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/feature.txt", rep.Dir()),
			[]byte("feature work 1"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("feature: work 1", nil))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/feature.txt", rep.Dir()),
			[]byte("feature work 2"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("feature: work 2", nil))

	// Back to main, add another commit
	require.NoError(t, rep.Checkout("main"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/file.txt", rep.Dir()),
			[]byte("second"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("main: second commit", nil))

	// Merge the feature branch into main
	_, err = libExec.Exec(wt.buildGitCommand(
		"merge", "feature", "--no-ff", "-m", "main: merge feature",
	))
	require.NoError(t, err)

	// ListCommits should only return first-parent commits (main line),
	// not the individual feature branch commits.
	commits, err := rep.ListCommits(nil)
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
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()

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
	require.NoError(t, rep.AddAllAndCommit("initial commit", nil))

	// Create branch-a and modify file1
	require.NoError(t, rep.CreateChildBranch("branch-a"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file1.txt", rep.Dir()),
			[]byte("changed by branch-a"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("branch-a: modify file1", nil))

	// Back to main, create branch-b, modify file2
	require.NoError(t, rep.Checkout("main"))
	require.NoError(t, rep.CreateChildBranch("branch-b"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file2.txt", rep.Dir()),
			[]byte("changed by branch-b"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("branch-b: modify file2", nil))

	// Merge branch-b into main
	require.NoError(t, rep.Checkout("main"))
	_, err = libExec.Exec(wt.buildGitCommand(
		"merge", "branch-b", "--no-ff", "-m", "merge branch-b",
	))
	require.NoError(t, err)

	// Merge branch-a into main
	_, err = libExec.Exec(wt.buildGitCommand(
		"merge", "branch-a", "--no-ff", "-m", "merge branch-a",
	))
	require.NoError(t, err)

	mergeCommitID, err := rep.LastCommitID()
	require.NoError(t, err)

	// GetDiffPathsForCommitID on the merge commit should return only
	// the file introduced by that merge (file1, from branch-a), not
	// file2 which was already on main via the earlier merge of branch-b.
	paths, err := rep.GetDiffPathsForCommitID(mergeCommitID)
	require.NoError(t, err)
	require.Equal(t, []string{"foo/file1.txt"}, paths)
}
