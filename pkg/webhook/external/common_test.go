package external

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const testSigningKey = "mysupersecrettoken"

func sign(content []byte) string {
	mac := hmac.New(sha256.New, []byte(testSigningKey))
	_, _ = mac.Write(content)
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}

func signWithoutAlgoPrefix(content []byte) string {
	mac := hmac.New(sha256.New, []byte(testSigningKey))
	_, _ = mac.Write(content)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestCollectPaths(t *testing.T) {
	type commit struct {
		files []string
	}
	getPaths := func(c commit) []string {
		return c.files
	}
	testCases := []struct {
		name    string
		commits []commit
		assert  func(*testing.T, []string)
	}{
		{
			name:    "no commits",
			commits: nil,
			assert: func(t *testing.T, paths []string) {
				require.Nil(t, paths)
			},
		},
		{
			name: "single commit with no paths",
			commits: []commit{
				{files: nil},
			},
			assert: func(t *testing.T, paths []string) {
				require.Nil(t, paths)
			},
		},
		{
			name: "single commit with paths",
			commits: []commit{
				{files: []string{"a.txt", "b.txt"}},
			},
			assert: func(t *testing.T, paths []string) {
				require.Equal(t, []string{"a.txt", "b.txt"}, paths)
			},
		},
		{
			name: "multiple commits with unique paths",
			commits: []commit{
				{files: []string{"a.txt"}},
				{files: []string{"b.txt"}},
			},
			assert: func(t *testing.T, paths []string) {
				require.Equal(t, []string{"a.txt", "b.txt"}, paths)
			},
		},
		{
			name: "multiple commits with duplicate paths",
			commits: []commit{
				{files: []string{"a.txt", "b.txt"}},
				{files: []string{"b.txt", "c.txt"}},
			},
			assert: func(t *testing.T, paths []string) {
				require.Equal(
					t, []string{"a.txt", "b.txt", "c.txt"}, paths,
				)
			},
		},
		{
			name: "results are sorted",
			commits: []commit{
				{files: []string{"c.txt", "a.txt"}},
				{files: []string{"b.txt"}},
			},
			assert: func(t *testing.T, paths []string) {
				require.Equal(
					t, []string{"a.txt", "b.txt", "c.txt"}, paths,
				)
			},
		},
		{
			name: "all duplicates",
			commits: []commit{
				{files: []string{"a.txt"}},
				{files: []string{"a.txt"}},
				{files: []string{"a.txt"}},
			},
			assert: func(t *testing.T, paths []string) {
				require.Equal(t, []string{"a.txt"}, paths)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			paths := collectPaths(testCase.commits, getPaths)
			testCase.assert(t, paths)
		})
	}
}
