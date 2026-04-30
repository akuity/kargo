package net

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
