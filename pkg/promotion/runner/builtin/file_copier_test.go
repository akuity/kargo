package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_filerCopier_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "inPath not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): inPath is required",
			},
		},
		{
			name: "inPath is empty string",
			config: promotion.Config{
				"inPath": "",
			},
			expectedProblems: []string{
				"inPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath not specified",
			config: promotion.Config{
				"inPath": "/source/path",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"inPath":  "/source/path",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "ignore is empty string",
			config: promotion.Config{
				"inPath":  "/source/path",
				"outPath": "/destination/path",
				"ignore":  "",
			},
			expectedProblems: []string{
				"ignore: String length must be greater than or equal to 1",
			},
		},
		{
			name: "both required fields missing",
			config: promotion.Config{
				"ignore": "*.log\n*.tmp",
			},
			expectedProblems: []string{
				"(root): inPath is required",
				"(root): outPath is required",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"inPath":  "/source/path",
				"outPath": "/destination/path",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with ignore patterns",
			config: promotion.Config{
				"inPath":  "/source/path",
				"outPath": "/destination/path",
				"ignore":  "*.log\n*.tmp\nnode_modules/\n.git/",
			},
			expectedProblems: nil,
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"inPath":  "/path/to/source/directory",
				"outPath": "/path/to/destination/directory",
				"ignore": `# Ignore log files
*.log
*.tmp

# Ignore build artifacts
build/
dist/

# Ignore version control
.git/
.svn/`,
			},
			expectedProblems: nil,
		},
	}

	r := newFileCopier(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*fileCopier)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_fileCopier_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        builtin.CopyConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "succeeds copying file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inPath := filepath.Join(tmpDir, "input.txt")
				require.NoError(t, os.WriteFile(inPath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input.txt",
				OutPath: "output.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))
			},
		},
		{
			name: "succeeds copying directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				nestedDir := filepath.Join(inDir, "nested")
				require.NoError(t, os.Mkdir(nestedDir, 0o755))
				nestedPath := filepath.Join(nestedDir, "nested.txt")
				require.NoError(t, os.WriteFile(nestedPath, []byte("nested content"), 0o600))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				nestedDir := filepath.Join(outDir, "nested")
				nestedPath := filepath.Join(nestedDir, "nested.txt")
				b, err = os.ReadFile(nestedPath)
				require.NoError(t, err)
				assert.Equal(t, "nested content", string(b))
			},
		},
		{
			name: "ignores symlink",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				symlinkPath := filepath.Join(inDir, "symlink.txt")
				require.NoError(t, os.Symlink("input.txt", symlinkPath))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				symlinkPath := filepath.Join(outDir, "symlink.txt")
				_, err = os.Lstat(symlinkPath)
				assert.Error(t, err)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "default ignore rule ignores .git directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git", "file.txt")
				require.NoError(t, os.MkdirAll(filepath.Dir(ignoreFilePath), 0o755))
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("ignored content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git", "file.txt")
				require.NoDirExists(t, filepath.Dir(ignoreFilePath))
				require.NoFileExists(t, ignoreFilePath)
			},
		},
		{
			name: "default ignore rule ignores .git file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git")
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("ignored content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git")
				require.NoFileExists(t, ignoreFilePath)
			},
		},
		{
			name: "overwrite default ignore rules",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git", "file.txt")
				require.NoError(t, os.MkdirAll(filepath.Dir(ignoreFilePath), 0o755))
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("included content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
				Ignore:  "!.git/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git", "file.txt")
				b, err = os.ReadFile(ignoreFilePath)
				require.NoError(t, err)
				assert.Equal(t, "included content", string(b))
			},
		},
		{
			name: "fails with invalid input path",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.CopyConfig{
				InPath: "input.txt",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				require.ErrorContains(t, err, "failed to copy")
			},
		},
	}

	runner := &fileCopier{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			result, err := runner.run(
				context.Background(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_fileCopier_loadIgnoreRules(t *testing.T) {
	tests := []struct {
		name       string
		inPath     string
		rules      string
		setup      func(*testing.T) string
		assertions func(*testing.T, string, gitignore.Matcher, error)
	}{
		{
			name:   "directory path",
			inPath: "testdir",
			rules: `*.txt
# comment
*.go`,
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(inPath, string(filepath.Separator))

				// Provided rules
				assert.True(t, matcher.Match(append(basePath, "file.txt"), false))
				assert.True(t, matcher.Match(append(basePath, "file.go"), false))
				assert.False(t, matcher.Match(append(basePath, "file.log"), false))

				// Default rules
				assert.True(t, matcher.Match(append(basePath, ".git"), true))
				assert.True(t, matcher.Match(append(basePath, ".git", "file.log"), true))
			},
		},
		{
			name:  "file path",
			rules: "*.log",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				inPath := filepath.Join(dir, "testfile.txt")
				assert.NoError(t, os.WriteFile(inPath, []byte("test"), 0o600))
				return inPath
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(filepath.Dir(inPath), string(filepath.Separator))
				assert.True(t, matcher.Match(append(basePath, "something.log"), false))
				assert.False(t, matcher.Match(append(basePath, "testfile.txt"), false))
				assert.True(t, matcher.Match(append(basePath, ".git", "file"), false))
			},
		},
		{
			name:   "non-existent path",
			inPath: "nonexistent",
			rules:  "*.tmp",
			setup: func(*testing.T) string {
				return "does-not-exist"
			},
			assertions: func(t *testing.T, _ string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
			},
		},
		{
			name:   "empty rules",
			inPath: "testdir",
			rules:  "",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(inPath, string(filepath.Separator))
				assert.True(t, matcher.Match(append(basePath, ".git", "file"), false))
				assert.False(t, matcher.Match(append(basePath, "file.txt"), false))
			},
		},
		{
			name: "invalid path",
			setup: func(*testing.T) string {
				return string([]byte{0})
			},
			rules: "*.tmp",
			assertions: func(t *testing.T, _ string, matcher gitignore.Matcher, err error) {
				assert.ErrorContains(t, err, "failed to determine domain")
				assert.Nil(t, matcher)
			},
		},
	}

	for _, tt := range tests {
		f := &fileCopier{}

		t.Run(tt.name, func(t *testing.T) {
			inPath := tt.setup(t)
			matcher, err := f.loadIgnoreRules(inPath, tt.rules)
			tt.assertions(t, inPath, matcher, err)
		})
	}
}
