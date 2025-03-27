package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/bacongobbler/browser"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	libConfig "github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	"github.com/akuity/kargo/internal/kubeclient"
)

const (
	defaultRandStringCharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	authMethodAdmin      = "admin"
	authMethodKubeconfig = "kubeconfig"
	authMethodSSO        = "sso"
)

//go:embed assets
var assets embed.FS

type loginOptions struct {
	Config        libConfig.CLIConfig
	InsecureTLS   bool
	UseAdmin      bool
	UseKubeconfig bool
	UseSSO        bool
	Password      string
	CallbackPort  int
	ServerAddress string
}

func NewCommand(
	cfg libConfig.CLIConfig,
) *cobra.Command {
	cmdOpts := &loginOptions{
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "login [SERVER_ADDRESS] (--admin | --kubeconfig | --sso)",
		Args:  option.MaximumNArgs(1),
		Short: "Log in to a Kargo API server",
		Example: templates.Example(`
# Log in using SSO
kargo login https://kargo.example.com --sso

# Log in (again) using the last used server address and method
kargo login

# Log in using the admin user
kargo login https://kargo.example.com --admin

# Log in using the local kubeconfig
kargo login https://kargo.example.com --kubeconfig

# Log in using the local kubeconfig and ignore cert warnings
kargo login https://kargo.example.com --kubeconfig --insecure-skip-tls-verify
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the login options to the provided command.
func (o *loginOptions) addFlags(cmd *cobra.Command) {
	option.InsecureTLS(cmd.PersistentFlags(), &o.InsecureTLS)

	cmd.Flags().BoolVar(&o.UseAdmin, "admin", false,
		"Log in as the Kargo admin user. If set, --kubeconfig and --sso must not be set.")
	cmd.Flags().BoolVar(&o.UseKubeconfig, "kubeconfig", false,
		"Log in using a token obtained from the local Kubernetes configuration's current context. "+
			"If set, --admin and --sso must not be set.")
	cmd.Flags().StringVar(&o.Password, "password", "",
		"Specify the password for non-interactive admin user login. Only used when --admin is specified.")
	cmd.Flags().BoolVar(&o.UseSSO, "sso", false,
		"Log in using OpenID Connect and the server's configured identity provider. "+
			"If set, --admin and --kubeconfig must not be set.")
	cmd.Flags().IntVar(&o.CallbackPort, "port", 0,
		"Port to use for the callback URL; 0 selects any available, unprivileged port. "+
			"Only used when --sso is specified.")

	// If we do not have a saved configuration, we require the user to specify a
	// login method. If we do have a saved configuration, we allow the user to
	// omit the login method and use the last used method.
	if o.Config.AuthMethod == "" {
		cmd.MarkFlagsOneRequired("admin", "kubeconfig", "sso")
	}

	cmd.MarkFlagsMutuallyExclusive("admin", "kubeconfig", "sso")
}

// complete sets the options from the command arguments.
func (o *loginOptions) complete(args []string) {
	// Use the API address in config as a default address
	o.ServerAddress = o.Config.APIAddress
	if len(args) == 1 {
		o.ServerAddress = normalizeURL(args[0])
	}

	// If no auth method flag was explicitly set, use the stored method
	if !o.UseAdmin && !o.UseKubeconfig && !o.UseSSO && o.Config.AuthMethod != "" {
		switch o.Config.AuthMethod {
		case authMethodAdmin:
			o.UseAdmin = true
		case authMethodKubeconfig:
			o.UseKubeconfig = true
		case authMethodSSO:
			o.UseSSO = true
		}
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *loginOptions) validate() error {
	if o.ServerAddress == "" {
		return errors.New("server address is required")
	}
	return nil
}

// run logs in to the Kargo API server using the method specified by the options.
func (o *loginOptions) run(ctx context.Context) error {
	var bearerToken, refreshToken string
	var err error

	var authMethod string
	switch {
	case o.UseAdmin:
		authMethod = authMethodAdmin
		for {
			if o.Password != "" {
				break
			}
			prompt := &survey.Password{
				Message: "Admin user password",
			}
			if err = survey.AskOne(prompt, &o.Password); err != nil {
				return err
			}
		}
		if bearerToken, err = adminLogin(ctx, o.ServerAddress, o.Password, o.InsecureTLS); err != nil {
			return err
		}
	case o.UseKubeconfig:
		authMethod = authMethodKubeconfig
		if bearerToken, err = kubeconfigLogin(ctx); err != nil {
			return err
		}
	case o.UseSSO:
		authMethod = authMethodSSO
		if bearerToken, refreshToken, err = ssoLogin(
			ctx, o.ServerAddress, o.CallbackPort, o.InsecureTLS,
		); err != nil {
			return err
		}
	default:
		// This should never happen.
		return errors.New("internal error: no login method selected")
	}

	if o.InsecureTLS {
		// When the user specifies during login that they want to ignore cert
		// warnings, we will force them to periodically re-assess that choice
		// by NOT using refresh tokens and requiring them to re-authenticate
		// instead. Since we plan not to use the refresh token for such a case,
		// it's more secure to throw it away immediately.
		refreshToken = ""
	}

	if o.Config.APIAddress != o.ServerAddress {
		o.Config = libConfig.NewDefaultCLIConfig()
	}

	o.Config.AuthMethod = authMethod
	o.Config.APIAddress = o.ServerAddress
	o.Config.BearerToken = bearerToken
	o.Config.RefreshToken = refreshToken
	o.Config.InsecureSkipTLSVerify = o.InsecureTLS

	if err = libConfig.SaveCLIConfig(o.Config); err != nil {
		return fmt.Errorf("error persisting configuration: %w", err)
	}
	return nil
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
		return "", fmt.Errorf("error retrieving public configuration from server: %w", err)
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
		return "", fmt.Errorf("error logging in as admin user: %w", err)
	}

	return loginRes.Msg.IdToken, nil
}

// kubeconfigLogin gleans a bearer token from the local kubeconfig's current
// context.
func kubeconfigLogin(ctx context.Context) (string, error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return "", fmt.Errorf("error loading kubeconfig: %w", err)
	}
	bearerToken, err := kubeclient.GetCredential(ctx, restCfg)
	if err != nil {
		return "", fmt.Errorf("error retrieving bearer token from kubeconfig: %w", err)
	}
	return bearerToken, nil
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
		return "", "", fmt.Errorf("error retrieving public configuration from server: %w", err)
	}

	if res.Msg.OidcConfig == nil {
		return "", "", errors.New("server does not support OpenID Connect")
	}

	scopes := res.Msg.OidcConfig.Scopes

	httpClient := cleanhttp.DefaultClient()
	if insecureTLS {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}
	ctx = oidc.ClientContext(ctx, httpClient)

	provider, err := oidc.NewProvider(ctx, res.Msg.OidcConfig.IssuerUrl)
	if err != nil {
		return "", "", fmt.Errorf("error initializing OIDC provider: %w", err)
	}

	providerClaims := struct {
		ScopesSupported []string `json:"scopes_supported"`
	}{}
	if err = provider.Claims(&providerClaims); err != nil {
		return "", "", fmt.Errorf("error retrieving provider claims: %w", err)
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
		return "", "", fmt.Errorf("error creating callback listener: %w", err)
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
		return "", "", fmt.Errorf("error generating state: %w", err)
	}

	codeCh := make(chan string)
	errCh := make(chan error)
	go receiveAuthCode(ctx, listener, state, codeCh, errCh)

	codeVerifier, codeChallenge, err := createPCKEVerifierAndChallenge()
	if err != nil {
		return "", "", fmt.Errorf("error creating PCKE code verifier and code challenge: %w", err)
	}
	u := cfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	if err = browser.Open(u); err != nil {
		return "", "", fmt.Errorf("error opening system default browser: %w", err)
	}

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		return "", "", fmt.Errorf("error in callback handler: %w", err)
	case <-time.After(5 * time.Minute):
		return "", "", errors.New("timed out waiting for user to complete authentication")
	case <-ctx.Done():
		return "", "", ctx.Err()
	}

	token, err := cfg.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return "", "", fmt.Errorf("error exchanging auth code for token: %w", err)
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
		case errCh <- fmt.Errorf("error running temporary HTTP server: %w", err):
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
		return "", "", fmt.Errorf("error creating PKCE code verifier: %w", err)
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

// normalizeURL ensures a given address has a scheme, defaulting to "https".
func normalizeURL(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return address
	}

	// If no scheme is present, prepend "https://"
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}

	// Parse to normalize
	parsedURL, err := url.Parse(address)
	if err != nil {
		return address // Return the best attempt if parsing fails
	}
	return parsedURL.String()
}

var splashHTML = []byte(`<!DOCTYPE html>
<html>
<head>
  <meta charset='utf-8'>
  <meta http-equiv='X-UA-Compatible' content='IE=edge'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>Kargo</title>
  <link rel="shortcut icon" type="image/png" href="/assets/kargo.png"/>
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
