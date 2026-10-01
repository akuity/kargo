package rest

import "github.com/gin-gonic/gin"

// Binder reads a value from a request. It reports false after responding
// with the error when the request carries no acceptable value.
type Binder[T any] func(c *gin.Context) (T, bool)

// Bind returns a handler that reads a value from the request with the Binder
// and hands it to handle. The handler runs only for a request that bound;
// every other request has been answered by the Binder already. It turns the
// "bind, check, return" opening every write handler would otherwise repeat
// into the handler's signature:
//
//	targets.POST("", guard.Require("create"), rest.Bind(bindTarget, h.create))
//
//	func (h *Handler) create(c *gin.Context, target *kargoapi.Target) { ... }
func Bind[T any](bind Binder[T], handle func(*gin.Context, T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := bind(c)
		if !ok {
			return
		}
		handle(c, v)
	}
}
