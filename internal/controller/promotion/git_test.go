package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewGitMechanism(t *testing.T) {
	const testName = "fake-name"
	pm := newGitMechanism(
		testName,
		fake.NewFakeClient(),
		&credentials.FakeDB{},
		func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
			return nil
		},
		func(
			context.Context,
			*kargoapi.Stage,
			*kargoapi.GitRepoUpdate,
			[]kargoapi.FreightReference,
			string, string, string,
			git.RepoCredentials,
		) ([]string, error) {
			return nil, nil
		},
	)
	gpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.Equal(t, testName, gpm.name)
	require.NotNil(t, gpm.client)
	require.NotNil(t, gpm.selectUpdatesFn)
	require.NotNil(t, gpm.doSingleUpdateFn)
	require.NotNil(t, gpm.getReadRefFn)
	require.NotNil(t, gpm.getAuthorFn)
	require.NotNil(t, gpm.getCredentialsFn)
	require.NotNil(t, gpm.gitCommitFn)
	require.NotNil(t, gpm.applyConfigManagementFn)
}

func TestGitGetName(t *testing.T) {
	const testName = "fake name"
	pm := newGitMechanism(testName, nil, nil, nil, nil)
	require.Equal(t, testName, pm.GetName())
}

func TestGitPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *gitMechanism
		assertions func(
			t *testing.T,
			status *kargoapi.PromotionStatus,
			newFreightIn []kargoapi.FreightReference,
			newFreightOut []kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name: "no updates",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error applying single update",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return []kargoapi.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					_ *kargoapi.Promotion,
					_ *kargoapi.GitRepoUpdate,
					newFreight []kargoapi.FreightReference,
				) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
					return nil, newFreight, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return []kargoapi.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Stage,
					_ *kargoapi.Promotion,
					_ *kargoapi.GitRepoUpdate,
					newFreight []kargoapi.FreightReference,
				) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
					return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, newFreight, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightIn := []kargoapi.FreightReference{}
			status, newFreightOut, err := testCase.promoMech.Promote(
				context.Background(),
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					},
				},
				&kargoapi.Promotion{},
				newFreightIn,
			)
			testCase.assertions(t, status, newFreightIn, newFreightOut, err)
		})
	}
}

func TestGitDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *gitMechanism
		assertions func(
			t *testing.T,
			status *kargoapi.PromotionStatus,
			newFreightIn []kargoapi.FreightReference,
			newFreightOut []kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name: "error getting readref",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					context.Context,
					client.Client,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
				) (string, *kargoapi.GitCommit, error) {
					return "", nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error getting repo credentials",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					context.Context,
					client.Client,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
				) (string, *kargoapi.GitCommit, error) {
					return testRef, nil, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error getting author",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					context.Context,
					client.Client,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
				) (string, *kargoapi.GitCommit, error) {
					return testRef, nil, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, errors.New("something went wrong")
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error committing change to repo",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					context.Context,
					client.Client,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
				) (string, *kargoapi.GitCommit, error) {
					return testRef, nil, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
					string,
					string,
					git.Repo,
					git.RepoCredentials,
				) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					_ context.Context,
					_ client.Client,
					_ *kargoapi.Stage,
					_ *kargoapi.GitRepoUpdate,
					freight []kargoapi.FreightReference,
				) (string, *kargoapi.GitCommit, error) {
					require.True(t, len(freight) > 0)
					require.True(t, len(freight[0].Commits) > 0)
					return testRef, &freight[0].Commits[0], nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.GitRepoUpdate,
					[]kargoapi.FreightReference,
					string,
					string,
					git.Repo,
					git.RepoCredentials,
				) (string, error) {
					return "fake-commit-id", nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn []kargoapi.FreightReference,
				newFreightOut []kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newFreightOut[0].Commits[0].HealthCheckCommit,
				)
				// The newFreight is otherwise unaltered
				newFreightIn[0].Commits[0].HealthCheckCommit = ""
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightIn := []kargoapi.FreightReference{{
				Commits: []kargoapi.GitCommit{{}},
			}}
			status, newFreightOut, err := testCase.promoMech.doSingleUpdate(
				context.Background(),
				&kargoapi.Stage{},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{Namespace: "fake-namespace"},
				},
				&kargoapi.GitRepoUpdate{RepoURL: "https://github.com/akuity/kargo"},
				newFreightIn,
			)
			testCase.assertions(t, status, newFreightIn, newFreightOut, err)
		})
	}
}

func TestGetReadRef(t *testing.T) {
	const testBranch = "fake-branch"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCommit := kargoapi.GitCommit{
		RepoURL: "fake-url",
		ID:      "fake-commit-id",
	}
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		freight    []kargoapi.FreightReference
		assertions func(
			t *testing.T,
			readBranch string,
			commit *kargoapi.GitCommit,
			err error,
		)
	}{
		{
			name: "update's RepoURL does not match any subscription",
			update: kargoapi.GitRepoUpdate{
				Origin:     &testOrigin,
				RepoURL:    "fake-url",
				ReadBranch: testBranch,
			},
			assertions: func(
				t *testing.T,
				readBranch string,
				commit *kargoapi.GitCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testBranch, readBranch)
				require.Nil(t, commit)
			},
		},
		{
			name: "success",
			update: kargoapi.GitRepoUpdate{
				Origin:  &testOrigin,
				RepoURL: "fake-url",
			},
			freight: []kargoapi.FreightReference{{
				Origin:  testOrigin,
				Commits: []kargoapi.GitCommit{testCommit},
			}},
			assertions: func(
				t *testing.T,
				_ string,
				commit *kargoapi.GitCommit,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, commit)
				require.Equal(t, testCommit, *commit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						GitRepoUpdates: []kargoapi.GitRepoUpdate{
							testCase.update,
						},
					},
				},
			}
			readBranch, commit, err := getReadRef(
				context.Background(),
				fake.NewFakeClient(),
				stage,
				&stage.Spec.PromotionMechanisms.GitRepoUpdates[0],
				testCase.freight,
			)
			testCase.assertions(t, readBranch, commit, err)
		})
	}
}

func TestGetRepoCredentials(t *testing.T) {
	testCases := []struct {
		name          string
		credentialsDB credentials.Database
		assertions    func(*testing.T, *git.RepoCredentials, error)
	}{
		{
			name: "error getting credentials from database",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{},
						false, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *git.RepoCredentials, err error) {
				require.ErrorContains(t, err, "error obtaining credentials")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no credentials found in database",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			assertions: func(t *testing.T, creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name: "credentials found in database",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "fake-username",
						Password: "fake-password",
					}, true, nil
				},
			},
			assertions: func(t *testing.T, creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "fake-username",
						Password: "fake-password",
					},
					creds,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := getRepoCredentialsFn(testCase.credentialsDB)(
				context.Background(),
				"fake-namespace",
				"fake-repo-url",
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestMoveRepoContents(t *testing.T) {
	const subdirCount = 50
	const fileCount = 50
	// Create dummy repo dir
	srcDir, err := createDummyRepoDir(t, subdirCount, fileCount)
	require.NoError(t, err)
	// Double-check the setup
	dirEntries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount+1)
	// Create destination dir
	destDir := t.TempDir()
	// Move
	err = moveRepoContents(srcDir, destDir)
	require.NoError(t, err)
	// .git should not have moved
	_, err = os.Stat(filepath.Join(srcDir, ".git"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(destDir, ".git"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	// Everything else should have moved
	dirEntries, err = os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)
	dirEntries, err = os.ReadDir(destDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount)
}

func TestDeleteRepoContents(t *testing.T) {
	const subdirCount = 50
	const fileCount = 50
	// Create dummy repo dir
	dir, err := createDummyRepoDir(t, subdirCount, fileCount)
	require.NoError(t, err)
	// Double-check the setup
	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount+1)
	// Delete
	err = deleteRepoContents(dir)
	require.NoError(t, err)
	// .git should not have been deleted
	_, err = os.Stat(filepath.Join(dir, ".git"))
	require.NoError(t, err)
	// Everything else should be deleted
	dirEntries, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)
}

func TestBuildCommitMessage(t *testing.T) {
	testCases := []struct {
		name          string
		changeSummary []string
		assertions    func(t *testing.T, msg string)
	}{
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "nil change summary",
			changeSummary: nil,
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "empty change summary",
			changeSummary: []string{},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			name: "change summary contains one item",
			changeSummary: []string{
				"fake-change",
			},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "fake-change", msg)
			},
		},
		{
			name: "change summary contains multiple items",
			changeSummary: []string{
				"fake-change",
				"another-fake-change",
			},
			assertions: func(t *testing.T, msg string) {
				require.Equal(
					t,
					[]string{
						"Kargo applied multiple changes",
						"",
						"Including:",
						"",
						"  * fake-change",
						"  * another-fake-change",
					},
					strings.Split(msg, "\n"),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, buildCommitMessage(testCase.changeSummary))
		})
	}
}

func createDummyRepoDir(t *testing.T, dirCount, fileCount int) (string, error) {
	t.Helper()
	// Create a temporary directory
	dir := t.TempDir()
	// Add a dummy .git/ subdir
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		return dir, err
	}
	// Add some dummy dirs
	for i := 0; i < dirCount; i++ {
		if err := os.Mkdir(
			filepath.Join(dir, fmt.Sprintf("dir-%d", i)),
			0755,
		); err != nil {
			return dir, err
		}
	}
	// Add some dummy files
	for i := 0; i < fileCount; i++ {
		file, err := os.Create(filepath.Join(dir, fmt.Sprintf("file-%d", i)))
		if err != nil {
			return dir, err
		}
		if err = file.Close(); err != nil {
			return dir, err
		}
	}
	return dir, nil
}
