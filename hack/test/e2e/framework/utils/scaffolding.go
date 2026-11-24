package utils

import (
	"log"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/akuity/kargo/hack/test/e2e/framework/funcsloader"
)

var (
	TestEnv env.Environment
)

func InitEnv(m *testing.M) {

	setup, finish := funcsloader.GetFuncs()

	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("envconf failed: %s", err)
	}

	TestEnv = env.NewWithConfig(cfg)

	// Run setup functions loaded from test env
	TestEnv.Setup(setup...)

	// Run teardown functions loaded from test env
	TestEnv.Finish(finish...)

	// Use Environment.Run to launch the test
	os.Exit(TestEnv.Run(m))
}
