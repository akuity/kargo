package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGenerateFreightID(t *testing.T) {
	freight := kargoapi.Freight{
		Origin: kargoapi.FreightOrigin{
			Kind: "fake-kind",
			Name: "fake-name",
		},
		Commits: []kargoapi.GitCommit{
			{
				RepoURL: "fake-git-repo",
				ID:      "fake-commit-id",
			},
		},
		Images: []kargoapi.Image{
			{
				RepoURL: "fake-image-repo",
				Tag:     "fake-image-tag",
			},
		},
		Charts: []kargoapi.Chart{
			{
				RepoURL: "fake-chart-repo",
				Name:    "fake-chart",
				Version: "fake-chart-version",
			},
		},
	}
	id := GenerateFreightID(&freight)
	expected := id
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		require.Equal(t, expected, GenerateFreightID(&freight))
	}
	// Changing anything should change the result
	freight.Commits[0].ID = "a-different-fake-commit"
	require.NotEqual(t, expected, GenerateFreightID(&freight))
}

// TestSubscriptionNameAffectsFreightID verifies that a non-empty SubscriptionName
// on a GitCommit, Image, or Chart causes a distinct Freight ID to be generated,
// while an empty SubscriptionName preserves backward-compatible byte-identical IDs.
func TestSubscriptionNameAffectsFreightID(t *testing.T) {
	t.Run("git commit SubscriptionName produces distinct ID", func(t *testing.T) {
		base := kargoapi.Freight{
			Commits: []kargoapi.GitCommit{{
				RepoURL: "https://github.com/example/repo",
				ID:      "abc123def456abc123def456abc123def456abc123",
			}},
		}
		named := kargoapi.Freight{
			Commits: []kargoapi.GitCommit{{
				RepoURL:          "https://github.com/example/repo",
				ID:               "abc123def456abc123def456abc123def456abc123",
				SubscriptionName: "my-sub",
			}},
		}
		namedSame := kargoapi.Freight{
			Commits: []kargoapi.GitCommit{{
				RepoURL:          "https://github.com/example/repo",
				ID:               "abc123def456abc123def456abc123def456abc123",
				SubscriptionName: "my-sub",
			}},
		}
		idBase := GenerateFreightID(&base)
		idNamed := GenerateFreightID(&named)
		idNamedSame := GenerateFreightID(&namedSame)
		require.NotEqual(t, idBase, idNamed,
			"non-empty SubscriptionName must produce a distinct Freight ID")
		require.Equal(t, idNamed, idNamedSame,
			"same SubscriptionName must produce the same Freight ID")
	})

	t.Run("image SubscriptionName produces distinct ID", func(t *testing.T) {
		base := kargoapi.Freight{
			Images: []kargoapi.Image{{
				RepoURL: "example.com/myimage",
				Tag:     "v1.2.3",
				Digest:  "sha256:deadbeef",
			}},
		}
		named := kargoapi.Freight{
			Images: []kargoapi.Image{{
				RepoURL:          "example.com/myimage",
				Tag:              "v1.2.3",
				Digest:           "sha256:deadbeef",
				SubscriptionName: "img-sub",
			}},
		}
		idBase := GenerateFreightID(&base)
		idNamed := GenerateFreightID(&named)
		require.NotEqual(t, idBase, idNamed,
			"non-empty image SubscriptionName must produce a distinct Freight ID")
	})

	t.Run("chart SubscriptionName produces distinct ID", func(t *testing.T) {
		base := kargoapi.Freight{
			Charts: []kargoapi.Chart{{
				RepoURL: "https://charts.example.com",
				Name:    "mychart",
				Version: "1.0.0",
			}},
		}
		named := kargoapi.Freight{
			Charts: []kargoapi.Chart{{
				RepoURL:          "https://charts.example.com",
				Name:             "mychart",
				Version:          "1.0.0",
				SubscriptionName: "chart-sub",
			}},
		}
		idBase := GenerateFreightID(&base)
		idNamed := GenerateFreightID(&named)
		require.NotEqual(t, idBase, idNamed,
			"non-empty chart SubscriptionName must produce a distinct Freight ID")
	})
}
