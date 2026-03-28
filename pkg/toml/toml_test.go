package toml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetValuesInBytes(t *testing.T) {
	tests := []struct {
		name       string
		inBytes    []byte
		updates    []Update
		assertions func(*testing.T, []byte, error)
	}{
		// Error cases
		{
			name:    "invalid TOML",
			inBytes: []byte("key = \n"),
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "error parsing input")
				require.Nil(t, bytes)
			},
		},
		{
			name:    "missing key",
			inBytes: []byte("title = \"old\"\n"),
			updates: []Update{{Key: "missing", Value: "new"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "key path not found")
				require.Nil(t, bytes)
			},
		},
		{
			name:    "non scalar target",
			inBytes: []byte("[service]\nname = \"api\"\n"),
			updates: []Update{{Key: "service", Value: "other"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "addresses Table instead of a scalar node")
				require.Nil(t, bytes)
			},
		},
		{
			name:    "unsupported value type",
			inBytes: []byte("title = \"old\"\n"),
			updates: []Update{{Key: "title", Value: []string{"a", "b"}}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "value is not a TOML scalar type")
				require.Nil(t, bytes)
			},
		},
		// Single top-level updates
		{
			name:    "updates scalar value",
			inBytes: []byte("title = \"old\"\nactive = true\n"),
			updates: []Update{{Key: "title", Value: "new"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("title = 'new'\nactive = true\n"), bytes)
			},
		},
		{
			name:    "updates escaped dot key",
			inBytes: []byte("\"example.com/version\" = \"1.0.0\"\n"),
			updates: []Update{{Key: `example\.com/version`, Value: "2.0.0"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("\"example.com/version\" = '2.0.0'\n"), bytes)
			},
		},
		// Single updates under tables
		{
			name:    "updates scalar under table header",
			inBytes: []byte("[package]\nversion = \"1.0.0\"\n"),
			updates: []Update{{Key: "package.version", Value: "2.0.0"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("[package]\nversion = '2.0.0'\n"), bytes)
			},
		},
		{
			name:    "updates scalar under nested table header",
			inBytes: []byte("[a.b]\nc = 1\n"),
			updates: []Update{{Key: "a.b.c", Value: 99}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("[a.b]\nc = 99\n"), bytes)
			},
		},
		{
			name:    "updates scalar under table header that follows other expressions",
			inBytes: []byte("threshold = 1\n\n[features]\nenabled = false\n"),
			updates: []Update{{Key: "features.enabled", Value: true}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("threshold = 1\n\n[features]\nenabled = true\n"), bytes)
			},
		},
		// Single updates in inline/array constructs
		{
			name:    "updates array item",
			inBytes: []byte("values = [1, 2, 3]\n"),
			updates: []Update{{Key: "values.1", Value: 42}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("values = [1, 42, 3]\n"), bytes)
			},
		},
		{
			name:    "updates inline table item",
			inBytes: []byte("service = { name = \"api\", port = 8080 }\n"),
			updates: []Update{{Key: "service.port", Value: 9090}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("service = { name = \"api\", port = 9090 }\n"), bytes)
			},
		},
		{
			name:    "updates array table item",
			inBytes: []byte("[[services]]\nname = \"api\"\nport = 8080\n\n[[services]]\nname = \"web\"\nport = 8081\n"),
			updates: []Update{{Key: "services.1.port", Value: 9090}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte("[[services]]\nname = \"api\"\nport = 8080\n\n[[services]]\nname = \"web\"\nport = 9090\n"),
					bytes,
				)
			},
		},
		// Formatting preservation
		{
			name:    "preserves inline comments around updated value",
			inBytes: []byte("title = \"old\" # keep me\nactive = true\n"),
			updates: []Update{{Key: "title", Value: "new"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("title = 'new' # keep me\nactive = true\n"), bytes)
			},
		},
		{
			name:    "preserves standalone comment lines",
			inBytes: []byte("# header comment\nv = 1\n# footer comment\n"),
			updates: []Update{{Key: "v", Value: 2}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("# header comment\nv = 2\n# footer comment\n"), bytes)
			},
		},
		{
			name:    "preserves blank lines between sections",
			inBytes: []byte("[a]\nx = 1\n\n\n[b]\ny = 2\n"),
			updates: []Update{{Key: "b.y", Value: 99}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("[a]\nx = 1\n\n\n[b]\ny = 99\n"), bytes)
			},
		},
		{
			name:    "value grows in length",
			inBytes: []byte("v = 1\nother = true\n"),
			updates: []Update{{Key: "v", Value: 99999}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("v = 99999\nother = true\n"), bytes)
			},
		},
		{
			name:    "value shrinks in length",
			inBytes: []byte("v = 99999\nother = true\n"),
			updates: []Update{{Key: "v", Value: 1}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("v = 1\nother = true\n"), bytes)
			},
		},
		// Multiple updates
		{
			name:    "multiple updates to different top-level keys",
			inBytes: []byte("a = 1\nb = 2\nc = 3\n"),
			updates: []Update{
				{Key: "a", Value: 10},
				{Key: "c", Value: 30},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("a = 10\nb = 2\nc = 30\n"), bytes)
			},
		},
		{
			name:    "multiple updates across top-level and table-scoped keys",
			inBytes: []byte("threshold = 1\n\n[package]\nversion = \"1.0.0\"\n\n[features]\nenabled = false\n"),
			updates: []Update{
				{Key: "threshold", Value: 100},
				{Key: "package.version", Value: "2.0.0"},
				{Key: "features.enabled", Value: true},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte("threshold = 100\n\n[package]\nversion = '2.0.0'\n\n[features]\nenabled = true\n"),
					bytes,
				)
			},
		},
		{
			name:    "multiple updates within same table",
			inBytes: []byte("[server]\nhost = \"localhost\"\nport = 8080\n"),
			updates: []Update{
				{Key: "server.host", Value: "0.0.0.0"},
				{Key: "server.port", Value: 9090},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte("[server]\nhost = '0.0.0.0'\nport = 9090\n"),
					bytes,
				)
			},
		},
		{
			name:    "duplicate key in updates uses last value",
			inBytes: []byte("v = 1\n"),
			updates: []Update{
				{Key: "v", Value: 2},
				{Key: "v", Value: 3},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("v = 3\n"), bytes)
			},
		},
		// Edge cases
		{
			name:    "no updates returns input unchanged",
			inBytes: []byte("title = \"hello\"\n"),
			updates: []Update{},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("title = \"hello\"\n"), bytes)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outBytes, err := SetValuesInBytes(test.inBytes, test.updates)
			test.assertions(t, outBytes, err)
		})
	}
}

func TestFormatValueString(t *testing.T) {
	require.Equal(t, `'value'`, FormatValueString("value"))
	require.Equal(t, "42", FormatValueString(42))
	require.Equal(t, "true", FormatValueString(true))
}
