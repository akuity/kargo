package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/bacongobbler/browser"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/cli/client"
	libConfig "github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/kubeclient"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

const (
	flagAdmin                = "admin"
	flagKubeconfig           = "kubeconfig"
	flagPassword             = "password"
	flagPort                 = "port"
	flagSSO                  = "sso"
	defaultRandStringCharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

//go:embed assets
var assets embed.FS

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login server-address",
		Args:    option.ExactArgs(1),
		Short:   "Log in to a Kargo API server",
		Example: "kargo login https://kargo.example.com --sso",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			useAdmin, err := cmd.Flags().GetBool(flagAdmin)
			if err != nil {
				return err
			}

			useKubeconfig, err := cmd.Flags().GetBool(flagKubeconfig)
			if err != nil {
				return err
			}

			useSSO, err := cmd.Flags().GetBool(flagSSO)
			if err != nil {
				return err
			}

			var flagCount int
			if useAdmin {
				flagCount++
			}
			if useKubeconfig {
				flagCount++
			}
			if useSSO {
				flagCount++
			}
			if flagCount != 1 {
				return errors.Errorf(
					"please specify exactly one of --admin, --kubeconfig, or --sso",
				)
			}

			serverAddress := args[0]
			var bearerToken, refreshToken string
			if useAdmin {
				var password string
				if password, err = cmd.Flags().GetString(flagPassword); err != nil {
					return err
				}

				for {
					if password != "" {
						break
					}
					prompt := &survey.Password{
						Message: "Admin user password",
					}
					if err = survey.AskOne(prompt, &password); err != nil {
						return err
					}
				}

				if bearerToken, err =
					adminLogin(ctx, serverAddress, password, opt.InsecureTLS); err != nil {
					return err
				}
			} else if useKubeconfig {
				bearerToken, err = kubeconfigLogin(ctx)
				if err != nil {
					return err
				}
			} else {
				var callbackPort int
				callbackPort, err = cmd.Flags().GetInt(flagPort)
				if err != nil {
					return err
				}

				if bearerToken, refreshToken, err =
					ssoLogin(ctx, serverAddress, callbackPort, opt.InsecureTLS); err != nil {
					return err
				}
			}

			if opt.InsecureTLS {
				// When the user specifies during login that they want to ignore cert
				// warnings, we will force them to periodically re-assess that choice
				// by NOT using refresh tokens and requiring them to re-authenticate
				// instead. Since we plan not to use the refresh token for such a case,
				// it's more secure to throw it away immediately.
				refreshToken = ""
			}

			err = libConfig.SaveCLIConfig(
				libConfig.CLIConfig{
					APIAddress:            serverAddress,
					BearerToken:           bearerToken,
					RefreshToken:          refreshToken,
					InsecureSkipTLSVerify: opt.InsecureTLS,
				},
			)
			return errors.Wrap(err, "error persisting configuration")
		},
	}
	cmd.Flags().BoolP(
		flagAdmin,
		"a",
		false,
		"Log in as the Kargo admin user; mutually exclusive with --kubeconfig and "+
			"--sso",
	)
	cmd.Flags().BoolP(
		flagKubeconfig,
		"k",
		false,
		"Log in using a token obtained from the local Kubernetes configuration's "+
			"current context; mutually exclusive with --admin and --sso",
	)
	cmd.Flags().StringP(
		flagPassword,
		"P",
		"",
		"Specify the password for non-interactive admin user login; only used "+
			"with --admin",
	)
	cmd.Flags().IntP(
		flagPort,
		"p",
		0,
		"Port to use for the callback URL; 0 selects any available, "+
			"unprivileged port; only used with --sso",
	)
	cmd.Flags().BoolP(
		flagSSO,
		"s",
		false,
		"Log in using OpenID Connect and the server's configured identity "+
			"provider; mutually exclusive with --admin and --kubeconfig",
	)
	return cmd
}

func adminLogin(
	ctx context.Context,
	serverAddress string,
	password string,
	insecureTLS bool,
) (string, error) {
	kargoClient := client.GetClient(serverAddress, "", insecureTLS)

	cfgRes, err := kargoClient.GetPublicConfig(
		ctx,
		connect.NewRequest(&v1alpha1.GetPublicConfigRequest{}),
	)
	if err != nil {
		return "", errors.Wrap(
			err,
			"error retrieving public configuration from server",
		)
	}

	if !cfgRes.Msg.AdminAccountEnabled {
		return "", errors.New("server does not support admin user login")
	}

	loginRes, err := kargoClient.AdminLogin(
		ctx,
		connect.NewRequest(&v1alpha1.AdminLoginRequest{
			Password: password,
		}),
	)
	if err != nil {
		return "", errors.Wrap(err, "error logging in as admin user")
	}

	return loginRes.Msg.IdToken, nil
}

// kubeconfigLogin gleans a bearer token from the local kubeconfig's current
// context.
func kubeconfigLogin(ctx context.Context) (string, error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "error loading kubeconfig")
	}
	bearerToken, err := kubeclient.GetCredential(ctx, restCfg)
	return bearerToken,
		errors.Wrap(err, "error retrieving bearer token from kubeconfig")
}

// ssoLogin performs a login using OpenID Connect. It first retrieves
// non-sensitive configuration from the Kargo API server, then uses that
// configuration to perform an authorization code flow with PKCE with the
// identity provider specified by the API server. Upon success, it returns
// the ID token and refresh token.
func ssoLogin(
	ctx context.Context,
	serverAddress string,
	callbackPort int,
	insecureTLS bool,
) (string, string, error) {
	kargoClient := client.GetClient(serverAddress, "", insecureTLS)

	res, err := kargoClient.GetPublicConfig(
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

	scopes := res.Msg.OidcConfig.Scopes

	ctx = oidc.ClientContext(
		ctx,
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureTLS, // nolint: gosec
				},
			},
		},
	)
	provider, err := oidc.NewProvider(ctx, res.Msg.OidcConfig.IssuerUrl)
	if err != nil {
		return "", "", errors.Wrap(err, "error initializing OIDC provider")
	}

	providerClaims := struct {
		ScopesSupported []string `json:"scopes_supported"`
	}{}
	if err = provider.Claims(&providerClaims); err != nil {
		return "", "", errors.Wrap(err, "error retrieving provider claims")
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
		return "", "", errors.Wrap(err, "error creating callback listener")
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
	if res.Msg.OidcConfig.CliClientId != "" {
		// There is an OIDC client ID specifically meant for CLI use
		cfg.ClientID = res.Msg.OidcConfig.CliClientId
	}

	// Per the spec, this must be guessable with probability <= 2^(-128). The
	// following call generates one of 52^24 random strings, ~= 2^136
	// possibilities.
	//
	// See: https://www.rfc-editor.org/rfc/rfc6749#section-10.10
	state, err := randString(24)
	if err != nil {
		return "", "", errors.Wrap(err, "error generating state")
	}

	codeCh := make(chan string)
	errCh := make(chan error)
	go receiveAuthCode(ctx, listener, state, codeCh, errCh)

	codeVerifier, codeChallenge, err := createPCKEVerifierAndChallenge()
	if err != nil {
		return "", "", errors.Wrap(
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
		return "", "", errors.Wrap(err, "error opening system default browser")
	}

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		return "", "", errors.Wrap(err, "error in callback handler")
	case <-time.After(5 * time.Minute):
		return "", "", errors.New(
			"timed out waiting for user to complete authentication",
		)
	case <-ctx.Done():
		return "", "", ctx.Err()
	}

	token, err := cfg.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return "", "", errors.Wrap(err, "error exchanging auth code for token")
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return "", "", errors.New("no id_token in token response")
	}

	// Slight delay to allow all assets used by the splash page to be served up
	<-time.After(2 * time.Second)

	return idToken, token.RefreshToken, nil
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

	mux.Handle("/assets/", http.FileServer(http.FS(assets)))

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
				_, _ = w.Write(splashHTML)
			case <-r.Context().Done():
				w.WriteHeader(http.StatusInternalServerError)
			}

		},
	)

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Minute,
	}
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

var splashHTML = []byte(`<!DOCTYPE html>
<html>
<head>
  <meta charset='utf-8'>
  <meta http-equiv='X-UA-Compatible' content='IE=edge'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>Kargo</title>
  <link rel="shortcut icon" type="image/jpg" href="/assets/favicon.ico"/>
  <link rel='stylesheet' type='text/css' media='screen' href='/assets/splash.css'>
</head>
<body>
  <div class="splash">
    <img src="/assets/kargo.png" alt="Kargo" />
    <p>You are now logged in and may resume using the Kargo CLI.</p>
  </div>
</body>
</html>
`)
