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
		{
			name:    "invalid TOML",
			inBytes: []byte("key = \n"),
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "error parsing input")
				require.Nil(t, bytes)
			},
		},
		{
			name:    "updates scalar value",
			inBytes: []byte("title = \"old\"\nactive = true\n"),
			updates: []Update{{Key: "title", Value: "new"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("title = \"new\"\nactive = true\n"), bytes)
			},
		},
		{
			name:    "preserves inline comments around updated value",
			inBytes: []byte("title = \"old\" # keep me\nactive = true\n"),
			updates: []Update{{Key: "title", Value: "new"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("title = \"new\" # keep me\nactive = true\n"), bytes)
			},
		},
		{
			name:    "updates escaped dot key",
			inBytes: []byte("\"example.com/version\" = \"1.0.0\"\n"),
			updates: []Update{{Key: `example\.com/version`, Value: "2.0.0"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("\"example.com/version\" = \"2.0.0\"\n"), bytes)
			},
		},
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
		{
			name:    "non scalar target",
			inBytes: []byte("[service]\nname = \"api\"\n"),
			updates: []Update{{Key: "service", Value: "other"}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "does not address a scalar node")
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
			name:    "unsupported value type",
			inBytes: []byte("title = \"old\"\n"),
			updates: []Update{{Key: "title", Value: []string{"a", "b"}}},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "value is not a TOML scalar type")
				require.Nil(t, bytes)
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
	require.Equal(t, `"value"`, FormatValueString("value"))
	require.Equal(t, "42", FormatValueString(42))
	require.Equal(t, "true", FormatValueString(true))
}
