//go:build e2e
//nolint:forcetypeassert
package http_promo_step

// This test implements the http promotion step example from
// https://github.com/akuity/kargo-examples (03-features/01-http-promo-step).
// The promotion posts a Slack-style message to an HTTP endpoint. The http step
// fails the promotion on a non-2xx response, so a successful promotion confirms
// the endpoint accepted the POST.
//
// The source example hard-codes the endpoint to the operator's host machine.
// This suite instead reads it from the test env (context.http_endpoint), which
// must point at an HTTP endpoint that accepts the POST and returns 2xx (e.g. an
// echo server). The multi-stage soak/verification pipeline is reduced to a
// single stage focused on the http step.

import (
	"testing"

	"github.com/akuity/kargo/hack/test/e2e/framework/utils"
)

// This file provides necessary setup for a test package to run environment setup for e2e test.
// Because golang doesn't allow import of test code, this code needs to be added to each test package.
func TestMain(m *testing.M) {
	utils.InitEnv(m)
}

func TestHTTPPromoStep(t *testing.T) {
	utils.TestEnv.Test(t, feature())
}
