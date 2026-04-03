package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdditionalParameters_Decode(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  string
		assert func(*testing.T, AdditionalParameters, error)
	}{
		{
			name:  "empty string yields empty map",
			input: "",
			assert: func(t *testing.T, p AdditionalParameters, err error) {
				require.NoError(t, err)
				require.Empty(t, p)
			},
		},
		{
			name:  "single key=value",
			input: "audience=https://kubernetes.default.svc",
			assert: func(t *testing.T, p AdditionalParameters, err error) {
				require.NoError(t, err)
				require.Equal(t, AdditionalParameters{
					"audience": "https://kubernetes.default.svc",
				}, p)
			},
		},
		{
			name:  "multiple key=value pairs",
			input: "audience=https://kubernetes.default.svc,domain_hint=corp.example.com",
			assert: func(t *testing.T, p AdditionalParameters, err error) {
				require.NoError(t, err)
				require.Equal(t, AdditionalParameters{
					"audience":    "https://kubernetes.default.svc",
					"domain_hint": "corp.example.com",
				}, p)
			},
		},
		{
			name:  "value containing equals sign",
			input: "redirect=https://example.com/callback?foo=bar",
			assert: func(t *testing.T, p AdditionalParameters, err error) {
				require.NoError(t, err)
				require.Equal(t, AdditionalParameters{
					"redirect": "https://example.com/callback?foo=bar",
				}, p)
			},
		},
		{
			name:  "missing equals sign returns error",
			input: "badparam",
			assert: func(t *testing.T, _ AdditionalParameters, err error) {
				require.ErrorContains(t, err, "badparam")
				require.ErrorContains(t, err, "key=value")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var p AdditionalParameters
			err := p.Decode(tc.input)
			tc.assert(t, p, err)
		})
	}
}
