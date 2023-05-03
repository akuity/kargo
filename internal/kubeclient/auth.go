package kubeclient

import (
	"context"
	"net/http"

	"k8s.io/client-go/rest"
)

func GetCredential(ctx context.Context, cfg *rest.Config) (string, error) {
	cfg.Wrap(newAuthorizationHeaderHook)
	rc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Host, nil)
	if err != nil {
		return "", err
	}
	res, err := rc.Do(req)
	if err != nil {
		return "", err
	}
	return res.Header.Get(xKargoUserCredentialHeader), nil
}
