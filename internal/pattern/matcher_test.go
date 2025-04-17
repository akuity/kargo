package pattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNamePattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "exact pattern",
			pattern: "exact",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &ExactMatcher{}, matcher)
				assert.Equal(t, "exact", matcher.String())
			},
		},
		{
			name:    "glob pattern",
			pattern: "glob:*.txt",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &GlobMatcher{}, matcher)
				assert.Equal(t, "*.txt", matcher.String())
			},
		},
		{
			name:    "glob pattern with invalid syntax",
			pattern: "glob:[",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
		{
			name:    "regex pattern",
			pattern: "regex:^dev-[a-zA-Z0-9]*",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &RegexpMatcher{}, matcher)
				assert.Equal(t, "^dev-[a-zA-Z0-9]*", matcher.String())
			},
		},
		{
			name:    "regex pattern with invalid syntax",
			pattern: "regex:[a-z",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
		{
			name:    "regexp pattern",
			pattern: "regexp:^dev-[a-zA-Z0-9]*",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &RegexpMatcher{}, matcher)
				assert.Equal(t, "^dev-[a-zA-Z0-9]*", matcher.String())
			},
		},
		{
			name:    "regexp pattern with invalid syntax",
			pattern: "regexp:[a-z",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher, err := ParseNamePattern(test.pattern)
			test.assertions(t, matcher, err)
		})
	}
}

func TestParsePathPattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "base directory pattern",
			pattern: "base/dir",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &BaseDirMatcher{}, matcher)
				assert.Equal(t, "base/dir", matcher.String())
			},
		},
		{
			name:    "empty base directory pattern",
			pattern: "",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &BaseDirMatcher{}, matcher)
				assert.Equal(t, "", matcher.String())
			},
		},
		{
			name:    "glob pattern",
			pattern: "glob:*.txt",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &GlobMatcher{}, matcher)
				assert.Equal(t, "*.txt", matcher.String())
			},
		},
		{
			name:    "glob pattern with invalid syntax",
			pattern: "glob:[",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
		{
			name:    "regex pattern for paths",
			pattern: "regex:^[a-z]+/config.yaml$",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &RegexpMatcher{}, matcher)
				assert.Equal(t, "^[a-z]+/config.yaml$", matcher.String())
			},
		},
		{
			name:    "regex pattern with invalid syntax",
			pattern: "regex:[a-z",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
		{
			name:    "regexp pattern",
			pattern: "regexp:^[a-z]+/config.yaml$",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
				assert.IsType(t, &RegexpMatcher{}, matcher)
				assert.Equal(t, "^[a-z]+/config.yaml$", matcher.String())
			},
		},
		{
			name:    "regexp pattern with invalid syntax",
			pattern: "regexp:[a-z",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher, err := ParsePathPattern(test.pattern)
			test.assertions(t, matcher, err)
		})
	}
}

func TestMatchers(t *testing.T) {
	exactPattern, _ := NewExactMatcher("config.yaml")
	globPattern, _ := NewGlobPattern("*.{yaml,yml}")
	regexpPattern, _ := NewRegexpMatcher("^app-[a-z]+$")
	baseDirPattern, _ := NewBaseDirMatcher("configs")

	tests := []struct {
		name       string
		matchers   Matchers
		assertions func(*testing.T, Matchers)
	}{
		{
			name:     "empty matchers",
			matchers: Matchers{},
			assertions: func(t *testing.T, m Matchers) {
				// Verify no matches with empty matchers
				assert.False(t, m.Matches("anything"))
				assert.False(t, m.Matches(""))

				// Verify string representation
				assert.Equal(t, "", m.String())
			},
		},
		{
			name:     "single matcher",
			matchers: Matchers{exactPattern},
			assertions: func(t *testing.T, m Matchers) {
				// Verify matches
				assert.True(t, m.Matches("config.yaml"))

				// Verify non-matches
				assert.False(t, m.Matches("other.yaml"))
				assert.False(t, m.Matches("config.yml"))

				// Verify string representation
				assert.Equal(t, "config.yaml", m.String())
			},
		},
		{
			name:     "multiple matchers",
			matchers: Matchers{exactPattern, globPattern, regexpPattern},
			assertions: func(t *testing.T, m Matchers) {
				// Verify matches - should match if ANY matcher matches
				assert.True(t, m.Matches("config.yaml")) // matches exact pattern
				assert.True(t, m.Matches("other.yaml"))  // matches glob pattern
				assert.True(t, m.Matches("app-service")) // matches regexp pattern
				assert.True(t, m.Matches("service.yml")) // matches glob pattern

				// Verify non-matches
				assert.False(t, m.Matches("data.json"))
				assert.False(t, m.Matches("app_service"))
				assert.False(t, m.Matches("App-service"))

				// Verify string representation
				assert.Equal(t, "config.yaml, *.{yaml,yml}, ^app-[a-z]+$", m.String())
			},
		},
		{
			name:     "different matcher types",
			matchers: Matchers{exactPattern, baseDirPattern},
			assertions: func(t *testing.T, m Matchers) {
				// Verify matches
				assert.True(t, m.Matches("config.yaml"))       // matches exact pattern
				assert.True(t, m.Matches("configs/file.json")) // matches base dir pattern
				assert.True(t, m.Matches("configs"))           // matches base dir pattern

				// Verify non-matches
				assert.False(t, m.Matches("settings.yaml"))
				assert.False(t, m.Matches("other/configs/file.json"))

				// Verify string representation
				assert.Equal(t, "config.yaml, configs", m.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.matchers)
		})
	}
}

func TestGlobMatcher(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "file wildcard",
			pattern: "*.yaml",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("deployment.yaml"))
				assert.True(t, matcher.Matches("service.yaml"))
				assert.True(t, matcher.Matches("configmap.yaml"))

				// Verify non-matches
				assert.False(t, matcher.Matches("deployment.yml"))
				assert.False(t, matcher.Matches("deployment.json"))
				assert.False(t, matcher.Matches("yaml"))
			},
		},
		{
			name:    "directory wildcard",
			pattern: "env/*/deployment.yaml",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("env/prod/deployment.yaml"))
				assert.True(t, matcher.Matches("env/dev/deployment.yaml"))

				// Verify non-matches
				assert.False(t, matcher.Matches("env/prod/nested/deployment.yaml"))
				assert.False(t, matcher.Matches("app/guestbook/deployment.yml"))
			},
		},
		{
			name:    "recursive superglob",
			pattern: "env/**/deployment.yaml",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("env/nested/deployment.yaml"))
				assert.True(t, matcher.Matches("env/even/more/nested/deployment.yaml"))
				assert.True(t, matcher.Matches("env/can/it/be/even/more/nested/default/apps/deployment.yaml"))

				// Verify non-matches
				assert.False(t, matcher.Matches("env/nested/service.yml"))
				assert.False(t, matcher.Matches("etc/deployment.yaml"))
			},
		},
		{
			name:    "pattern with character class",
			pattern: "app-[a-z].yaml",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("app-a.yaml"))
				assert.True(t, matcher.Matches("app-z.yaml"))

				// Verify non-matches
				assert.False(t, matcher.Matches("app-ab.yaml"))
				assert.False(t, matcher.Matches("app-1.yaml"))
				assert.False(t, matcher.Matches("app-A.yaml"))
			},
		},
		{
			name:    "multiple resource types",
			pattern: "*.{yaml,yml,json}",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				assert.True(t, matcher.Matches("deployment.yaml"))
				assert.True(t, matcher.Matches("service.yml"))
				assert.True(t, matcher.Matches("configmap.json"))

				assert.False(t, matcher.Matches("deployment.xml"))
				assert.False(t, matcher.Matches("manifest.jsonnet"))
			},
		},
		{
			name:    "invalid glob pattern",
			pattern: "[",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewGlobPattern(tt.pattern)
			tt.assertions(t, matcher, err)
		})
	}
}

func TestRegexpMatcher(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "simple word pattern",
			pattern: "app",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("app"))
				assert.True(t, matcher.Matches("application"))
				assert.True(t, matcher.Matches("happy"))

				// Verify non-matches
				assert.False(t, matcher.Matches("API"))
				assert.False(t, matcher.Matches("App"))
			},
		},
		{
			name:    "anchored pattern",
			pattern: "^app$",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("app"))

				// Verify non-matches
				assert.False(t, matcher.Matches("application"))
				assert.False(t, matcher.Matches("happy"))
				assert.False(t, matcher.Matches("APP"))
			},
		},
		{
			name:    "kubernetes resource pattern",
			pattern: "^(deployment|service|configmap)-[a-z0-9]+-[a-z0-9]+$",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("deployment-app-prod"))
				assert.True(t, matcher.Matches("service-api-dev"))
				assert.True(t, matcher.Matches("configmap-db-test"))

				// Verify non-matches
				assert.False(t, matcher.Matches("ingress-app-prod"))
				assert.False(t, matcher.Matches("deployment-APP-prod"))
				assert.False(t, matcher.Matches("deployment-app"))
				assert.False(t, matcher.Matches("deployment_app_prod"))
			},
		},
		{
			name:    "semantic version pattern",
			pattern: "^v\\d+\\.\\d+\\.\\d+$",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("v1.0.0"))
				assert.True(t, matcher.Matches("v0.1.0"))
				assert.True(t, matcher.Matches("v12.34.56"))

				// Verify non-matches
				assert.False(t, matcher.Matches("version1.0.0"))
				assert.False(t, matcher.Matches("v1.0"))
				assert.False(t, matcher.Matches("v1.0.0-beta"))
				assert.False(t, matcher.Matches("1.0.0"))
			},
		},
		{
			name:    "invalid regexp",
			pattern: "(unclosed",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.Error(t, err)
				assert.Nil(t, matcher)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewRegexpMatcher(tt.pattern)
			tt.assertions(t, matcher, err)
		})
	}
}

func TestBaseDirMatcher(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "simple base directory",
			pattern: "configs",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("configs/app.yaml"))
				assert.True(t, matcher.Matches("configs/dev/database.yaml"))
				assert.True(t, matcher.Matches("configs"))

				// Verify non-matches
				assert.False(t, matcher.Matches("app/configs/settings.yaml"))
				assert.False(t, matcher.Matches("configurations/app.yaml"))
				assert.False(t, matcher.Matches("../configs/app.yaml"))
			},
		},
		{
			name:    "nested base directory",
			pattern: "environments/prod",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("environments/prod/deployment.yaml"))
				assert.True(t, matcher.Matches("environments/prod/configs/database.yaml"))
				assert.True(t, matcher.Matches("environments/prod"))

				// Verify non-matches
				assert.False(t, matcher.Matches("environments/dev/deployment.yaml"))
				assert.False(t, matcher.Matches("environments"))
				assert.False(t, matcher.Matches("apps/environments/prod/config.yaml"))
				assert.False(t, matcher.Matches("../environments/prod/config.yaml"))
			},
		},
		{
			name:    "with relative path components",
			pattern: "config/shared",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("config/shared/common.yaml"))
				assert.True(t, matcher.Matches("config/shared"))

				// Verify non-matches
				assert.False(t, matcher.Matches("config/shared/../private/secrets.yaml"))
				assert.False(t, matcher.Matches("config/other/file.yaml"))
				assert.False(t, matcher.Matches("app/config/shared/file.yaml"))
			},
		},
		{
			name:    "handling invalid paths",
			pattern: "valid/path",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Test with paths that would cause filepath.Rel to error
				assert.False(t, matcher.Matches("invalid:path"))
				assert.False(t, matcher.Matches("\x00malformed"))

				// Test with paths that escape base directory
				assert.False(t, matcher.Matches("valid/path/../../secret.txt"))
				assert.False(t, matcher.Matches("completely/different/path"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewBaseDirMatcher(tt.pattern)
			tt.assertions(t, matcher, err)
		})
	}
}

func TestExactMatcher(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		assertions func(*testing.T, Matcher, error)
	}{
		{
			name:    "basic pattern",
			pattern: "deployment",
			assertions: func(t *testing.T, matcher Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				// Verify matches
				assert.True(t, matcher.Matches("deployment"))

				// Verify non-matches
				assert.False(t, matcher.Matches("deployments"))
				assert.False(t, matcher.Matches("Deployment"))
				assert.False(t, matcher.Matches("deploy"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewExactMatcher(tt.pattern)
			tt.assertions(t, matcher, err)
		})
	}
}
