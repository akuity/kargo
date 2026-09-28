package net

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
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
	// awsMetadataV6 is the AWS EC2 instance metadata service's IPv6 address.
	// Unlike the IPv4 metadata endpoint, it lives in the unique-local (not
	// link-local) range, so it must be denied explicitly.
	awsMetadataV6 = net.ParseIP("fd00:ec2::254")
)

// isLinkLocal returns true if the given IP is in the IPv4 link-local range
// (169.254.0.0/16) or the IPv6 link-local range (fe80::/10).
func isLinkLocal(ip net.IP) bool {
	return linkLocalV4.Contains(ip) || linkLocalV6.Contains(ip)
}

// isBlockedMetadataAddress returns true if the given IP is a link-local
// address or the AWS EC2 IPv6 instance metadata address. Other private and
// internal addresses are permitted by design.
func isBlockedMetadataAddress(ip net.IP) bool {
	return isLinkLocal(ip) || ip.Equal(awsMetadataV6)
}

// safeDialContext returns a DialContext function that blocks connections to
// link-local IP addresses and the AWS EC2 IPv6 instance metadata address.
//
// If dialer is nil, a new dialer is used. Any supplied dialer is never
// modified.
//
// Whether supplied or defaulted, resolution and connection are left entirely to
// the dialer, so that its normal behavior for a host with multiple resolved
// addresses (trying each in turn until one succeeds) is preserved. Each
// candidate address is vetoed just before it is connected to, rather than
// picking a single address up front.
func safeDialContext(dialer *net.Dialer) func(
	ctx context.Context,
	network string,
	addr string,
) (net.Conn, error) {
	d := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if dialer != nil {
		d = *dialer
	}

	// net.Dialer ignores Control when ControlContext is set, so capture both
	// and let the replacement below delegate to whichever the caller
	// configured.
	priorControlContext := d.ControlContext
	if priorControlContext == nil && d.Control != nil {
		control := d.Control
		priorControlContext = func(
			_ context.Context,
			network string,
			address string,
			c syscall.RawConn,
		) error {
			return control(network, address, c)
		}
	}
	d.Control = nil

	d.ControlContext = func(
		ctx context.Context,
		network string,
		address string,
		c syscall.RawConn,
	) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("failed to parse address %q: %w", address, err)
		}

		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("failed to parse IP %q", host)
		}

		if isBlockedMetadataAddress(ip) {
			return fmt.Errorf(
				"connections to link-local or cloud instance metadata "+
					"addresses are not permitted (address %q)",
				host,
			)
		}

		if priorControlContext != nil {
			return priorControlContext(ctx, network, address, c)
		}
		return nil
	}
	return d.DialContext
}

// HardenTransport configures the given transport to block connections to
// link-local IP addresses (169.254.0.0/16 and fe80::/10) and the AWS EC2 IPv6
// instance metadata address (fd00:ec2::254). This prevents SSRF attacks
// targeting cloud instance metadata endpoints. It modifies the transport in
// place. All four of the transport's dial functions are replaced or cleared.
// All other properties of the transport are left alone, including proxy and TLS
// configuration.
//
// If dialer is nil, a new dialer with 30 second connect and keep-alive timeouts
// is used. Any supplied dialer is never modified.
func HardenTransport(transport *http.Transport, dialer *net.Dialer) {
	// Clearing the next three dial functions ensures that all connections go
	// through DialContext. Dial and DialTLS are deprecated in favor of
	// DialContext and DialTLSContext, but a caller-supplied transport could
	// still have them set, which would bypass the metadata check below.
	transport.Dial = nil    // nolint: staticcheck
	transport.DialTLS = nil // nolint: staticcheck
	transport.DialTLSContext = nil
	transport.DialContext = safeDialContext(dialer)
}
