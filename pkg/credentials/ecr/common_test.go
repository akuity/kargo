package ecr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ecrURLRegex(t *testing.T) {
	testCases := []struct {
		name          string
		repoURL       string
		wantMatch     bool
		wantAccountID string
		wantRegion    string
	}{
		{
			name:          "standard ECR URL",
			repoURL:       "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			wantMatch:     true,
			wantAccountID: "123456789012",
			wantRegion:    "us-west-2",
		},
		{
			name:          "cross-account ECR URL",
			repoURL:       "114557438263.dkr.ecr.eu-west-1.amazonaws.com/oidc-login-app",
			wantMatch:     true,
			wantAccountID: "114557438263",
			wantRegion:    "eu-west-1",
		},
		{
			name:          "nested repository path",
			repoURL:       "123456789012.dkr.ecr.eu-central-1.amazonaws.com/team/service/image",
			wantMatch:     true,
			wantAccountID: "123456789012",
			wantRegion:    "eu-central-1",
		},
		{
			name:      "not an ECR URL",
			repoURL:   "not-an-ecr-url",
			wantMatch: false,
		},
		{
			name:      "GAR URL",
			repoURL:   "us-docker.pkg.dev/project/repo/image",
			wantMatch: false,
		},
		{
			name:      "ECR host without repository path",
			repoURL:   "123456789012.dkr.ecr.us-west-2.amazonaws.com",
			wantMatch: false,
		},
		{
			name:      "account ID too short",
			repoURL:   "12345678901.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			wantMatch: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matches := ecrURLRegex.FindStringSubmatch(testCase.repoURL)
			if !testCase.wantMatch {
				assert.Nil(t, matches)
				return
			}
			// A successful match yields the full match plus two capture
			// groups: account ID (group 1) and region (group 2).
			assert.Len(t, matches, 3)
			assert.Equal(t, testCase.wantAccountID, matches[1])
			assert.Equal(t, testCase.wantRegion, matches[2])
		})
	}
}

func Test_tokenCacheKey(t *testing.T) {
	testCases := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "single part",
			parts: []string{"region1"},
			want:  "7507acda9c58034d4f38545edd121b4c8572483cbc5c7dc40f3daa2c74d8430a",
		},
		{
			name:  "multiple parts",
			parts: []string{"region1", "key1", "secret1"},
			want:  "559495d4cca6055810e755d40dfdeeb1aa0a937f3030463be970b9cd2d586002",
		},
		{
			name:  "no parts",
			parts: []string{},
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for i := 0; i < 100000; i++ {
				result := tokenCacheKey(testCase.parts...)
				assert.Equal(t, testCase.want, result)
			}
		})
	}
}
