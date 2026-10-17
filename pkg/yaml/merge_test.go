package yaml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	testCases := []struct {
		name       string
		docs       []string
		assertions func(t *testing.T, out []byte, err error)
	}{
		{
			name: "no documents",
			assertions: func(t *testing.T, out []byte, err error) {
				require.NoError(t, err)
				require.Empty(t, out)
			},
		},
		{
			name: "only empty documents",
			docs: []string{"", "\n"},
			assertions: func(t *testing.T, out []byte, err error) {
				require.NoError(t, err)
				require.Empty(t, out)
			},
		},
		{
			name: "document is not valid YAML",
			docs: []string{":"},
			assertions: func(t *testing.T, _ []byte, err error) {
				require.ErrorContains(t, err, "error parsing document 0")
			},
		},
		{
			name: "successful merge",
			// Note: This is not an exhaustive test of all merge behavior. We presume
			// kyaml to be well-tested. We DO, however, make a point of verifying that
			// list merge behavior is consistent with Helm (full replacement).
			docs: []string{
				"", // Implementation should tolerate this
				`
character:
  name: Vader
  ability: Force
  affiliation: Empire
  saberColor: red
vehicles:
- Star Destroyer`,
				`
character:
  name: Luke # Alters character.name from the base
  # ability: Force  # Should be inherited from the base without alteration
  affiliation: Rebel Alliance # Alters character.affiliation from the base
  saberColor: blue # Alters character.saberColor from the base
  siblings: # A completely new field that the base does not have
  - Leia
vehicles: # Completely replaces vehicles list from the base
- Millennium Falcon`,
			},
			assertions: func(t *testing.T, out []byte, err error) {
				require.NoError(t, err)
				require.YAMLEq(
					t,
					`
character:
  name: Luke
  ability: Force
  affiliation: Rebel Alliance
  saberColor: blue
  siblings:
  - Leia
vehicles:
- Millennium Falcon`,
					string(out),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			docs := make([][]byte, len(testCase.docs))
			for i, doc := range testCase.docs {
				docs[i] = []byte(doc)
			}
			out, err := Merge(docs)
			testCase.assertions(t, out, err)
		})
	}
}
