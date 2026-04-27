package main

import (
	"context"
	"fmt"
	"os"

	"github.com/akuity/kargo/internal/kargomcp"
	"github.com/akuity/kargo/pkg/cli/config"
)

func main() {
	cfg := loadConfig()
	server := kargomcp.New(cfg)
	fmt.Fprintf(os.Stderr, "kargo-mcp starting (address: %s)\n", cfg.APIAddress)
	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "kargo-mcp error: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig merges environment variables (priority) and the CLI config file.
func loadConfig() config.CLIConfig {
	env := config.NewEnvVarCLIConfig()
	file, err := config.LoadCLIConfig()
	if err == nil {
		if env.APIAddress == "" {
			env.APIAddress = file.APIAddress
		}
		if env.BearerToken == "" {
			env.BearerToken = file.BearerToken
			env.RefreshToken = file.RefreshToken
			env.InsecureSkipTLSVerify = file.InsecureSkipTLSVerify
		}
	}
	return env
}
