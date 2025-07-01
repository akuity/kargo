package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/config"
)

// tokenRefresher is a component that helps to refresh tokens.
type tokenRefresher struct {
	// The following behaviors are overridable for testing purposes:

	redeemRefreshTokenFn func(
		ctx context.Context,
		serverAddress string,
		refreshToken string,
		insecureTLS bool,
	) (string, string, error)

	saveCLIConfigFn func(cfg config.CLIConfig) error
}

// newTokenRefresher returns a new tokenRefresher.
func newTokenRefresher() *tokenRefresher {
	return &tokenRefresher{
		redeemRefreshTokenFn: redeemRefreshToken,
		saveCLIConfigFn:      config.SaveCLIConfig,
	}
}

// refreshToken checks the token and refresh token in the provided config. If
// the token is not parsable as a JWT, it will assume the token is not something
// refreshable and return the provided config unmodified. If the token is
// parsable as a JWT and is not expired, it will return the provided config
// unmodified. If the token is expired and no refresh token is available OR TLS
// cert verification is disabled, an error is returned indicating that the user
// must re-authenticate. If a refresh token is available, it will attempt to
// redeem that token and return updated config.
func (t *tokenRefresher) refreshToken(
	ctx context.Context,
	cfg config.CLIConfig,
	insecureTLS bool,
) (config.CLIConfig, error) {
	jwtParser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var untrustedClaims jwt.RegisteredClaims
	if _, _, err :=
		jwtParser.ParseUnverified(cfg.BearerToken, &untrustedClaims); err != nil {
		// This token isn't a JWT. So it's probably a bearer token for the
		// Kubernetes API server. Just return. There's nothing further to do.
		return cfg, nil
	}

	// If we get to here, we're dealing with a JWT. It could have been issued:
	//
	//   1. Directly by the Kargo API server (in the case of admin)
	//   2. By Kargo's OpenID Connect identity provider
	//   3. By the Kubernetes cluster's identity provider
	//   4. By Kubernetes itself (a service account token, perhaps)

	if untrustedClaims.ExpiresAt == nil || time.Now().Before(untrustedClaims.ExpiresAt.Time) {
		// Token doesn't expire (possible for case 4) or hasn't yet. There's nothing
		// further to do.
		return cfg, nil
	}

	// If we get to here, the token is expired.

	if cfg.InsecureSkipTLSVerify || cfg.RefreshToken == "" {
		// We don't have a refresh token OR TLS cert verification is disabled. We'll
		// prompt the user to re-authenticate.
		return cfg, errors.New(
			"your token is expired; please use `kargo login` to re-authenticate",
		)
	}

	var err error
	if cfg.BearerToken, cfg.RefreshToken, err = t.redeemRefreshTokenFn(
		ctx,
		cfg.APIAddress,
		cfg.RefreshToken,
		insecureTLS,
	); err != nil {
		return cfg, errors.New(
			"error refreshing token; please use `kargo login` to re-authenticate",
		)
	}

	// Save and return the updated config
	return cfg, t.saveCLIConfigFn(cfg)
}

// redeemRefreshToken redeems the provided refresh token for a new ID token and
// refresh token.
func redeemRefreshToken(
	ctx context.Context,
	serverAddress string,
	refreshToken string,
	insecureTLS bool,
) (string, string, error) {
	client := GetClient(serverAddress, "", insecureTLS)

	res, err := client.GetPublicConfig(
		ctx,
		connect.NewRequest(&v1alpha1.GetPublicConfigRequest{}),
	)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving public configuration from server: %w", err)
	}

	if res.Msg.OidcConfig == nil {
		return "", "", errors.New("server does not support OpenID Connect")
	}

	provider, err := oidc.NewProvider(ctx, res.Msg.OidcConfig.IssuerUrl)
	if err != nil {
		return "", "", fmt.Errorf("error initializing OIDC provider: %w", err)
	}

	cfg := oauth2.Config{
		ClientID: res.Msg.OidcConfig.ClientId,
		Endpoint: provider.Endpoint(),
	}

	token, err := cfg.TokenSource(
		ctx,
		&oauth2.Token{
			RefreshToken: refreshToken,
		},
	).Token()
	if err != nil {
		return "", "", err
	}
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return "", "", errors.New("no id_token in token response")
	}
	return idToken, token.RefreshToken, nil
}
