package stage

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Manage stages",
	}
	option.InsecureTLS(cmd.PersistentFlags(), opt)
	option.LocalServer(cmd.PersistentFlags(), opt)

	cmd.AddCommand(newPromoteCommand(cfg, opt))
	cmd.AddCommand(newEnableAutoPromotion(cfg, opt))
	cmd.AddCommand(newDisableAutoPromotion(cfg, opt))
	cmd.AddCommand(newPromoteSubscribersCommand(cfg, opt))
	return cmd
}
