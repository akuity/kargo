package git

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	testingPkg "github.com/akuity/kargo/api/testing"
	"github.com/akuity/kargo/pkg/types"
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
	testRepoCreds := RepoCredentials{
		Username: "fake-username",
		Password: "fake-password",
	}

	// This will be something to opt into because on some OSes, this will lead
	// to keychain-related prompts.
	var useAuth bool
	if useAuthStr := os.Getenv("TEST_GIT_CLIENT_WITH_AUTH"); useAuthStr != "" {
		useAuth = types.MustParseBool(useAuthStr)
	}
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
			Auth:       useAuth,
		},
	)
	require.NoError(t, service.Setup())
	service.AuthFunc =
		func(cred gitkit.Credential, _ *gitkit.Request) (bool, error) {
			return cred.Username == testRepoCreds.Username &&
				cred.Password == testRepoCreds.Password, nil
		}
	server := httptest.NewServer(service)
	defer server.Close()

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	setupRep, err := Clone(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, setupRep)
	defer setupRep.Close()
	err = os.WriteFile(fmt.Sprintf("%s/%s", setupRep.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = setupRep.AddAllAndCommit(fmt.Sprintf("initial commit %s", uuid.NewString()), nil)
	require.NoError(t, err)
	err = setupRep.Push(nil)
	require.NoError(t, err)
	err = setupRep.Close()
	require.NoError(t, err)

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
		// "master" is still the default branch name for a new repository unless
		// you configure it otherwise.
		&AddWorkTreeOptions{Ref: "master"},
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
