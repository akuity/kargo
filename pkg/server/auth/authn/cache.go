package authn

import (
	"context"
	"crypto/sha256"
	"time"

	utilcache "k8s.io/apimachinery/pkg/util/cache"

	"github.com/akuity/kargo/pkg/server/user"
)

// tokenCacheSize bounds how many tokens a token cache remembers at once. Past
// it, the least recently used token is forgotten first.
const tokenCacheSize = 4096

// tokenCache remembers the Identity each token resolved to.
type tokenCache struct {
	next  Authenticator
	ttl   time.Duration
	clock utilcache.Clock
	cache *utilcache.LRUExpireCache
}

var _ Authenticator = (*tokenCache)(nil)

// WithTokenCache returns an Authenticator that remembers, for up to ttl, the
// Identity each token next resolved to, so that a token presented again in
// that time is not verified again. Without it, every request presenting a
// Kubernetes token costs a TokenReview, and every request presenting an
// OpenID Connect token a ServiceAccount lookup.
//
// Only tokens next accepted are remembered: a rejection, or a failure to
// check, is asked of next again every time. No token is remembered past its
// own expiry. Tokens are remembered by their SHA-256 digest, never in the
// clear.
//
// The cost is that a token revoked, or a ServiceAccount mapping changed, is
// not noticed until the token's entry expires, so ttl should be short. A ttl
// of zero or less disables the cache and returns next itself.
func WithTokenCache(next Authenticator, ttl time.Duration) Authenticator {
	if ttl <= 0 {
		return next
	}
	return newTokenCache(next, ttl, realClock{})
}

func newTokenCache(
	next Authenticator,
	ttl time.Duration,
	clock utilcache.Clock,
) *tokenCache {
	return &tokenCache{
		next:  next,
		ttl:   ttl,
		clock: clock,
		cache: utilcache.NewLRUExpireCacheWithClock(tokenCacheSize, clock),
	}
}

// Authenticate implements Authenticator.
func (c *tokenCache) Authenticate(
	ctx context.Context,
	rawToken string,
) (user.Identity, bool, error) {
	key := sha256.Sum256([]byte(rawToken))
	if cached, ok := c.cache.Get(key); ok {
		if id, ok := cached.(user.Identity); ok {
			return id, true, nil
		}
	}
	id, ok, err := c.next.Authenticate(ctx, rawToken)
	if err != nil || !ok {
		return id, ok, err
	}
	if ttl := c.ttlFor(rawToken); ttl > 0 {
		c.cache.Add(key, id, ttl)
	}
	return id, true, nil
}

// ttlFor returns how long an accepted token may be remembered: the cache's
// TTL, or less if the token expires sooner. The token has been verified by
// now, so its expiry can be believed.
func (c *tokenCache) ttlFor(rawToken string) time.Duration {
	claims, ok := unverifiedClaims(rawToken)
	if !ok || claims.ExpiresAt == nil {
		return c.ttl
	}
	return min(c.ttl, claims.ExpiresAt.Sub(c.clock.Now()))
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
