package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/kelseyhightower/envconfig"

	"github.com/akuity/kargo/pkg/governance"
	"github.com/akuity/kargo/pkg/logging"
)

type config struct {
	GitHubWebhookSecret string `envconfig:"GITHUB_WEBHOOK_SECRET" required:"true"`

	GitHubAppClientID          string `envconfig:"GITHUB_APP_CLIENT_ID" required:"true"`
	GitHubAppPrivateKeyEncoded string `envconfig:"GITHUB_APP_PRIVATE_KEY" required:"true"`
	GitHubAppPrivateKey        []byte `envconfig:"-"`

	Port string `envconfig:"PORT" default:"8080"`

	AWSLambdaRuntimeAPI string `envconfig:"AWS_LAMBDA_RUNTIME_API"`
}

func configFromEnv() (config, error) {
	cfg := config{}
	envconfig.MustProcess("", &cfg)
	var err error
	cfg.GitHubAppPrivateKey, err = base64.StdEncoding.DecodeString(cfg.GitHubAppPrivateKeyEncoded)
	if err != nil {
		return config{}, fmt.Errorf("error decoding GitHub app private key: %w", err)
	}
	return cfg, nil
}

func main() {
	cfg, err := configFromEnv()
	if err != nil {
		log.Fatalf("error loading config from environment: %s", err)
	}

	clientFactory, err := governance.NewGitHubClientFactory(
		cfg.GitHubAppClientID,
		cfg.GitHubAppPrivateKey,
	)
	if err != nil {
		log.Fatalf("error creating GitHub client factory: %s", err)
	}

	handler := governance.NewHandler(
		[]byte(cfg.GitHubWebhookSecret),
		clientFactory,
	)

	if cfg.AWSLambdaRuntimeAPI != "" {
		lambda.Start(httpadapter.NewV2(handler).ProxyWithContext)
	} else {
		logging.LoggerFromContext(context.Background()).
			Info(fmt.Sprintf("listening on :%s", cfg.Port))
		srv := &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("error starting server")
		}
	}
}
