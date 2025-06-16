package azure

import (
	"testing"

	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/stretchr/testify/require"
)

func TestParseRepoURL(t *testing.T) {
	testCases := []struct {
		name            string
		url             string
		expectedBaseUrl string
		expectedOrg     string
		expectedProj    string
		expectedRepo    string
		errExpected     bool
	}{
		{
			name:        "invalid URL",
			url:         "not-a-url",
			errExpected: true,
		},
		{
			name:        "unsupported host",
			url:         "https://github.com/org/repo",
			errExpected: true,
		},
		{
			name:        "modern URL with missing parts",
			url:         "https://dev.azure.com/org",
			errExpected: true,
		},
		{
			name:        "legacy URL with missing parts",
			url:         "https://org.visualstudio.com",
			errExpected: true,
		},
		{
			name:            "modern URL format",
			url:             "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedBaseUrl: "dev.azure.com",
			expectedOrg:     "myorg",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "modern URL format with .git suffix",
			url:             "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			expectedBaseUrl: "dev.azure.com",
			expectedOrg:     "myorg",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "legacy URL format",
			url:             "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expectedBaseUrl: "dev.azure.com",
			expectedOrg:     "myorg",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "legacy URL format with .git suffix",
			url:             "https://myorg.visualstudio.com/myproject/_git/myrepo.git",
			expectedBaseUrl: "dev.azure.com",
			expectedOrg:     "myorg",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "modern URL format with dot in repo name",
			url:             "https://dev.azure.com/myorg/myproject/_git/my.repo",
			expectedBaseUrl: "dev.azure.com",
			expectedOrg:     "myorg",
			expectedProj:    "myproject",
			expectedRepo:    "my.repo",
			errExpected:     false,
		},
		{
			name:            "self hosted URL format with missing parts",
			url:             "https://azure.mycompany.org/mycollection/myproject",
			expectedBaseUrl: "azure.mycompany.org",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     true,
		},
		{
			name:            "self hosted URL format with unsupported path segment foo",
			url:             "https://azure.mycompany.org/foo/mycollection/myproject/_git/myrepo",
			expectedBaseUrl: "azure.mycompany.org",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     true,
		},
		{
			name:            "self hosted URL format with 5 parts",
			url:             "https://azure.mycompany.org/mycollection/myproject/_git/myrepo",
			expectedBaseUrl: "azure.mycompany.org",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "self hosted URL format with 6 parts",
			url:             "https://azure.mycompany.org/tfs/mycollection/myproject/_git/myrepo",
			expectedBaseUrl: "azure.mycompany.org/tfs",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
		{
			name:            "self hosted URL format with unsupported path segment instead of _git",
			url:             "https://azure.mycompany.org/tfs/mycollection/myproject/git/myrepo",
			expectedBaseUrl: "azure.mycompany.org/tfs",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     true,
		},
		{
			name:            "self hosted URL format with 6 parts host does not contain providername",
			url:             "https://devops.mycompany.org/tfs/mycollection/myproject/_git/myrepo",
			expectedBaseUrl: "devops.mycompany.org/tfs",
			expectedOrg:     "mycollection",
			expectedProj:    "myproject",
			expectedRepo:    "myrepo",
			errExpected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host, org, proj, repo, err := parseRepoURL(tc.url)
			if tc.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedBaseUrl, host)
				require.Equal(t, tc.expectedOrg, org)
				require.Equal(t, tc.expectedProj, proj)
				require.Equal(t, tc.expectedRepo, repo)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	type args struct {
		repoURL string
		opts    *gitprovider.Options
	}
	testCases := []struct {
		name        string
		args        args
		wantErr     bool
		errContains string
		wantBaseUrl string
		wantProject string
		wantRepo    string
	}{
		{
			name:        "nil options",
			args:        args{repoURL: "https://dev.azure.com/org/proj/_git/repo", opts: nil},
			wantErr:     true,
			errContains: "token is required",
		},
		{
			name:        "empty token",
			args:        args{repoURL: "https://dev.azure.com/org/proj/_git/repo", opts: &gitprovider.Options{Token: ""}},
			wantErr:     true,
			errContains: "token is required",
		},
		{
			name:        "invalid repo url",
			args:        args{repoURL: "not-a-url", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     true,
			errContains: "invalid Azure DevOps Server URL",
		},
		{
			name:        "valid modern url",
			args:        args{repoURL: "https://dev.azure.com/org/proj/_git/repo", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     false,
			wantBaseUrl: "dev.azure.com",
			wantProject: "proj",
			wantRepo:    "repo",
		},
		{
			name:        "valid legacy url",
			args:        args{repoURL: "https://org.visualstudio.com/proj/_git/repo", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     false,
			wantBaseUrl: "dev.azure.com",
			wantProject: "proj",
			wantRepo:    "repo",
		},
		{
			name:        "valid self-hosted url",
			args:        args{repoURL: "https://azure.mycompany.org/mycollection/myproject/_git/myrepo", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     false,
			wantBaseUrl: "azure.mycompany.org",
			wantProject: "myproject",
			wantRepo:    "myrepo",
		},
		{
			name:        "valid self-hosted url",
			args:        args{repoURL: "https://azure.mycompany.org/tfs/mycollection/myproject/_git/myrepo", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     false,
			wantBaseUrl: "azure.mycompany.org/tfs",
			wantProject: "myproject",
			wantRepo:    "myrepo",
		},
		{
			name:        "invalid self-hosted url",
			args:        args{repoURL: "https://azure.mycompany.org/foo/bar", opts: &gitprovider.Options{Token: "token"}},
			wantErr:     true,
			errContains: "invalid Azure DevOps Server URL",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProvider(tt.args.repoURL, tt.args.opts)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				// Use type assertion to access internal fields for further validation
				p, ok := got.(*provider)
				require.True(t, ok)
				require.Equal(t, tt.wantProject, p.project)
				require.Equal(t, tt.wantRepo, p.repo)
				require.NotNil(t, p.connection)
				require.NotNil(t, tt.wantBaseUrl, p.connection.BaseUrl)
			}
		})
	}
}
