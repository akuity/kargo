package kubeclient

import (
	"net/http"

	netutil "k8s.io/apimachinery/pkg/util/net"
)

const (
	xKargoUserCredentialHeader = "X-Kargo-User-Credential"
)

var (
	_ netutil.RoundTripperWrapper = &credentialHook{}
)

type credentialHook struct {
	rt http.RoundTripper
}

func newAuthorizationHeaderHook(rt http.RoundTripper) http.RoundTripper {
	return &credentialHook{
		rt: rt,
	}
}

func (rt *authRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	if v, ok := AuthCredentialFromContext(req.Context()); ok {
		req.Header.Set("Authorization", v)
	}
	return res, err
}

func (rt *credentialHook) WrappedRoundTripper() http.RoundTripper {
	return rt.rt
}
