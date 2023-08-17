package main

import (
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	versionpkg "github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newVersionCommand(opt *option.Option) *cobra.Command {
	return &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var serverVersion *svcv1alpha1.VersionInfo
			if !opt.UseLocalServer {
				kargoCli, err := client.GetClientFromConfig(ctx, opt)
				if err != nil {
					if !client.IsConfigNotFoundErr(err) {
						return errors.Wrap(err, "get Kargo client from config")
					}
					// Skip initializing server version if config not found (not logged in).
				} else {
					resp, err := kargoCli.GetVersionInfo(ctx, connect.NewRequest(&svcv1alpha1.GetVersionInfoRequest{}))
					if err != nil {
						return errors.Wrap(err, "get version info from server")
					}
					serverVersion = resp.Msg.GetVersionInfo()
				}
			}

			data, err := protojson.Marshal(&svcv1alpha1.ComponentVersions{
				Server: serverVersion,
				Cli:    typesv1alpha1.ToVersionProto(versionpkg.GetVersion()),
			})
			if err != nil {
				return errors.Wrap(err, "get version info from server")
			}
			fmt.Println(string(data))
			return nil
		},
	}
}
