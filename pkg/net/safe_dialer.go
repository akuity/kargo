package net

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	linkLocalV4 = net.IPNet{
		IP:   net.IP{169, 254, 0, 0},
		Mask: net.CIDRMask(16, 32),
	}
	linkLocalV6 = net.IPNet{
		IP:   net.ParseIP("fe80::"),
		Mask: net.CIDRMask(10, 128),
	}
)

// isLinkLocal returns true if the given IP is in the IPv4 link-local range
// (169.254.0.0/16) or the IPv6 link-local range (fe80::/10).
func isLinkLocal(ip net.IP) bool {
	return linkLocalV4.Contains(ip) || linkLocalV6.Contains(ip)
}

// SafeDialContext returns a DialContext function that blocks connections to
// link-local IP addresses (169.254.0.0/16 and fe80::/10). This prevents SSRF
// attacks targeting cloud instance metadata endpoints (e.g. 169.254.169.254).
//
// The returned function resolves the hostname before connecting and rejects the
// connection if all resolved addresses are link-local.
func SafeDialContext(dialer *net.Dialer) func(
	ctx context.Context,
	network string,
	addr string,
) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address %q: %w", addr, err)
		}

		// Resolve the hostname to IP addresses.
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host %q: %w", host, err)
		}

		// Filter out link-local addresses.
		var safe []net.IPAddr
		for _, ip := range ips {
			if !isLinkLocal(ip.IP) {
				safe = append(safe, ip)
			}
		}

		if len(safe) == 0 {
			return nil, fmt.Errorf(
				"connections to link-local addresses are not permitted "+
					"(host %q resolved to link-local IPs only)",
				host,
			)
		}

		// Dial using the first safe address.
		safeAddr := net.JoinHostPort(safe[0].IP.String(), port)
		return dialer.DialContext(ctx, network, safeAddr)
	}
}

// SafeTransport wraps the given transport's DialContext to block connections to
// link-local IP addresses.
func SafeTransport(t *http.Transport) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	t.DialContext = SafeDialContext(dialer)
	return t
}
