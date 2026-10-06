package net

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isLinkLocal(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "IPv4 link-local lower bound",
			ip:       "169.254.0.0",
			expected: true,
		},
		{
			name:     "IPv4 link-local metadata endpoint",
			ip:       "169.254.169.254",
			expected: true,
		},
		{
			name:     "IPv4 link-local upper bound",
			ip:       "169.254.255.255",
			expected: true,
		},
		{
			name:     "IPv4 just below link-local range",
			ip:       "169.253.255.255",
			expected: false,
		},
		{
			name:     "IPv4 just above link-local range",
			ip:       "169.255.0.0",
			expected: false,
		},
		{
			name:     "IPv4 private 10.x",
			ip:       "10.0.0.1",
			expected: false,
		},
		{
			name:     "IPv4 public",
			ip:       "8.8.8.8",
			expected: false,
		},
		{
			name:     "IPv4 loopback",
			ip:       "127.0.0.1",
			expected: false,
		},
		{
			name:     "IPv6 link-local",
			ip:       "fe80::1",
			expected: true,
		},
		{
			name:     "IPv6 link-local upper bound",
			ip:       "febf::ffff",
			expected: true,
		},
		{
			name:     "IPv6 just outside link-local",
			ip:       "fec0::1",
			expected: false,
		},
		{
			name:     "IPv6 loopback",
			ip:       "::1",
			expected: false,
		},
		{
			name:     "IPv6 public",
			ip:       "2001:db8::1",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.Equal(t, tt.expected, isLinkLocal(ip))
		})
	}
}

func Test_isBlockedMetadataAddress(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "IPv4 link-local metadata endpoint",
			ip:       "169.254.169.254",
			expected: true,
		},
		{
			name:     "IPv6 link-local",
			ip:       "fe80::1",
			expected: true,
		},
		{
			name:     "AWS IPv6 metadata endpoint",
			ip:       "fd00:ec2::254",
			expected: true,
		},
		{
			name:     "IPv6 unique-local neighbor of AWS metadata endpoint",
			ip:       "fd00:ec2::253",
			expected: false,
		},
		{
			name:     "IPv6 unrestricted unique-local",
			ip:       "fd12::1",
			expected: false,
		},
		{
			name:     "IPv4 private 10.x",
			ip:       "10.0.0.1",
			expected: false,
		},
		{
			name:     "IPv4 public",
			ip:       "8.8.8.8",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.Equal(t, tt.expected, isBlockedMetadataAddress(ip))
		})
	}
}

// listen starts a TCP listener on 127.0.0.1 that accepts and immediately
// closes a single connection, and returns it for the caller to dial against.
func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	return ln
}

func Test_safeDialContext(t *testing.T) {
	t.Run("dials a safe address", func(t *testing.T) {
		ln := listen(t)
		dial := safeDialContext(&net.Dialer{Timeout: 2 * time.Second})

		conn, err := dial(context.Background(), "tcp", ln.Addr().String())
		require.NoError(t, err)
		conn.Close()
	})

	t.Run("blocks a link-local address", func(t *testing.T) {
		dial := safeDialContext(&net.Dialer{Timeout: 2 * time.Second})

		_, err := dial(context.Background(), "tcp", "169.254.169.254:80")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
	})

	t.Run("blocks the AWS IPv6 metadata address", func(t *testing.T) {
		dial := safeDialContext(&net.Dialer{Timeout: 2 * time.Second})

		_, err := dial(context.Background(), "tcp", "[fd00:ec2::254]:80")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
	})

	t.Run("falls back to another resolved address when the first is unreachable", func(t *testing.T) {
		ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), "localhost")
		require.NoError(t, err)
		if len(ips) < 2 {
			t.Skip("localhost does not resolve to multiple addresses on this system")
		}

		// Listen only on the second resolved address, leaving the first
		// unreachable. A dialer that gives up after the first resolved
		// address fails would never reach this listener.
		ln, err := net.Listen("tcp", net.JoinHostPort(ips[1].IP.String(), "0"))
		require.NoError(t, err)
		defer ln.Close()
		go func() {
			acceptedConn, acceptErr := ln.Accept()
			if acceptErr == nil {
				acceptedConn.Close()
			}
		}()

		_, port, err := net.SplitHostPort(ln.Addr().String())
		require.NoError(t, err)

		dial := safeDialContext(&net.Dialer{Timeout: 2 * time.Second})
		conn, err := dial(context.Background(), "tcp", net.JoinHostPort("localhost", port))
		require.NoError(t, err)
		defer conn.Close()

		// Confirm the connection actually landed on our listener, and
		// didn't merely succeed by coincidence against something else
		// listening on the first resolved address.
		assert.Equal(t, ln.Addr().String(), conn.RemoteAddr().String())
	})

	t.Run("defaults to a 30s dialer when none is supplied", func(t *testing.T) {
		ln := listen(t)

		dial := safeDialContext(nil)
		conn, err := dial(context.Background(), "tcp", ln.Addr().String())
		require.NoError(t, err)
		conn.Close()
	})

	t.Run("does not mutate the supplied dialer", func(t *testing.T) {
		dialer := &net.Dialer{Timeout: 5 * time.Second}

		_ = safeDialContext(dialer)

		assert.Nil(t, dialer.ControlContext)
		assert.Nil(t, dialer.Control)
	})

	t.Run("runs the metadata check before a prior ControlContext", func(t *testing.T) {
		var priorCalledWith []string
		prior := func(_ context.Context, _ string, address string, _ syscall.RawConn) error {
			priorCalledWith = append(priorCalledWith, address)
			if strings.HasPrefix(address, "10.0.0.1:") {
				return errors.New("blocked by prior control")
			}
			return nil
		}
		dial := safeDialContext(&net.Dialer{
			Timeout:        2 * time.Second,
			ControlContext: prior,
		})

		t.Run("a blocked address never reaches the prior control", func(t *testing.T) {
			priorCalledWith = nil
			_, err := dial(context.Background(), "tcp", "169.254.169.254:80")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not permitted")
			assert.Empty(t, priorCalledWith)
		})

		t.Run("a safe address rejected by the prior control still fails", func(t *testing.T) {
			priorCalledWith = nil
			_, err := dial(context.Background(), "tcp", "10.0.0.1:80")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "blocked by prior control")
			assert.NotEmpty(t, priorCalledWith)
		})

		t.Run("a safe address allowed by the prior control succeeds", func(t *testing.T) {
			ln := listen(t)
			priorCalledWith = nil

			conn, err := dial(context.Background(), "tcp", ln.Addr().String())
			require.NoError(t, err)
			conn.Close()
			assert.NotEmpty(t, priorCalledWith)
		})
	})

	t.Run("wraps a prior deprecated Control the same way", func(t *testing.T) {
		ln := listen(t)

		var called bool
		prior := func(_, _ string, _ syscall.RawConn) error {
			called = true
			return nil
		}
		dial := safeDialContext(&net.Dialer{
			Timeout: 2 * time.Second,
			Control: prior,
		})

		conn, err := dial(context.Background(), "tcp", ln.Addr().String())
		require.NoError(t, err)
		conn.Close()
		assert.True(t, called, "expected the prior Control function to be invoked")
	})
}

func TestHardenTransport(t *testing.T) {
	t.Run("clears every dial field except DialContext", func(t *testing.T) {
		noopDial := func(_, _ string) (net.Conn, error) { return nil, nil }
		noopDialContext := func(_ context.Context, _, _ string) (net.Conn, error) { return nil, nil }

		transport := &http.Transport{
			Dial:           noopDial, // nolint: staticcheck
			DialTLS:        noopDial, // nolint: staticcheck
			DialTLSContext: noopDialContext,
			DialContext:    noopDialContext,
		}

		HardenTransport(transport, nil)

		assert.Nil(t, transport.Dial)    // nolint: staticcheck
		assert.Nil(t, transport.DialTLS) // nolint: staticcheck
		assert.Nil(t, transport.DialTLSContext)
		assert.NotNil(t, transport.DialContext)
	})

	t.Run("leaves other transport configuration alone", func(t *testing.T) {
		transport := &http.Transport{MaxIdleConns: 42}

		HardenTransport(transport, nil)

		assert.Equal(t, 42, transport.MaxIdleConns)
	})

	t.Run("hardened transport blocks metadata addresses end-to-end", func(t *testing.T) {
		transport := &http.Transport{}
		HardenTransport(transport, nil)

		req, err := http.NewRequestWithContext(
			context.Background(), http.MethodGet, "http://169.254.169.254/", nil,
		)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not permitted")
	})
}
