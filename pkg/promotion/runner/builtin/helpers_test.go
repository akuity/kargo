package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
)

// newTestStepContext returns a StepContext for workDir whose Root is open for
// the duration of the test.
func newTestStepContext(t *testing.T, workDir string) *promotion.StepContext {
	t.Helper()
	return &promotion.StepContext{WorkDir: workDir, Root: newTestRoot(t, workDir)}
}

// newTestRoot opens workDir as an os.Root that is closed when the test ends.
func newTestRoot(t *testing.T, workDir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(workDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })
	return root
}

// escapingWorkspace returns a workspace directory containing a relative
// symlink named "out" that points to a sibling directory outside the
// workspace, as a git checkout could contain, and the path of that outside
// directory.
func escapingWorkspace(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	outside := filepath.Join(base, "outside")
	require.NoError(t, os.Mkdir(workDir, 0o755))
	require.NoError(t, os.Mkdir(outside, 0o755))
	require.NoError(t, os.Symlink(filepath.Join("..", "outside"), filepath.Join(workDir, "out")))
	return workDir, outside
}
