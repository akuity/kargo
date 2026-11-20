package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NewIdleTimeoutTransport returns a RoundTripper that cancels a request when
// timeout passes between reads of its response body, whether the server or the
// caller is idle. It bounds a stalled transfer, not the total duration of one
// that is progressing.
func NewIdleTimeoutTransport(
	next http.RoundTripper,
	timeout time.Duration,
) http.RoundTripper {
	return idleTimeoutTransport{next: next, timeout: timeout}
}

// idleTimeoutError is the cause of a request canceled for an idle body.
type idleTimeoutError struct {
	timeout time.Duration
}

func (e idleTimeoutError) Error() string {
	return fmt.Sprintf("no response data received for %s", e.timeout)
}

type idleTimeoutTransport struct {
	next    http.RoundTripper
	timeout time.Duration
}

// WrappedRoundTripper returns the transport beneath.
func (t idleTimeoutTransport) WrappedRoundTripper() http.RoundTripper {
	return t.next
}

// RoundTrip implements http.RoundTripper.
func (t idleTimeoutTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	ctx, cancel := context.WithCancelCause(req.Context())
	res, err := t.next.RoundTrip(req.WithContext(ctx))
	if err != nil {
		cancel(nil)
		return nil, err
	}
	if res.Body == nil || res.Body == http.NoBody {
		cancel(nil)
		return res, nil
	}
	res.Body = &idleBody{
		ReadCloser: res.Body,
		ctx:        ctx,
		cancel:     cancel,
		timeout:    t.timeout,
		timer: time.AfterFunc(t.timeout, func() {
			cancel(idleTimeoutError{timeout: t.timeout})
		}),
	}
	return res, nil
}

// idleBody cancels its request when no data is read for timeout.
type idleBody struct {
	io.ReadCloser
	ctx     context.Context
	cancel  context.CancelCauseFunc
	timeout time.Duration
	timer   *time.Timer
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.timer.Reset(b.timeout)
	}
	if err == nil {
		return n, nil
	}
	b.timer.Stop()
	if errors.Is(err, io.EOF) {
		// The transport releases the connection before returning EOF.
		b.cancel(nil)
		return n, err
	}
	var idleErr idleTimeoutError
	if cause := context.Cause(b.ctx); errors.As(cause, &idleErr) &&
		!errors.As(err, &idleErr) {
		return n, fmt.Errorf("%w: %w", cause, err)
	}
	return n, err
}

func (b *idleBody) Close() error {
	b.timer.Stop()
	defer b.cancel(nil)
	return b.ReadCloser.Close()
}
