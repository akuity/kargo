package main

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	versionpkg "github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newVersionCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			printToStdout := ptr.Deref(opt.PrintFlags.OutputFormat, "") == ""

			cliVersion := svcv1alpha1.ToVersionProto(versionpkg.GetVersion())
			if printToStdout {
				fmt.Println("Client Version:", cliVersion.GetVersion())
			}

			var serverVersion *svcv1alpha1.VersionInfo
			var serverErr error
			if !opt.UseLocalServer && !opt.ClientVersionOnly {
				serverVersion, serverErr = getServerVersion(ctx, cfg, opt)
			}

			if printToStdout {
				if serverVersion != nil {
					fmt.Println("Server Version:", serverVersion.GetVersion())
				}
				return serverErr
			}

			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			obj, err := componentVersionsToRuntimeObject(&svcv1alpha1.ComponentVersions{
				Server: serverVersion,
				Cli:    cliVersion,
			})
			if err != nil {
				return errors.Wrap(err, "map component versions to runtime object")
			}

			if err := printer.PrintObj(obj, opt.IOStreams.Out); err != nil {
				return errors.Wrap(err, "printing object")
			}
			return serverErr
		},
	}

	opt.PrintFlags.AddFlags(cmd)
	option.InsecureTLS(cmd.PersistentFlags(), opt)
	option.ClientVersion(cmd.PersistentFlags(), opt)
	return cmd
}

func getServerVersion(ctx context.Context, cfg config.CLIConfig, opt *option.Option) (*svcv1alpha1.VersionInfo, error) {
	if cfg.APIAddress == "" || cfg.BearerToken == "" {
		return nil, nil
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
	if err != nil {
		return nil, errors.Wrap(err, "get client from config")
	}
	resp, err := kargoSvcCli.GetVersionInfo(
		ctx,
		connect.NewRequest(&svcv1alpha1.GetVersionInfoRequest{}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "get version info from server")
	}

	return resp.Msg.GetVersionInfo(), nil
}

func componentVersionsToRuntimeObject(v *svcv1alpha1.ComponentVersions) (runtime.Object, error) {
	data, err := protojson.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "marshal component versions")
	}
	var content map[string]any
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, errors.Wrap(err, "unmarshal component versions")
	}
	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(content)
	u.SetAPIVersion(kargoapi.GroupVersion.String())
	u.SetKind("ComponentVersions")
	return u, nil
}
