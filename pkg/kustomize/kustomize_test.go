package kustomize

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestBuild(t *testing.T) {
	configMapFiles := map[string]string{
		"kustomization.yaml": "resources:\n- configmap.yaml\n",
		"configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: example
`,
	}
	testCases := []struct {
		name  string
		files map[string]string
		use   func(ResMap) error
		// resources is the number of resources use saw, or -1 if use was not
		// called.
		assert func(t *testing.T, resources int, err error)
	}{
		{
			name: "build error",
			use:  func(ResMap) error { return nil },
			assert: func(t *testing.T, resources int, err error) {
				require.Error(t, err)
				require.Equal(t, -1, resources)
			},
		},
		{
			name:  "error from use",
			files: configMapFiles,
			use: func(ResMap) error {
				return errors.New("something went wrong")
			},
			assert: func(t *testing.T, resources int, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, 1, resources)
			},
		},
		{
			name:  "panic",
			files: configMapFiles,
			use:   func(ResMap) error { panic("something went wrong") },
			assert: func(t *testing.T, resources int, err error) {
				require.ErrorContains(
					t,
					err,
					"recovered from kustomize build panic: something went wrong",
				)
				require.Equal(t, 1, resources)
			},
		},
		{
			name:  "success",
			files: configMapFiles,
			use:   func(ResMap) error { return nil },
			assert: func(t *testing.T, resources int, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, resources)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range testCase.files {
				err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
				require.NoError(t, err)
			}
			resources := -1
			err := Build(
				filesys.MakeFsOnDisk(),
				dir,
				krusty.MakeDefaultOptions(),
				func(rm ResMap) error {
					resources = rm.Size()
					// The ResMap's methods read the schema, so use must run with the
					// schema lock held.
					if schemaMu.TryLock() {
						schemaMu.Unlock()
						return errors.New("use ran without the schema lock")
					}
					return testCase.use(rm)
				},
			)
			testCase.assert(t, resources, err)
		})
	}
}

func TestBuild_resetsOpenAPIBetweenBuilds(t *testing.T) {
	kusFS := filesys.MakeFsOnDisk()

	firstDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(firstDir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
openapi:
  path: schema.json
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(firstDir, "configmap.yaml"), []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: first-build
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(firstDir, "schema.json"), []byte(`
{"definitions": {}}
`), 0o600))

	err := Build(
		kusFS,
		firstDir,
		krusty.MakeDefaultOptions(),
		func(ResMap) error { return nil },
	)
	require.NoError(t, err)

	secondDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(secondDir, "component"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- widget.yaml
components:
- component
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "widget.yaml"), []byte(`
apiVersion: example.com/v1
kind: Widget
metadata:
  name: example
spec:
  items:
  - name: item
    value: preserved
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "component", "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
openapi:
  path: schema.json
patches:
- path: patch.yaml
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "component", "patch.yaml"), []byte(`
apiVersion: example.com/v1
kind: Widget
metadata:
  name: example
spec:
  items:
  - name: item
    changed: true
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "component", "schema.json"), []byte(`
{
  "definitions": {
    "example.com.v1.Widget": {
      "type": "object",
      "x-kubernetes-group-version-kind": [
        {
          "group": "example.com",
          "version": "v1",
          "kind": "Widget"
        }
      ],
      "properties": {
        "spec": {
          "type": "object",
          "properties": {
            "items": {
              "type": "array",
              "items": {
                "type": "object"
              },
              "x-kubernetes-patch-merge-key": "name",
              "x-kubernetes-patch-strategy": "merge"
            }
          }
        }
      }
    }
  }
}
`), 0o600))

	var result []byte
	err = Build(
		kusFS,
		secondDir,
		krusty.MakeDefaultOptions(),
		func(rm ResMap) error {
			var yamlErr error
			result, yamlErr = rm.AsYaml()
			return yamlErr
		},
	)
	require.NoError(t, err)
	assert.Contains(t, string(result), "value: preserved")
	assert.Contains(t, string(result), "changed: true")
}

func TestMerge(t *testing.T) {
	// Not parallel: some cases hold the package-level schema lock.
	testCases := []struct {
		name   string
		lock   func()
		unlock func()
		assert func(t *testing.T, done <-chan struct{}, unlock func())
	}{
		{
			name:   "nothing in progress",
			lock:   func() {},
			unlock: func() {},
			assert: func(t *testing.T, done <-chan struct{}, _ func()) {
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					require.Fail(t, "merge did not complete")
				}
			},
		},
		{
			name:   "waits while a build or merge is in progress",
			lock:   schemaMu.Lock,
			unlock: schemaMu.Unlock,
			assert: func(t *testing.T, done <-chan struct{}, unlock func()) {
				select {
				case <-done:
					require.Fail(t, "merge ran while the schema was in use")
				case <-time.After(100 * time.Millisecond):
				}
				unlock()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					require.Fail(t, "merge did not run after the schema was released")
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dest, err := yaml.Parse("a: 1\n")
			require.NoError(t, err)
			src, err := yaml.Parse("b: 2\n")
			require.NoError(t, err)

			var result *yaml.RNode
			var mergeErr error
			done := make(chan struct{})
			testCase.lock()
			unlock := sync.OnceFunc(testCase.unlock)
			t.Cleanup(func() {
				unlock()
				<-done
			})
			go func() {
				defer close(done)
				result, mergeErr = Merge(src, dest, yaml.MergeOptions{})
			}()
			testCase.assert(t, done, unlock)

			require.NoError(t, mergeErr)
			assert.Equal(t, "a: 1\nb: 2\n", result.MustString())
		})
	}
}
