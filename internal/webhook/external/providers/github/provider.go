package github

import (
	"errors"
	"io"
	"net/http"
	"os"

	gh "github.com/google/go-github/v71/github"

	xhttp "github.com/akuity/kargo/internal/http"
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
	// ErrUnsupportedEventType is returned when the received
	// request body does not conform to a github push event.
	ErrUnsupportedEventType = errors.New("unsupported event type")
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

func (p *provider) GetRepository(r *http.Request) (string, error) {
	var signature string
	if h := r.Header.Get(gh.SHA1SignatureHeader); h != "" {
		signature = h
	}
	if h := r.Header.Get(gh.SHA256SignatureHeader); h != "" {
		signature = h
	}
	if signature == "" {
		return "", xhttp.UnauthorizedError(ErrMissingSignature)
	}

	const maxBytes = 2 << 20
	lr := io.LimitReader(r.Body, maxBytes)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return "", xhttp.BadRequestErrorf("failed to read request body: %w", err)
	}

	// If we read exactly the maximum, the body might be larger
	if len(bodyBytes) == maxBytes {
		// Try to read one more byte
		buf := make([]byte, 1)
		var n int
		if n, err = r.Body.Read(buf); err != nil && err != io.EOF {
			return "", xhttp.BadRequestErrorf("failed to check for additional content: %w", err)
		}
		if n > 0 || err != io.EOF {
			return "", xhttp.BadRequestErrorf("response body exceeds maximum size of %d bytes", maxBytes)
		}
	}

	if err = gh.ValidateSignature(
		signature,
		bodyBytes,
		[]byte(p.secret),
	); err != nil {
		return "", xhttp.UnauthorizedError(ErrInvalidSignature)
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "push" {
		return "", xhttp.BadRequestError(ErrUnsupportedEventType)
	}

	e, err := gh.ParseWebHook(eventType, bodyBytes)
	if err != nil {
		return "", xhttp.BadRequestErrorf("failed to parse webhook event: %w", err)
	}

	pe, ok := e.(*gh.PushEvent)
	if !ok {
		return "", xhttp.BadRequestError(ErrUnsupportedEventType)
	}
	return *pe.Repo.HTMLURL, nil
}
