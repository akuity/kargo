package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/webhook/external/events"
)

var (
	// ErrMissingSignature is returned when the 'X-Hub-Signature-256'
	// header is not found or empty.
	ErrMissingSignature = errors.New("missing signature")
	// ErrInvalidSignature is returned when the 'X-Hub-Signature-256'
	// header is not the value that was expected.
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrSecretUnset is returned when the 'GH_WEBHOOK_SECRET'
	// environment variable is empty.
	ErrSecretUnset = errors.New("secret is unset")
)

type provider struct {
	secret string
}

func NewProvider() (*provider, error) {
	secret, ok := os.LookupEnv("GH_WEBHOOK_SECRET")
	if !ok {
		return nil, ErrSecretUnset
	}
	return &provider{secret: secret}, nil
}

func (p *provider) Name() string {
	return "github"
}

func (p *provider) Authenticate(r *http.Request) error {
	sig := r.Header.Get("X-Hub-Signature-256")
	if sig == "" || !strings.HasPrefix(sig, "sha256=") {
		return ErrMissingSignature
	}

	b, err := xhttp.PeakBody(r)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	expectedSig := sig[len("sha256="):]
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write(b)
	computedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(computedSig)) {
		return ErrInvalidSignature
	}
	return nil
}

func (p *provider) Event(r *http.Request) (events.Event, error) {
	if eventType := r.Header.Get("X-GitHub-Event"); eventType != "push" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
	pe := new(pushEvent)
	if err := json.NewDecoder(r.Body).Decode(pe); err != nil {
		return nil, fmt.Errorf("failed to decode push event: %w", err)
	}
	return pe, nil
}

// implements the event.Event interface
type pushEvent struct {
	Repo struct {
		// format: "https://github.com/fuskovic/wh-test-repo"
		URL string `json:"html_url"`
	} `json:"repository"`
	User struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"pusher"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
}

func (p *pushEvent) Repository() string {
	return git.NormalizeURL(p.Repo.URL)
}

func (p *pushEvent) PushedBy() string {
	return p.User.Name
}

func (p *pushEvent) Commit() string {
	return p.HeadCommit.ID
}
