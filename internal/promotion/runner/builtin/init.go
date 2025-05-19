package builtin

import (
	"sync/atomic"
	"time"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/promotion"
	promoPkg "github.com/akuity/kargo/pkg/promotion"
)

var initialized atomic.Uint32

// Initialize registers all built-in promotion.StepRunners with the promotion
// package's internal StepRunner registry.
func Initialize(kargoClient, argocdClient client.Client, credsDB credentials.Database) {
	if !initialized.CompareAndSwap(0, 1) {
		panic("built-in promotion step runners already initialized")
	}
	builtIns := []promoPkg.StepRunner{
		NewRetryableStepRunner(
			newArgocdUpdater(argocdClient),
			ptr.To(5*time.Minute),
			0,
		),
		newHelmChartUpdater(credsDB),
		newFileCopier(),
		newFileDeleter(),
		newGitCloner(credsDB),
		newGitCommitter(),
		newGitPROpener(credsDB),
		newGitPRWaiter(credsDB),
		newGitPusher(credsDB),
		newGitTreeClearer(),
		newHelmTemplateRunner(),
		newHTTPRequester(),
		newJSONParser(),
		newJSONUpdater(),
		newKustomizeBuilder(),
		newKustomizeImageSetter(kargoClient),
		newOutputComposer(),
		newYAMLParser(),
		newYAMLUpdater(),
	}
	for _, builtIn := range builtIns {
		promotion.RegisterStepRunner(builtIn)
	}
}
