package subscription

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestSchema_Validate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		subType string
		cfg     string
		assert  func(*testing.T, field.ErrorList)
	}{
		{
			name:    "malformed config",
			subType: "chart",
			cfg:     `{`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeInternal, errs[0].Type)
				require.Equal(t, "spec.subscriptions[0].chart", errs[0].Field)
			},
		},
		{
			name:    "valid config",
			subType: "chart",
			cfg:     `{"repoURL":"oci://example.com/chart","semverConstraint":"<1.1.0","discoveryLimit":20}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name:    "unknown field",
			subType: "chart",
			cfg:     `{"repoURL":"oci://example.com/chart","constraint":"<1.1.0"}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeForbidden, errs[0].Type)
				require.Equal(t, "spec.subscriptions[0].chart.constraint", errs[0].Field)
				require.Equal(t, "unknown field", errs[0].Detail)
			},
		},
		{
			name:    "missing required field",
			subType: "chart",
			cfg:     `{}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeRequired, errs[0].Type)
				require.Equal(t, "spec.subscriptions[0].chart.repoURL", errs[0].Field)
			},
		},
		{
			name:    "invalid value",
			subType: "chart",
			cfg:     `{"repoURL":"oci://example.com/chart","discoveryLimit":101}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Equal(t, "spec.subscriptions[0].chart.discoveryLimit", errs[0].Field)
				require.EqualValues(t, 101, errs[0].BadValue)
			},
		},
		{
			name:    "unsupported value",
			subType: "image",
			cfg:     `{"repoURL":"example.com/image","imageSelectionStrategy":"Newest"}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeNotSupported, errs[0].Type)
				require.Equal(
					t,
					"spec.subscriptions[0].image.imageSelectionStrategy",
					errs[0].Field,
				)
				require.Equal(t, "Newest", errs[0].BadValue)
				require.Contains(t, errs[0].Detail, `"Digest"`)
			},
		},
		{
			name:    "invalid array element",
			subType: "git",
			cfg:     `{"repoURL":"https://example.com/repo","includePaths":["foo",1]}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Equal(t, "spec.subscriptions[0].git.includePaths[1]", errs[0].Field)
			},
		},
		{
			name:    "several errors",
			subType: "git",
			cfg:     `{"repoURL":"bogus","allowTags":"^v1"}`,
			assert: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 2)
				fields := []string{errs[0].Field, errs[1].Field}
				require.ElementsMatch(
					t,
					[]string{
						"spec.subscriptions[0].git.allowTags",
						"spec.subscriptions[0].git.repoURL",
					},
					fields,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			schema := mustLoadSchema(testCase.subType)
			testCase.assert(
				t,
				schema.Validate(
					field.NewPath("spec", "subscriptions").Index(0).Child(testCase.subType),
					[]byte(testCase.cfg),
				),
			)
		})
	}
}

func TestSchemas(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		subType string
		cfg     string
		// errFields are the paths, relative to the configuration, of the
		// expected errors
		errFields []string
	}{
		{
			name:    "valid OCI chart",
			subType: "chart",
			cfg:     `{"repoURL":"oci://ghcr.io/example/chart","semverConstraint":"^1.0.0","discoveryLimit":20}`,
		},
		{
			name:    "valid HTTPS chart",
			subType: "chart",
			cfg:     `{"repoURL":"https://charts.example.com","name":"mychart","discoveryLimit":20}`,
		},
		{
			name:      "chart repoURL empty",
			subType:   "chart",
			cfg:       `{"repoURL":""}`,
			errFields: []string{"repoURL", "repoURL"},
		},
		{
			name:      "chart repoURL invalid",
			subType:   "chart",
			cfg:       `{"repoURL":"bogus-url"}`,
			errFields: []string{"repoURL"},
		},
		{
			name:      "chart repoURL with empty port",
			subType:   "chart",
			cfg:       `{"repoURL":"oci://ghcr.io:/example/chart"}`,
			errFields: []string{"repoURL"},
		},
		{
			name:      "OCI chart with name",
			subType:   "chart",
			cfg:       `{"repoURL":"oci://ghcr.io/example/chart","name":"should-not-be-here"}`,
			errFields: []string{"name"},
		},
		{
			name:      "HTTPS chart without name",
			subType:   "chart",
			cfg:       `{"repoURL":"https://charts.example.com"}`,
			errFields: []string{"name"},
		},
		{
			name:      "HTTP chart with empty name",
			subType:   "chart",
			cfg:       `{"repoURL":"http://charts.example.com","name":""}`,
			errFields: []string{"name"},
		},
		{
			name:      "chart discoveryLimit too small",
			subType:   "chart",
			cfg:       `{"repoURL":"oci://ghcr.io/example/chart","discoveryLimit":0}`,
			errFields: []string{"discoveryLimit"},
		},
		{
			name:      "chart discoveryLimit too large",
			subType:   "chart",
			cfg:       `{"repoURL":"oci://ghcr.io/example/chart","discoveryLimit":101}`,
			errFields: []string{"discoveryLimit"},
		},
		{
			name:    "valid image",
			subType: "image",
			cfg: `{"repoURL":"ghcr.io/akuity/kargo","imageSelectionStrategy":"SemVer",` +
				`"constraint":"^1.0.0","platform":"linux/amd64","cacheByTag":true,"discoveryLimit":20}`,
		},
		{
			name:    "image platform with variant",
			subType: "image",
			cfg:     `{"repoURL":"ghcr.io/akuity/kargo","platform":"linux/arm/v7"}`,
		},
		{
			name:      "image repoURL empty",
			subType:   "image",
			cfg:       `{"repoURL":""}`,
			errFields: []string{"repoURL", "repoURL"},
		},
		{
			name:      "image repoURL invalid",
			subType:   "image",
			cfg:       `{"repoURL":"bogus invalid"}`,
			errFields: []string{"repoURL"},
		},
		{
			name:      "image selection strategy invalid",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","imageSelectionStrategy":"InvalidStrategy"}`,
			errFields: []string{"imageSelectionStrategy"},
		},
		{
			name:      "Digest strategy without constraint",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","imageSelectionStrategy":"Digest"}`,
			errFields: []string{"constraint"},
		},
		{
			name:      "Digest strategy with empty constraint",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","imageSelectionStrategy":"Digest","constraint":""}`,
			errFields: []string{"constraint"},
		},
		{
			name:      "image platform invalid",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","platform":"bogus"}`,
			errFields: []string{"platform"},
		},
		{
			name:      "image discoveryLimit too small",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","discoveryLimit":0}`,
			errFields: []string{"discoveryLimit"},
		},
		{
			name:      "image discoveryLimit too large",
			subType:   "image",
			cfg:       `{"repoURL":"ghcr.io/akuity/kargo","discoveryLimit":101}`,
			errFields: []string{"discoveryLimit"},
		},
		{
			name:    "valid git",
			subType: "git",
			cfg: `{"repoURL":"https://github.com/akuity/kargo.git","branch":"main",` +
				`"commitSelectionStrategy":"NewestFromBranch","semverConstraint":"^1.0.0","discoveryLimit":20}`,
		},
		{
			name:      "git repoURL empty",
			subType:   "git",
			cfg:       `{"repoURL":""}`,
			errFields: []string{"repoURL", "repoURL"},
		},
		{
			name:      "git repoURL invalid",
			subType:   "git",
			cfg:       `{"repoURL":"bogus"}`,
			errFields: []string{"repoURL"},
		},
		{
			name:      "git repoURL is an SSH URL",
			subType:   "git",
			cfg:       `{"repoURL":"git@github.com:akuity/kargo.git"}`,
			errFields: []string{"repoURL"},
		},
		{
			name:      "git branch too long",
			subType:   "git",
			cfg:       `{"repoURL":"https://github.com/akuity/kargo.git","branch":"a` + strings.Repeat("b", 255) + `"}`,
			errFields: []string{"branch"},
		},
		{
			name:      "git branch invalid",
			subType:   "git",
			cfg:       `{"repoURL":"https://github.com/akuity/kargo.git","branch":"-invalid"}`,
			errFields: []string{"branch"},
		},
		{
			name:      "git commit selection strategy invalid",
			subType:   "git",
			cfg:       `{"repoURL":"https://github.com/akuity/kargo.git","commitSelectionStrategy":"InvalidStrategy"}`,
			errFields: []string{"commitSelectionStrategy"},
		},
		{
			name:      "git discoveryLimit too small",
			subType:   "git",
			cfg:       `{"repoURL":"https://github.com/akuity/kargo.git","discoveryLimit":0}`,
			errFields: []string{"discoveryLimit"},
		},
		{
			name:      "git discoveryLimit too large",
			subType:   "git",
			cfg:       `{"repoURL":"https://github.com/akuity/kargo.git","discoveryLimit":101}`,
			errFields: []string{"discoveryLimit"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			errs := mustLoadSchema(testCase.subType).Validate(nil, []byte(testCase.cfg))
			fields := make([]string, len(errs))
			for i, err := range errs {
				fields[i] = err.Field
			}
			if len(testCase.errFields) == 0 {
				require.Empty(t, fields)
				return
			}
			require.ElementsMatch(t, testCase.errFields, fields)
		})
	}
}

// schemaAccepts reports whether the schema for subscriptions of type subType
// accepts value as the value of the named field. Errors concerning other
// fields are disregarded.
func schemaAccepts(t *testing.T, subType, name, value string) bool {
	t.Helper()
	cfg, err := json.Marshal(map[string]string{name: value})
	require.NoError(t, err)
	for _, fieldErr := range mustLoadSchema(subType).Validate(nil, cfg) {
		if fieldErr.Field == name {
			return false
		}
	}
	return true
}
