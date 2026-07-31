package credentials

import "time"

// CalculateCacheTTL calculates how long a credential may be cached for, given
// the credential's own expiry time and a safety margin. It returns nil when the
// credential expires within that margin, caching one that is spent, or nearly
// so, being how a caller comes to be handed a credential that has stopped
// working. A zero expiry time means the credential's lifetime is unknown, for
// which it returns a zero duration, deferring to whatever default TTL the cache
// was configured with.
func CalculateCacheTTL(expiry time.Time, margin time.Duration) *time.Duration {
	if expiry.IsZero() {
		return new(time.Duration)
	}
	remaining := time.Until(expiry) - margin
	if remaining <= 0 {
		return nil
	}
	return &remaining
}
