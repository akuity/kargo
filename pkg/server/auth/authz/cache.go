package authz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	utilcache "k8s.io/apimachinery/pkg/util/cache"

	"github.com/akuity/kargo/pkg/server/user"
)

// decisionCacheSize bounds how many decisions a decision cache remembers at
// once. Past it, the least recently used decision is forgotten first.
const decisionCacheSize = 16384

// decisionCache remembers the answers next gave.
type decisionCache struct {
	next     Authorizer
	allowTTL time.Duration
	denyTTL  time.Duration
	cache    *utilcache.LRUExpireCache
}

var _ Authorizer = (*decisionCache)(nil)

// WithDecisionCache returns an Authorizer that remembers each answer next
// gives about a subject and an operation: an allow for allowTTL, a denial for
// denyTTL. Without it, every authorized operation costs at least one
// SubjectAccessReview.
//
// Answers are remembered per subject, not per user, so every OpenID Connect
// user mapped to the same ServiceAccount shares them. A failure to answer is
// never remembered.
//
// The cost is that a change to RBAC is not noticed until the affected entries
// expire. Denials are typically held for less time than allows, so that a
// newly granted permission takes effect soon. A TTL of zero or less disables
// caching of that kind of answer; with both disabled, next itself is
// returned.
func WithDecisionCache(
	next Authorizer,
	allowTTL time.Duration,
	denyTTL time.Duration,
) Authorizer {
	if allowTTL <= 0 && denyTTL <= 0 {
		return next
	}
	return newDecisionCache(next, allowTTL, denyTTL, realClock{})
}

func newDecisionCache(
	next Authorizer,
	allowTTL time.Duration,
	denyTTL time.Duration,
	clock utilcache.Clock,
) *decisionCache {
	return &decisionCache{
		next:     next,
		allowTTL: allowTTL,
		denyTTL:  denyTTL,
		cache:    utilcache.NewLRUExpireCacheWithClock(decisionCacheSize, clock),
	}
}

// Authorize implements Authorizer.
func (c *decisionCache) Authorize(
	ctx context.Context,
	subject user.Subject,
	ra authv1.ResourceAttributes,
) (bool, error) {
	key, keyed := decisionKey(subject, ra)
	if keyed {
		if cached, ok := c.cache.Get(key); ok {
			if allowed, ok := cached.(bool); ok {
				return allowed, nil
			}
		}
	}
	allowed, err := c.next.Authorize(ctx, subject, ra)
	if err != nil {
		return false, err
	}
	ttl := c.denyTTL
	if allowed {
		ttl = c.allowTTL
	}
	if keyed && ttl > 0 {
		c.cache.Add(key, allowed, ttl)
	}
	return allowed, nil
}

// decisionKey identifies a question by the digest of everything asked: the
// whole subject, including its groups and extras, and the whole operation. It
// reports false if the question cannot be encoded, in which case it is not
// cached.
func decisionKey(
	subject user.Subject,
	ra authv1.ResourceAttributes,
) ([sha256.Size]byte, bool) {
	// encoding/json writes map keys in sorted order, so equal questions
	// encode identically.
	encoded, err := json.Marshal(struct {
		Subject user.Subject
		Access  authv1.ResourceAttributes
	}{Subject: subject, Access: ra})
	if err != nil {
		return [sha256.Size]byte{}, false
	}
	return sha256.Sum256(encoded), true
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
