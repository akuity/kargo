package kubeclient

import (
	"net/http"

	netutil "k8s.io/apimachinery/pkg/util/net"

	ctxutil "github.com/akuityio/kargo/internal/context"
)

var (
	_ netutil.RoundTripperWrapper = &authRoundTripper{}
)

type authRoundTripper struct {
	rt http.RoundTripper
}

func newAuthRoundTripper(rt http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{
		rt: rt,
	}
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if v, ok := ctxutil.GetAuthCredential(req.Context()); ok {
		req.Header.Set("Authorization", v)
	}
	return rt.rt.RoundTrip(req)
}

func (rt *authRoundTripper) WrappedRoundTripper() http.RoundTripper {
	return rt.rt
}
