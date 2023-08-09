package client

import (
	"context"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/akuity/kargo/internal/cli/config"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
// unmodified. If the token is expired and no refresh token is available, an
// error is returned indicating that the user must re-authenticate. If a refresh
// token is available, it will attempt to redeem that token and return updated
// config.
func (t *tokenRefresher) refreshToken(
	ctx context.Context,
	cfg config.CLIConfig,
	insecureTLS bool,
) (config.CLIConfig, error) {
	jwtParser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims jwt.RegisteredClaims
	if _, _, err :=
		jwtParser.ParseUnverified(cfg.BearerToken, &claims); err != nil {
		// This token isn't a JWT. So it's probably a bearer token for the
		// Kubernetes API server. Just return. There's nothing further to do.
		return cfg, nil
	}

	// If we get to here, we're dealing with an ID token (JWT) issued either by
	// an OpenID Connect identity provider or by the Kargo API server itself.

	if time.Now().Before(claims.ExpiresAt.Time) {
		// Token is still valid. There's nothing further to do.
		return cfg, nil
	}

	// If we get to here, the token is expired.

	if cfg.RefreshToken == "" {
		// We don't have a refresh token. So all we can do is prompt the user to
		// re-authenticate.
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
		return "", "", errors.Wrap(
			err,
			"error retrieving public configuration from server",
		)
	}

	if res.Msg.OidcConfig == nil {
		return "", "", errors.New("server does not support OpenID Connect")
	}

	provider, err := oidc.NewProvider(ctx, res.Msg.OidcConfig.IssuerUrl)
	if err != nil {
		return "", "", errors.Wrap(err, "error initializing OIDC provider")
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
