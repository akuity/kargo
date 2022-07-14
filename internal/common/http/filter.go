package http

import "net/http"

// Filter is an interface to be implemented by components that can decorate a
// http.HandlerFunc.
type Filter interface {
	// Decorate decorates one http.HandlerFunc with another.
	Decorate(http.HandlerFunc) http.HandlerFunc
}
