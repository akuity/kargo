package promotion

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// newGenericGitMechanism returns a gitMechanism that only only selects and
// performs updates that do not involve any configuration management tools.
func newGenericGitMechanism(
	cl client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"generic Git promotion mechanism",
		cl,
		credentialsDB,
		selectGenericGitUpdates,
		nil,
	)
}

// selectGenericGitUpdates returns a subset of the given updates that do not
// involve any configuration management tools.
func selectGenericGitUpdates(updates []kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
	selectedUpdates := make([]kargoapi.GitRepoUpdate, 0, len(updates))
	for _, update := range updates {
		if update.Kustomize == nil &&
			update.Helm == nil &&
			update.Render == nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}
