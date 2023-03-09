package os

import (
	"context"
	"os/signal"
	"syscall"
)

func NotifyOnShutdown(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}
