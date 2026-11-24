package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/server/config"
)

// ConfigureEngine applies the settings every Gin engine serving Kargo's API
// must share, so that an engine built outside this package behaves like the
// one built within it.
//
// Note: This sets Gin's mode, which is package-level state. Every engine in
// the process is in release mode as a result, which is what we want of all of
// them anyway.
func ConfigureEngine(engine *gin.Engine, cfg config.ServerConfig) error {
	// Suppresses Gin-related log noise that we don't want, even at development
	// time.
	gin.SetMode(gin.ReleaseMode)

	// Gin trusts every proxy by default, which would let any client choose the
	// IP address it is taken to have come from. An empty list trusts none.
	if err := engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return fmt.Errorf("error setting trusted proxies: %w", err)
	}
	// Consulted, in order, only for requests arriving from a trusted proxy.
	engine.RemoteIPHeaders = []string{"X-Forwarded-For"}
	if cfg.ClientIPHeader != "" {
		engine.RemoteIPHeaders = []string{cfg.ClientIPHeader, "X-Forwarded-For"}
	}
	return nil
}
