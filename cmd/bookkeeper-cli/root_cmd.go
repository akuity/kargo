package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var desc = "Bookkeeper renders environment-specific configurations to " +
	"environment-specific branches of your gitops repos"

func newRootCommand() (*cobra.Command, error) {
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
					envVarValue := os.Getenv(envVarName)
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
