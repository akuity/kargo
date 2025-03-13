package builtin

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/internal/credentials"
)

// Initialize registers all built-in promotion.StepRunners with the promotion
// package's internal StepRunner registry.
func Initialize(kargoClient, argocdClient client.Client, credsDB credentials.Database) {
	builtIns := []promotion.StepRunner{
		newArgocdUpdater(argocdClient),
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
