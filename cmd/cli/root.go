package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/akuityio/k8sta/internal/common/os"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRootCommand() (*cobra.Command, error) {
	const desc = "Bookkeeper renders environment-specific configurations to " +
		"environment-specific branches of your gitops repos"
	command := &cobra.Command{
		Use:              "bookkeeper",
		Short:            desc,
		Long:             desc,
		PersistentPreRun: persistentPreRun,
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
		DisableAutoGenTag: true,
		SilenceUsage:      true,
	}
	renderCommand, err := newRenderCommand()
	if err != nil {
		return nil, err
	}
	command.AddCommand(renderCommand)
	updateImagesCommand, err := newUpdateImagesCommand()
	if err != nil {
		return nil, err
	}
	command.AddCommand(updateImagesCommand)
	command.AddCommand(newVersionCommand())
	return command, nil
}

func persistentPreRun(cmd *cobra.Command, _ []string) {
	cmd.Flags().VisitAll(
		func(flag *pflag.Flag) {
			switch flag.Name {
			case flagRepoPassword, flagRepoUsername, flagServer:
				if !flag.Changed {
					envVarName := fmt.Sprintf(
						"BOOKKEEPER_%s",
						strings.ReplaceAll(
							strings.ToUpper(flag.Name),
							"-",
							"_",
						),
					)
					envVarValue := os.GetEnvVar(envVarName, "")
					if envVarValue != "" {
						if err := cmd.Flags().Set(flag.Name, envVarValue); err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		},
	)
}
