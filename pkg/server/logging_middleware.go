package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/user"
)

// LoggingMiddleware returns Gin middleware that records one line per request
// using Kargo's own logger, so that LOG_LEVEL governs the request log as it
// governs everything else.
//
// Each line names the actor the request was authenticated as, in the same
// form used to attribute Promotions and Events, so that a request log can
// answer who did something and not only where it came from. A request that
// never authenticated is attributed to EventActorUnknown.
//
// A server error is recorded at error level. A request that was refused is
// recorded at info level, because a refusal is never routine and is the first
// thing anyone looks for when a user reports being unable to do something.
// Everything else is operational detail and recorded at debug level, unless
// REQUEST_LOG_ALL_ENABLED raises it to info level.
//
// The address a request came from is recorded only when
// REQUEST_LOG_SOURCE_IP_ENABLED is set, because it is personal data and is
// usually better recorded by whatever proxy sits in front of the server.
//
// These settings are read from the environment here, rather than taken as
// arguments, so that every router using this middleware honors them alike.
//
// This must be the outermost middleware, so that the status and any reported
// error it records are the ones the client actually received.
func LoggingMiddleware() gin.HandlerFunc {
	return loggingMiddleware(loggingConfigFromEnv())
}

type loggingConfig struct {
	// AllEnabled records routine requests at info level instead of debug level.
	AllEnabled bool `envconfig:"REQUEST_LOG_ALL_ENABLED"`
	// SourceIPEnabled adds the address each request came from.
	SourceIPEnabled bool `envconfig:"REQUEST_LOG_SOURCE_IP_ENABLED"`
	// TrustedProxies are the proxies whose X-Forwarded-For entries and
	// ClientIPHeader are believed when resolving a request's source address.
	TrustedProxies trustedProxies `envconfig:"REQUEST_LOG_TRUSTED_PROXIES"`
	// ClientIPHeader names a header that every trusted proxy sets to the
	// client's address, e.g. CF-Connecting-IP.
	ClientIPHeader string `envconfig:"REQUEST_LOG_CLIENT_IP_HEADER"`
}

func loggingConfigFromEnv() loggingConfig {
	cfg := loggingConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// trustedProxies is decoded from a comma-separated list of addresses and CIDRs.
type trustedProxies []netip.Prefix

func (t *trustedProxies) Decode(value string) error {
	var prefixes []netip.Prefix
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			addr, addrErr := netip.ParseAddr(item)
			if addrErr != nil || addr.Zone() != "" {
				return fmt.Errorf("invalid trusted proxy %q: expected an IP address or CIDR", item)
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		// Addresses are compared unmapped, so an IPv4-mapped entry has to be
		// unmapped too or it would never match.
		if addr := prefix.Addr(); addr.Is4In6() {
			if prefix.Bits() < 96 {
				return fmt.Errorf("invalid trusted proxy %q: prefix is wider than the IPv4-mapped range", item)
			}
			prefix = netip.PrefixFrom(addr.Unmap(), prefix.Bits()-96)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	*t = prefixes
	return nil
}

func (t trustedProxies) contains(addr netip.Addr) bool {
	// A prefix never contains a zoned address.
	addr = addr.WithZone("")
	for _, prefix := range t {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// parseForwardedAddr parses an address as it may appear in a forwarding header:
// bare, bracketed, or followed by a port, which some proxies append.
func parseForwardedAddr(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(value)
	if addrPort, err := netip.ParseAddrPort(value); err == nil {
		return addrPort.Addr().Unmap(), true
	}
	value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
	addr, err := netip.ParseAddr(value)
	return addr.Unmap(), err == nil
}

// sourceIP returns the address the request came from. That is the peer on the
// other end of the connection, which a client cannot choose, unless the peer
// is a trusted proxy. In that case it is the client IP header, if one is
// configured and holds an address, and otherwise the rightmost
// X-Forwarded-For entry that is not itself a trusted proxy. Every entry to
// the right of that one was appended by a proxy we trust, so it is the last
// address the client could not have made up.
func (c loggingConfig) sourceIP(r *http.Request) string {
	addrPort, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	source := addrPort.Addr().Unmap()
	if !c.TrustedProxies.contains(source) {
		return source.String()
	}
	if c.ClientIPHeader != "" {
		if addr, ok := parseForwardedAddr(r.Header.Get(c.ClientIPHeader)); ok {
			return addr.String()
		}
	}
	forwarded := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		if strings.TrimSpace(forwarded[i]) == "" {
			continue
		}
		addr, ok := parseForwardedAddr(forwarded[i])
		if !ok {
			break
		}
		source = addr
		if !c.TrustedProxies.contains(source) {
			break
		}
	}
	return source.String()
}

func loggingMiddleware(cfg loggingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Taken on the way in and used for the line recorded below, so that line
		// does not depend on what anything within does to the request's context.
		ctx := c.Request.Context()
		logger := logging.LoggerFromContext(ctx)
		if cfg.SourceIPEnabled {
			// Bound to the request's logger, so that everything logged on the
			// request's behalf says where it came from.
			logger = logger.WithValues("sourceIP", cfg.sourceIP(c.Request))
			// As received, and so only as trustworthy as whatever sent it.
			if forwarded := c.Request.Header.Values("X-Forwarded-For"); len(forwarded) > 0 {
				logger = logger.WithValues("forwardedFor", strings.Join(forwarded, ", "))
			}
			c.Request = c.Request.WithContext(logging.ContextWithLogger(ctx, logger))
		}

		c.Next()

		status := c.Writer.Status()
		// The authentication middleware binds the actor to the request context
		// on the way in, so it is available here on the way out.
		actor := kargoapi.EventActorUnknown
		if u, ok := user.InfoFromContext(c.Request.Context()); ok {
			actor = api.FormatEventUserActor(u)
		}
		// Without stack traces, because this middleware sits above every handler
		// and every other middleware, so a trace from here describes only the path
		// through Gin and reveals nothing about what went wrong.
		logger = logger.
			WithoutStackTraces().
			WithValues(
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"actor", actor,
				"status", status,
				// Logged as a string because the encoder would otherwise render a
				// duration as a bare number of seconds.
				"duration", time.Since(start).String(),
			)
		reported := c.Errors.Last()

		if status < http.StatusInternalServerError {
			if reported != nil {
				logger = logger.WithValues("error", reported.Err.Error())
			}
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				logger.Info("refused request")
				return
			}
			if cfg.AllEnabled {
				logger.Info("handled request")
			} else {
				logger.Debug("handled request")
			}
			return
		}
		if reported == nil {
			// Nothing was reported, which means a handler answered with a server
			// error on its own rather than leaving that to the error-handling
			// middleware.
			logger.Error(errors.New("no error was reported"), "handled request")
			return
		}
		logger.Error(reported.Err, "handled request")
	}
}
