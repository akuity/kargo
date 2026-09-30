package builtin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	securefs "github.com/fluxcd/pkg/kustomize/filesys"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func Test_validateQueryValues(t *testing.T) {
	testCases := []struct {
		name   string
		entry  string
		assert func(*testing.T, error)
	}{
		{
			name:  "local path",
			entry: "../base",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "remote without query",
			entry: "https://github.com/example/repo//base",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "remote with safe ref",
			entry: "https://github.com/example/repo//base?ref=main",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "remote with safe version and other params",
			entry: "https://github.com/example/repo//base?version=v1.2.3&timeout=90s",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "unparseable query",
			entry: "https://github.com/example/repo?ref=%zz",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "value containing but not beginning with -",
			entry: "https://github.com/example/repo?ref=release-1.0&timeout=1m",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "ref beginning with -",
			entry: "https://github.com/example/repo?ref=--upload-pack=touch /tmp/pwned",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter ref "--upload-pack=touch /tmp/pwned"`)
			},
		},
		{
			name:  "version beginning with -",
			entry: "https://github.com/example/repo?version=-x",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter version "-x"`)
			},
		},
		{
			name:  "percent-encoded ref beginning with -",
			entry: "https://github.com/example/repo?ref=%2D%2Dupload-pack%3Dx",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter ref "--upload-pack=x"`)
			},
		},
		{
			name:  "safe ref followed by unsafe duplicate",
			entry: "https://github.com/example/repo?ref=main&ref=--upload-pack=x",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter ref "--upload-pack=x"`)
			},
		},
		{
			name:  "safe ref with unsafe version",
			entry: "https://github.com/example/repo?ref=main&version=--upload-pack=x",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter version "--upload-pack=x"`)
			},
		},
		{
			name:  "other known parameter beginning with -",
			entry: "https://github.com/example/repo?ref=main&timeout=-5",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter timeout "-5"`)
			},
		},
		{
			name:  "unknown parameter beginning with -",
			entry: "https://github.com/example/repo?ref=main&future=--upload-pack=x",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter future "--upload-pack=x"`)
			},
		},
		{
			name:  "local path with parameter beginning with -",
			entry: "../base?foo=-bar",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter foo "-bar"`)
			},
		},
		{
			name:  "partially unparseable query with parameter beginning with -",
			entry: "https://github.com/example/repo?bad=%zz&ref=--upload-pack=x",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `specifies query parameter ref "--upload-pack=x"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, validateQueryValues(testCase.entry))
		})
	}
}

func Test_kustomizeValidatorFS_ReadFile(t *testing.T) {
	const badRef = "https://github.com/example/repo?ref=--upload-pack=x"
	testCases := []struct {
		name     string
		fileName string
		content  string
		assert   func(t *testing.T, content []byte, err, recordedErr error)
	}{
		{
			name:     "file does not exist",
			fileName: "",
			assert: func(t *testing.T, _ []byte, err, recordedErr error) {
				require.Error(t, err)
				require.NoError(t, recordedErr)
			},
		},
		{
			name:     "non-kustomization file is not inspected",
			fileName: "resources.yaml",
			content:  "resources:\n- " + badRef + "\n",
			assert: func(t *testing.T, content []byte, err, recordedErr error) {
				require.NoError(t, err)
				require.NoError(t, recordedErr)
				require.Contains(t, string(content), badRef)
			},
		},
		{
			name:     "unparseable kustomization is passed through",
			fileName: "kustomization.yaml",
			content:  "unknownField: " + badRef + "\n",
			assert: func(t *testing.T, content []byte, err, recordedErr error) {
				require.NoError(t, err)
				require.NoError(t, recordedErr)
				require.Contains(t, string(content), badRef)
			},
		},
		{
			name:     "safe kustomization is passed through",
			fileName: "kustomization.yaml",
			content: `resources:
- ../base
- https://github.com/example/repo?ref=main
`,
			assert: func(t *testing.T, content []byte, err, recordedErr error) {
				require.NoError(t, err)
				require.NoError(t, recordedErr)
				require.Contains(t, string(content), "ref=main")
			},
		},
	}
	for _, field := range []string{
		"resources",
		"bases",
		"components",
		"generators",
		"transformers",
		"validators",
	} {
		for _, fileName := range []string{
			"kustomization.yaml",
			"kustomization.yml",
			"Kustomization",
		} {
			testCases = append(testCases, struct {
				name     string
				fileName string
				content  string
				assert   func(t *testing.T, content []byte, err, recordedErr error)
			}{
				name:     fmt.Sprintf("unsafe ref in %s of %s", field, fileName),
				fileName: fileName,
				content:  fmt.Sprintf("%s:\n- %s\n", field, badRef),
				assert: func(t *testing.T, content []byte, err, recordedErr error) {
					require.ErrorContains(t, err, "must not begin with")
					require.Nil(t, content)
					require.Equal(t, err, recordedErr)
				},
			})
		}
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			memFS := filesys.MakeFsInMemory()
			path := filepath.Join("/work", "missing")
			if testCase.fileName != "" {
				path = filepath.Join("/work", testCase.fileName)
				require.NoError(t, memFS.WriteFile(path, []byte(testCase.content)))
			}
			validatingFS := newKustomizeValidatorFS(memFS)
			content, err := validatingFS.ReadFile(path)
			testCase.assert(t, content, err, validatingFS.Err())
		})
	}
}

// Test_kustomizeBuild_remoteGitRefInjection is a regression test for
// GHSA-927c-fm24-cq67. Kustomize passes the ref of a remote entry to
// `git fetch` as a literal argument, so a ref such as --upload-pack=<cmd>
// causes git to execute <cmd>. kustomizeBuild must reject such refs -- both
// directly and in remote entries referenced by other remote entries -- while
// still resolving remote entries with safe refs.
func Test_kustomizeBuild_remoteGitRefInjection(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not available")
	}

	// A script git would execute as its upload-pack helper if an injected
	// option were honored.
	markerPath := filepath.Join(t.TempDir(), "pwned")
	scriptPath := filepath.Join(t.TempDir(), "upload-pack.sh")
	require.NoError(t, os.WriteFile(
		scriptPath,
		[]byte("#!/bin/sh\ntouch "+markerPath+"\nexit 1\n"),
		0o700, // nolint: gosec
	))
	maliciousRef := "?ref=--upload-pack=" + scriptPath

	// A repository that generates a single, harmless resource.
	safeRepo := newTestGitRepo(t, gitPath, `configMapGenerator:
- name: remote
  literals:
  - key=value
`)
	// A repository whose own kustomization contains a malicious ref.
	maliciousRepo := newTestGitRepo(t, gitPath, fmt.Sprintf(
		"resources:\n- file://%s%s\n",
		safeRepo, maliciousRef,
	))

	testCases := []struct {
		name     string
		resource string
		assert   func(*testing.T, error)
	}{
		{
			name:     "remote resource with safe ref",
			resource: "file://" + safeRepo + "?ref=main",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "remote resource with malicious ref",
			resource: "file://" + safeRepo + maliciousRef,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "must not begin with")
			},
		},
		{
			name:     "remote resource that references a malicious ref",
			resource: "file://" + maliciousRepo + "?ref=main",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "must not begin with")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workDir := t.TempDir()
			require.NoError(t, os.WriteFile(
				filepath.Join(workDir, "kustomization.yaml"),
				[]byte("resources:\n- "+testCase.resource+"\n"),
				0o600,
			))
			diskFS, err := securefs.MakeFsOnDiskSecureBuild(workDir)
			require.NoError(t, err)

			_, err = kustomizeBuild(diskFS, workDir, nil)
			require.NoFileExists(t, markerPath)
			testCase.assert(t, err)
		})
	}
}

// newTestGitRepo creates a Git repository with a main branch containing a
// kustomization.yaml with the given content, and returns its path.
func newTestGitRepo(t *testing.T, gitPath, kustomization string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "kustomization.yaml"),
		[]byte(kustomization),
		0o600,
	))
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"add", "."},
		{
			"-c", "user.name=test",
			"-c", "user.email=test@example.com",
			"-c", "commit.gpgsign=false",
			"commit", "-m", "initial commit",
		},
	} {
		cmd := exec.Command(gitPath, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	return dir
}
