package freight

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqualJSON(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		a, b  string
		equal bool
	}{
		{name: "absent metadata", equal: true},
		{name: "absent differs from empty object", b: `{}`},
		{name: "format and key order", a: `{"a":1,"b":"x"}`, b: `{ "b": "x", "a": 1 }`, equal: true},
		{name: "string escapes", a: `{"s":"\u0061"}`, b: `{"s":"a"}`, equal: true},
		{name: "large integer is exact", a: `{"n":9007199254740993}`, b: `{"n":9007199254740992}`},
		{name: "exponent notation", a: `{"n":1.2300e3}`, b: `{"n":1230}`, equal: true},
		{name: "negative exponent", a: `{"n":1e-3}`, b: `{"n":0.0010}`, equal: true},
		{name: "negative number", a: `{"n":-100.0}`, b: `{"n":-1e+2}`, equal: true},
		{name: "signed zero", a: `{"n":-0.00e200}`, b: `{"n":0}`, equal: true},
		{name: "large exponent", a: `{"n":1e9999999999}`, b: `{"n":10e9999999998}`, equal: true},
		{name: "nested values", a: `{"a":[{"n":1e2},null,true]}`, b: `{"a":[{"n":100},null,true]}`, equal: true},
		{name: "array order matters", a: `{"a":[1,2]}`, b: `{"a":[2,1]}`},
		{name: "number is not a string", a: `{"n":1}`, b: `{"n":"1"}`},
		{name: "invalid left", a: `invalid`, b: `{}`},
		{name: "invalid right", a: `{}`, b: `invalid`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.equal, equalJSON([]byte(testCase.a), []byte(testCase.b)))
		})
	}
}
