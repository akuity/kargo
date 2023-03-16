package backoff

import (
	"math"
	"time"

	"github.com/akuityio/kargo/internal/rand"
)

var seededRand = rand.NewSeeded()

// JitteredExponential returns a time.Duration to wait before the next retry
// when employing a "jittered" exponential backoff. The value returned is based,
// in-part on the number of failures to date and the maximum desired retry
// interval. This value is "jittered" before it is returned. The importance of
// this is that if many failures of any sort occur in rapid succession, the
// retries will not only be staggered, but will become increasingly so as the
// failure count increases. This strategy helps to mitigate further
// complications in the event that the initial error was due to resource
// contention or rate limiting.
func JitteredExponential(
	failureCount int,
	maxDelay time.Duration,
) time.Duration {
	base := math.Pow(2, float64(failureCount))
	capped := math.Min(base, maxDelay.Seconds())
	jittered := (1 + seededRand.Float64()) * (capped / 2)
	scaled := jittered * float64(time.Second)
	return time.Duration(scaled)
}
