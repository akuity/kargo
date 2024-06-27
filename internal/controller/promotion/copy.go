package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/fs"
	libGit "github.com/akuity/kargo/internal/git"
)

// newCopyMechanism returns a gitMechanism that only selects and performs
// copy-related updates.
func newCopyMechanism(
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Copy promotion mechanism",
		credentialsDB,
		selectCopyUpdates,
		applyCopyPatches,
	)
}

// selectCopyUpdates returns a subset of the given updates that involve copying.
func selectCopyUpdates(updates []kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
	selectedUpdates := make([]kargoapi.GitRepoUpdate, 0, len(updates))
	for _, update := range updates {
		if len(update.Patches) > 0 {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// applyCopyPatches applies the copy patches from the updates to the given Git
// repository. The source directory for the copy operation is determined based
// on whether the patch specifies a RepoURL. If a RepoURL is specified, the
// source directory is the working directory of the corresponding repository in
// the map of Freight repositories. If no RepoURL is specified, the source
// directory is the working directory of the given repository. The function
// returns a slice of strings describing the changes made, or an error if any
// of the copy operations fail.
func applyCopyPatches(
	_ context.Context,
	update kargoapi.GitRepoUpdate,
	_ kargoapi.FreightReference,
	_ string,
	_ string,
	_ string,
	workingDir string,
	freightRepos map[string]string,
	_ git.RepoCredentials,
) ([]string, error) {
	changes := make([]string, 0, len(update.Patches))
	for _, patch := range update.Patches {
		if patch.Copy == nil {
			continue
		}

		sourceDir := workingDir
		if patch.Copy.RepoURL != "" {
			var ok bool
			if sourceDir, ok = freightRepos[libGit.NormalizeURL(patch.Copy.RepoURL)]; !ok {
				return nil, fmt.Errorf("no Freight repository found for URL %q", patch.Copy.RepoURL)
			}
		}
		if err := applyCopyPatch(sourceDir, workingDir, *patch.Copy); err != nil {
			return nil, fmt.Errorf("error performing copy operation: %w", err)
		}

		if patch.Copy.RepoURL != "" {
			changes = append(changes, fmt.Sprintf(
				"copied %s from %s to %s",
				patch.Copy.Source, patch.Copy.RepoURL, patch.Copy.Destination),
			)
		} else {
			changes = append(changes, fmt.Sprintf("copied %s to %s", patch.Copy.Source, patch.Copy.Destination))
		}
	}
	return changes, nil
}

// applyCopyPatch applies a single CopyPatchOperation to the target directory.
// If the source path is a file, it is copied to the destination path. If the
// source path is a directory, it is copied recursively to the destination path.
// The function returns an error if the operation fails.
func applyCopyPatch(sourceDir, targetDir string, patch kargoapi.CopyPatchOperation) error {
	// Ensure the source path is within the repository working directory
	srcPath := filepath.Join(sourceDir, patch.Source)
	if !fs.WithinBasePath(sourceDir, srcPath) {
		return fmt.Errorf("source path %q is not within the repository root", patch.Source)
	}

	// Ensure the destination path is within the repository working directory.
	dstPath, err := securejoin.SecureJoin(targetDir, patch.Destination)
	if err != nil {
		return fmt.Errorf("error resolving destination path %q: %w", patch.Destination, err)
	}

	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("error getting info for source path %q: %w", patch.Source, err)
	}

	switch {
	case srcInfo.Mode().IsRegular():
		if err = os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("error creating destination directory: %w", err)
		}
		if err = fs.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("error copying file %q to %q: %w", patch.Source, patch.Destination, err)
		}
		return nil
	case srcInfo.IsDir():
		if err = os.MkdirAll(dstPath, 0o755); err != nil {
			return fmt.Errorf("error creating destination directory: %w", err)
		}
		if err = fs.CopyDir(srcPath, dstPath); err != nil {
			return fmt.Errorf("error copying directory %q to %q: %w", patch.Source, patch.Destination, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported file type for source path %q", patch.Source)
	}
}
