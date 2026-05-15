package delete

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_deleteFreightOptions_validate(t *testing.T) {
	testCases := []struct {
		name       string
		opts       deleteFreightOptions
		assertions func(*testing.T, error)
	}{
		{
			name: "valid with name",
			opts: deleteFreightOptions{
				Project: "my-project",
				Names:   []string{"abc123"},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "valid with alias",
			opts: deleteFreightOptions{
				Project: "my-project",
				Aliases: []string{"my-alias"},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "valid with multiple names",
			opts: deleteFreightOptions{
				Project: "my-project",
				Names:   []string{"abc123", "def456"},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "missing project",
			opts: deleteFreightOptions{
				Names: []string{"abc123"},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "project is required")
			},
		},
		{
			name: "missing name and alias",
			opts: deleteFreightOptions{
				Project: "my-project",
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "name or alias is required")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.validate()
			tc.assertions(t, err)
		})
	}
}
