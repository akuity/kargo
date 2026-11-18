package completion

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
)

func TestProjectCompletion(t *testing.T) {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1beta1/projects" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"kargo-demo"}},{"metadata":{"name":"other"}}]}`))
		},
	)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	tlsSrv := httptest.NewTLSServer(handler)
	t.Cleanup(tlsSrv.Close)

	loggedIn := config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"}
	// The server answers 500 for any path under /broken.
	brokenServer := config.CLIConfig{APIAddress: srv.URL + "/broken", BearerToken: "test-token"}
	// The TLS server's certificate is not trusted.
	untrusted := config.CLIConfig{APIAddress: tlsSrv.URL, BearerToken: "test-token"}

	testCases := []struct {
		name string
		cfg  config.CLIConfig
		args []string
		want string
	}{
		{
			name: "argument",
			cfg:  loggedIn,
			args: []string{"projects", ""},
			want: "kargo-demo\nother\n:4\n",
		},
		{
			name: "project flag",
			cfg:  loggedIn,
			args: []string{"stages", "--project", ""},
			want: "kargo-demo\nother\n:4\n",
		},
		{
			name: "not logged in",
			cfg:  config.CLIConfig{},
			args: []string{"projects", ""},
			want: ":4\n",
		},
		{
			name: "server error",
			cfg:  brokenServer,
			args: []string{"stages", "--project", ""},
			want: ":4\n",
		},
		{
			name: "untrusted certificate",
			cfg:  untrusted,
			args: []string{"projects", ""},
			want: ":4\n",
		},
		{
			name: "insecure-skip-tls-verify",
			cfg:  untrusted,
			args: []string{"stages", "--insecure-skip-tls-verify", "--project", ""},
			want: "kargo-demo\nother\n:4\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRoot(tc.cfg)
			out := &bytes.Buffer{}
			root.SetOut(out)
			root.SetArgs(append([]string{cobra.ShellCompRequestCmd}, tc.args...))
			require.NoError(t, root.Execute())
			require.Equal(t, tc.want, out.String())
		})
	}
}

func TestProjectCompletionTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		},
	))
	t.Cleanup(srv.Close)

	defer func(d time.Duration) { timeout = d }(timeout)
	timeout = 200 * time.Millisecond

	root := newRoot(config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"})
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{cobra.ShellCompRequestCmd, "projects", ""})
	start := time.Now()
	require.NoError(t, root.Execute())
	require.Less(t, time.Since(start), 5*time.Second)
	require.Equal(t, ":4\n", out.String())
}

func newRoot(cfg config.CLIConfig) *cobra.Command {
	root := &cobra.Command{Use: "kargo"}
	(&client.Options{}).AddFlags(root.PersistentFlags())
	root.AddCommand(&cobra.Command{
		Use:               "projects",
		ValidArgsFunction: ProjectNames(cfg),
	})
	stages := &cobra.Command{Use: "stages"}
	var project string
	option.Project(stages.Flags(), &project, "", "")
	root.AddCommand(stages)
	RegisterProjectFlag(root, cfg)
	root.SetErr(io.Discard)
	return root
}

func TestStageAndWarehouseCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var body string
			switch r.URL.Path {
			case "/v1beta1/projects/kargo-demo/stages":
				body = `{"items":[{"metadata":{"name":"test"},"spec":{}},{"metadata":{"name":"uat"},"spec":{}}]}`
			case "/v1beta1/projects/kargo-demo/warehouses":
				body = `{"items":[{"metadata":{"name":"main"},"spec":{}}]}`
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		},
	))
	t.Cleanup(srv.Close)

	testCases := []struct {
		name           string
		defaultProject string
		args           []string
		want           string
	}{
		{
			name: "stages in the project from the flag",
			args: []string{"stages", "--project", "kargo-demo", ""},
			want: "test\nuat\n:4\n",
		},
		{
			name: "warehouses in the project from the flag",
			args: []string{"warehouses", "--project=kargo-demo", ""},
			want: "main\n:4\n",
		},
		{
			name:           "stages in the default project",
			defaultProject: "kargo-demo",
			args:           []string{"stages", ""},
			want:           "test\nuat\n:4\n",
		},
		{
			name:           "flag overrides the default project",
			defaultProject: "kargo-demo",
			args:           []string{"stages", "--project", "other", ""},
			want:           ":4\n",
		},
		{
			name: "no project",
			args: []string{"warehouses", ""},
			want: ":4\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.CLIConfig{
				APIAddress:  srv.URL,
				BearerToken: "test-token",
				Project:     tc.defaultProject,
			}
			root := &cobra.Command{Use: "kargo"}
			for use, complete := range map[string]cobra.CompletionFunc{
				"stages":     StageNames(cfg),
				"warehouses": WarehouseNames(cfg),
			} {
				cmd := &cobra.Command{Use: use, ValidArgsFunction: complete}
				var project string
				option.Project(cmd.Flags(), &project, cfg.Project, "")
				root.AddCommand(cmd)
			}
			out := &bytes.Buffer{}
			root.SetOut(out)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{cobra.ShellCompRequestCmd}, tc.args...))
			require.NoError(t, root.Execute())
			require.Equal(t, tc.want, out.String())
		})
	}
}
