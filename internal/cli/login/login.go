package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bacongobbler/browser"
	"github.com/bufbuild/connect-go"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/utils/strings/slices"

	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

const (
	flagPort                 = "port"
	flagSSO                  = "sso"
	defaultRandStringCharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func NewCommand(_ *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login server-address",
		Args:    cobra.ExactArgs(1),
		Short:   "Log in to a Kargo API server",
		Example: "kargo login https://kargo.example.com --sso",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			useSSO, err := cmd.Flags().GetBool(flagSSO)
			if err != nil {
				return err
			}

			if !useSSO {
				return errors.Errorf("the login command currently only supports SSO")
			}

			fmt.Print(
				"\nWARNING: This command initiates authentication using the " +
					"specified server's configured OpenID Connect identity provider, " +
					"but the resulting ID token is not yet stored or used for any " +
					"purpose.\n\n",
			)

			callbackPort, err := cmd.Flags().GetInt(flagPort)
			if err != nil {
				return err
			}

			return ssoLogin(ctx, args[0], callbackPort)
		},
	}
	cmd.Flags().IntP(
		flagPort,
		"p",
		0,
		"Port to use for the callback URL; 0 selects any available, "+
			"unprivileged port",
	)
	cmd.Flags().BoolP(
		flagSSO,
		"s",
		false,
		"Log in using OpenID Connect and the server's configured identity provider",
	)
	return cmd
}

// ssoLogin performs a login using OpenID Connect. It first retrieves
// non-sensitive configuration from the Kargo API server, then uses that
// configuration to perform an authorization code flow with PKCE with the
// identity provider specified by the API server.
func ssoLogin(
	ctx context.Context,
	serverAddress string,
	callbackPort int,
) error {
	client := svcv1alpha1connect.NewKargoServiceClient(
		http.DefaultClient,
		serverAddress,
	)

	res, err := client.GetPublicConfig(
		ctx,
		connect.NewRequest(&v1alpha1.GetPublicConfigRequest{}),
	)
	if err != nil {
		return errors.Wrap(
			err,
			"error retrieving public configuration from server",
		)
	}

	if res.Msg.OidcConfig == nil {
		return errors.New("server does not support OpenID Connect")
	}

	scopes := res.Msg.OidcConfig.Scopes

	provider, err := oidc.NewProvider(ctx, res.Msg.OidcConfig.IssuerUrl)
	if err != nil {
		return errors.Wrap(err, "error initializing OIDC provider")
	}

	providerClaims := struct {
		ScopesSupported []string `json:"scopes_supported"`
	}{}
	if err = provider.Claims(&providerClaims); err != nil {
		return errors.Wrap(err, "error retrieving provider claims")
	}
	const offlineAccessScope = "offline_access"
	// If the provider supports the "offline_access" scope, request it so that
	// we can get a refresh token.
	if slices.Contains(providerClaims.ScopesSupported, offlineAccessScope) {
		scopes = append(scopes, offlineAccessScope)
	}

	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf("localhost:%d", callbackPort),
	)
	if err != nil {
		return errors.Wrap(err, "error creating callback listener")
	}

	cfg := oauth2.Config{
		ClientID: res.Msg.OidcConfig.ClientId,
		Endpoint: provider.Endpoint(),
		Scopes:   scopes,
		RedirectURL: fmt.Sprintf(
			"http://localhost:%s/auth/callback",
			strings.Split(listener.Addr().String(), ":")[1],
		),
	}

	// Per the spec, this must be guessable with probability <= 2^(-128). The
	// following call generates one of 52^24 random strings, ~= 2^136
	// possibilities.
	//
	// See: https://www.rfc-editor.org/rfc/rfc6749#section-10.10
	state, err := randString(24)
	if err != nil {
		return errors.Wrap(err, "error generating state")
	}

	codeCh := make(chan string)
	errCh := make(chan error)
	go receiveAuthCode(ctx, listener, state, codeCh, errCh)

	codeVerifier, codeChallenge, err := createPCKEVerifierAndChallenge()
	if err != nil {
		return errors.Wrap(
			err,
			"error creating PCKE code verifier and code challenge",
		)
	}
	url := cfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	if err = browser.Open(url); err != nil {
		return errors.Wrap(err, "error opening system default browser")
	}

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		return errors.Wrap(err, "error in callback handler")
	case <-time.After(5 * time.Minute):
		return errors.New(
			"timed out waiting for user to complete authentication",
		)
	case <-ctx.Done():
		return ctx.Err()
	}

	token, err := cfg.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return errors.Wrap(err, "error exchanging auth code for token")
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return errors.New("no id_token in token response")
	}

	// TODO: Do something more meaningful with these tokens
	fmt.Printf("ID token: %s\n\nRefresh token: %s\n", idToken, token.RefreshToken)

	return nil
}

// receiveAuthCode runs a web server that serves the callback endpoint for
// receiving the authorization code at the end of an authorization code flow.
// It returns the authorization code or any error that occurs via the provided
// channels.
func receiveAuthCode(
	ctx context.Context,
	listener net.Listener,
	state string,
	codeCh chan<- string,
	errCh chan<- error,
) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Minute,
	}
	mux.HandleFunc(
		"/auth/callback",
		func(w http.ResponseWriter, r *http.Request) {
			callbackState := r.FormValue("state")
			if callbackState == "" {
				callbackState = r.URL.Query().Get("state")
			}
			if callbackState == "" {
				select {
				case errCh <- errors.New("no state found in callback request"):
				case <-r.Context().Done():
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if callbackState != state {
				select {
				case errCh <- errors.New("unrecognized state in callback request"):
				case <-r.Context().Done():
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			code := r.FormValue("code")
			if code == "" {
				code = r.URL.Query().Get("code")
			}
			if code == "" {
				select {
				case errCh <- errors.New("no code found in callback request"):
				case <-r.Context().Done():
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			select {
			case codeCh <- code:
				w.WriteHeader(http.StatusOK)
				// TODO: Return a nicer page
				w.Write( // nolint: errcheck
					[]byte(
						"You are now logged in. You may close this window and resume " +
							"using the Kargo CLI.",
					),
				)
			case <-r.Context().Done():
				w.WriteHeader(http.StatusInternalServerError)
			}

		},
	)

	if err := srv.Serve(listener); err != nil {
		select {
		case errCh <- errors.Wrap(err, "error running temporary HTTP server"):
		case <-ctx.Done():
		}
	}
}

// createPCKEVerifierAndChallenge creates a PKCE code verifier and code
// challenge. The relationship between the two is that the code challenge is the
// base64 URL-encoded SHA256 hash of the code verifier. The returned code
// challenge can be sent to an identity provider at the start of an
// authorization code flow with PKCE, while the returned code verifier can be
// used to exchange the authorization code from the identity provider for a
// user's identity token and/or an access token.
func createPCKEVerifierAndChallenge() (string, string, error) {
	codeVerifier, err := randStringFromCharset(
		// This is the max length allowed by the spec.
		//
		// See: https://www.rfc-editor.org/rfc/rfc7636#section-4.1
		128,
		// These are the unreserved characters specified in the spec.
		"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~",
	)
	if err != nil {
		return "", "", errors.Wrap(err, "error creating PKCE code verifier")
	}
	codeChallengeHash := sha256.Sum256([]byte(codeVerifier))
	// [:] converts [32]byte to []byte
	codeChallenge := base64.RawURLEncoding.EncodeToString(codeChallengeHash[:])
	return codeVerifier, codeChallenge, nil
}

// randString generates a random string of the specified length using only
// characters from a default character set.
func randString(n int) (string, error) {
	return randStringFromCharset(n, defaultRandStringCharSet)
}

// randStringFromCharset generates a random string of the specified length using
// only characters from the specified character set.
func randStringFromCharset(n int, charset string) (string, error) {
	b := make([]byte, n)
	maxIdx := big.NewInt(int64(len(charset)))
	for i := 0; i < n; i++ {
		randIdx, err := rand.Int(rand.Reader, maxIdx)
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		// randIdx is necessarily safe to convert to int, because the max came from
		// an int
		randIdxInt := int(randIdx.Int64())
		b[i] = charset[randIdxInt]
	}
	return string(b), nil
}
