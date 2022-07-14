package scratch

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestK8sTAConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(Config, error)
	}{
		{
			name: "CONFIG_PATH not set",
			assertions: func(_ Config, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "CONFIG_PATH")
			},
		},
		{
			name: "CONFIG_PATH path does not exist",
			setup: func() {
				t.Setenv("CONFIG_PATH", "/completely/bogus/path")
			},
			assertions: func(_ Config, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"file /completely/bogus/path does not exist",
				)
			},
		},
		{
			name: "CONFIG_PATH does not contain valid json",
			setup: func() {
				configFile, err := ioutil.TempFile("", "config.json")
				require.NoError(t, err)
				defer configFile.Close()
				_, err = configFile.Write([]byte("this is not json"))
				require.NoError(t, err)
				t.Setenv("CONFIG_PATH", configFile.Name())
			},
			assertions: func(_ Config, err error) {
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "invalid character",
				)
			},
		},
		{
			name: "success",
			setup: func() {
				configFile, err := ioutil.TempFile("", "config.json")
				require.NoError(t, err)
				defer configFile.Close()
				_, err =
					configFile.Write(
						[]byte(
							`[{"name":"guestbook","imageRepository":"akuityio/guestbook","namespace":"argocd","environments":["guestbook-dev","guestbook-stage","guestbook-prod"]}]`, // nolint: lll
						),
					)
				require.NoError(t, err)
				t.Setenv("CONFIG_PATH", configFile.Name())
			},
			assertions: func(config Config, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, config.LineCount())
				line, ok := config.GetLineByName("guestbook")
				require.True(t, ok)
				require.Equal(t, "guestbook", line.Name)
				require.Equal(t, "akuityio/guestbook", line.ImageRepository)
				require.Equal(t, "argocd", line.Namespace)
				require.Equal(
					t,
					[]string{"guestbook-dev", "guestbook-stage", "guestbook-prod"},
					line.Environments,
				)
				line, ok = config.GetLineByImageRepository("akuityio/guestbook")
				require.True(t, ok)
				require.Equal(t, "guestbook", line.Name)
				require.Equal(t, "akuityio/guestbook", line.ImageRepository)
				require.Equal(t, "argocd", line.Namespace)
				require.Equal(
					t,
					[]string{"guestbook-dev", "guestbook-stage", "guestbook-prod"},
					line.Environments,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			config, err := K8staConfig()
			testCase.assertions(config, err)
		})
	}
}
