package http

import (
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// IPFilterConfig encapsulates IPFilterConfig configuration.
type IPFilterConfig struct {
	// AllowedRanges are the IP ranges that are permitted to send requests.
	AllowedRanges []net.IPNet
}

// ipFilter is a component that implements the Filter interface and can
// conditionally allow or disallow a request on the basis of the client's IP
// address.
type ipFilter struct {
	config IPFilterConfig
}

// NewIPFilter returns a component that implements the Filter interface and can
// conditionally allow or disallow a request on the basis of the client's IP
// address.
func NewIPFilter(config IPFilterConfig) Filter {
	return &ipFilter{
		config: config,
	}
}

func (i *ipFilter) Decorate(handle http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Requests are permitted only if they originated from an allowed IP.
		//
		// Try to get the IP from the X-FORWARDED-FOR header first. This header is
		// likely to be populated if there were any reverse proxies between the
		// client and the server.
		ipStr := r.Header.Get("X-FORWARDED-FOR")
		// If the X-FORWARDED-FOR header didn't contain an IP, fall back on
		// r.RemoteAddr
		if ipStr == "" {
			ipStr = r.RemoteAddr
		}
		// If we couldn't determine the IP, we don't allow the request to proceed
		if ipStr == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Remove trailing port info if there is any
		ipStr = strings.Split(ipStr, ":")[0]
		// Parse the IP string to get a net.IP
		ip := net.ParseIP(ipStr)
		if ip == nil {
			log.Errorf("could not parse %q as an IP", ipStr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Determine if this IP is allowed...
		var allowedIP bool
		// Loop through all allowed IP ranges...
		for _, allowedIPRange := range i.config.AllowedRanges {
			if allowedIPRange.Contains(ip) {
				allowedIP = true
				break
			}
		}
		if !allowedIP {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		handle(w, r)
	}
}
