package external

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libGit "github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/logging"
)

func Test_needsRefresh_Git(t *testing.T) {
	for _, test := range []struct {
		name         string
		rc           *refreshEligibilityChecker
		rs           kargoapi.RepoSubscription
		repoURLs     []string
		needsRefresh bool
	}{
		{
			name:         "refresh checker code change unset",
			repoURLs:     []string{"https://github.com/username/repo.git"},
			rc:           &refreshEligibilityChecker{git: nil},
			rs:           kargoapi.RepoSubscription{Git: nil},
			needsRefresh: false,
		},
		{
			name:     "semver - invalid semver constraint",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{Tag: "v1.0.0"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
					// validation is optional for warehouse semver constraints
					// so we have to consider an invalid input.
					SemverConstraint: "invalid-semver-constraint",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - tag is not semver formatted",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{Tag: "not-semver-tag"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
					SemverConstraint:        "^v1.0.0",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - not matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{Tag: "v1.2.3"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
					SemverConstraint:        "^v2.2.3",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{Tag: "v1.2.3"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
					SemverConstraint:        "^v1.0.0",
				},
			},
			needsRefresh: true,
		},
		{
			name:     "newest from branch - not matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{branch: "release-1.0"},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				},
			},
			needsRefresh: false,
		},
		{
			name:     "newest from branch - matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					branch: "main",
					tag:    &libGit.TagMetadata{Tag: "v1.0.0"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				},
			},
			needsRefresh: true,
		},
		{
			name:     "lexical - not matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					branch: "main",
					tag:    &libGit.TagMetadata{Tag: "v1.0.0"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
					AllowTags:               "^nightly-\\d{8}$",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "lexical - matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					branch: "main",
					tag:    &libGit.TagMetadata{Tag: "nightly-20231001"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
					AllowTags:               `^nightly-\d{8}$`,
				},
			},
			needsRefresh: true,
		},
		// From this point on, we are testing the newet tag strategy.
		// In this context(webhooks) we are always working with the newest tag,
		// so from here on out we will largely be testing the base filters.
		// This includes mostly path filters and expressions; since allow/ignore
		// rules were already tested above with the lexical strategy.
		{
			name:     "newest tag - path filters not matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag:   &libGit.TagMetadata{Tag: "v1.0.0"},
					diffs: []string{"src/file.txt"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
					IncludePaths:            []string{"docs/"},
				},
			},
			needsRefresh: false,
		},
		{
			name:     "newest tag - path filters matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag:   &libGit.TagMetadata{Tag: "v1.0.0"},
					diffs: []string{"docs/file.txt"},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
					IncludePaths:            []string{"docs/"},
				},
			},
			needsRefresh: true,
		},
		{
			name:     "newest tag - expression filters not matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{
						Tag:         "v1.0.0",
						CreatorDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
					ExpressionFilter:        `creatorDate.After(date('2024-01-01'))`,
				},
			},
			needsRefresh: false,
		},
		{
			name:     "newest tag - expression filters matching",
			repoURLs: []string{"https://github.com/username/repo.git"},
			rc: &refreshEligibilityChecker{
				git: &codeChange{
					tag: &libGit.TagMetadata{
						Tag:         "v1.0.0",
						CreatorDate: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			rs: kargoapi.RepoSubscription{
				Git: &kargoapi.GitSubscription{
					RepoURL:                 "https://github.com/username/repo.git",
					Branch:                  "main",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
					ExpressionFilter:        `creatorDate.After(date('2024-01-01'))`,
				},
			},
			needsRefresh: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.ContextWithLogger(t.Context(),
				logging.NewLogger(logging.DebugLevel),
			)
			require.Equal(t, test.needsRefresh,
				test.rc.needsRefresh(ctx, []kargoapi.RepoSubscription{test.rs}, test.repoURLs...),
			)
		})
	}
}

func Test_needsRefresh_Image(t *testing.T) {
	for _, test := range []struct {
		name         string
		rc           *refreshEligibilityChecker
		rs           kargoapi.RepoSubscription
		repoURLs     []string
		needsRefresh bool
	}{
		{
			name:         "refresh checker image change unset",
			repoURLs:     []string{"testregistry.io/hello-world"},
			rc:           &refreshEligibilityChecker{newImageTag: nil},
			rs:           kargoapi.RepoSubscription{Image: nil},
			needsRefresh: false,
		},
		{
			name:     "lexical - not matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
					AllowTags:              "^nightly-\\d{8}$",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "lexical - matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("nightly-20231001"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategyLexical,
					AllowTags:              `^nightly-\d{8}$`,
				},
			},
			needsRefresh: true,
		},
		{
			name:     "semver - invalid semver constraint",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
					// validation is optional for warehouse semver constraints
					// so we have to consider an invalid input.
					SemverConstraint: "invalid-semver-constraint",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - tag is not semver formatted",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("invalid-semver-tag"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
					SemverConstraint:       "^v1.0.0",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - not matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
					SemverConstraint:       "^v2.2.3",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
					SemverConstraint:       "^v1.0.0",
				},
			},
			needsRefresh: true,
		},
		{
			name:     "newest build - matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategyNewestBuild,
					SemverConstraint:       "^v1.0.0",
				},
			},
			needsRefresh: true,
		},
		{
			name:     "digest - not matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
					SemverConstraint:       "^v1.0.0",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "digest - matching",
			repoURLs: []string{"testregistry.io/hello-world"},
			rc: &refreshEligibilityChecker{
				newImageTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Image: &kargoapi.ImageSubscription{
					RepoURL:                "testregistry.io/hello-world",
					ImageSelectionStrategy: kargoapi.ImageSelectionStrategyDigest,
					SemverConstraint:       "latest",
				},
			},
			needsRefresh: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.ContextWithLogger(t.Context(),
				logging.NewLogger(logging.DebugLevel),
			)
			require.Equal(t, test.needsRefresh,
				test.rc.needsRefresh(ctx, []kargoapi.RepoSubscription{test.rs}, test.repoURLs...),
			)
		})
	}
}

func Test_needsRefresh_Chart(t *testing.T) {
	for _, test := range []struct {
		name         string
		rc           *refreshEligibilityChecker
		rs           kargoapi.RepoSubscription
		repoURLs     []string
		needsRefresh bool
	}{
		{
			name:         "refresh checker chart change unset",
			repoURLs:     []string{"https://charts.example.com/hello-world"},
			rc:           &refreshEligibilityChecker{newChartTag: nil},
			rs:           kargoapi.RepoSubscription{Chart: nil},
			needsRefresh: false,
		},
		{
			name:     "semver - unset",
			repoURLs: []string{"https://charts.example.com/hello-world"},
			rc: &refreshEligibilityChecker{
				newChartTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{
					RepoURL:          "https://charts.example.com/hello-world",
					SemverConstraint: "",
				},
			},
			needsRefresh: true,
		},
		{
			name:     "semver - invalid semver constraint",
			repoURLs: []string{"https://charts.example.com/hello-world"},
			rc: &refreshEligibilityChecker{
				newChartTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{
					RepoURL: "https://charts.example.com/hello-world",
					// validation is optional for warehouse semver constraints
					// so we have to consider an invalid input.
					SemverConstraint: "invalid-semver-constraint",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - tag is not semver formatted",
			repoURLs: []string{"https://charts.example.com/hello-world"},
			rc: &refreshEligibilityChecker{
				newChartTag: strPtr("invalid-semver-tag"),
			},
			rs: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{
					RepoURL:          "https://charts.example.com/hello-world",
					SemverConstraint: "^v1.0.0",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - not matching",
			repoURLs: []string{"https://charts.example.com/hello-world"},
			rc: &refreshEligibilityChecker{
				newChartTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{
					RepoURL:          "https://charts.example.com/hello-world",
					SemverConstraint: "^v2.2.3",
				},
			},
			needsRefresh: false,
		},
		{
			name:     "semver - matching",
			repoURLs: []string{"https://charts.example.com/hello-world"},
			rc: &refreshEligibilityChecker{
				newChartTag: strPtr("v1.0.0"),
			},
			rs: kargoapi.RepoSubscription{
				Chart: &kargoapi.ChartSubscription{
					RepoURL:          "https://charts.example.com/hello-world",
					SemverConstraint: "^v1.0.0",
				},
			},
			needsRefresh: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.ContextWithLogger(t.Context(),
				logging.NewLogger(logging.DebugLevel),
			)
			require.Equal(t, test.needsRefresh,
				test.rc.needsRefresh(ctx, []kargoapi.RepoSubscription{test.rs}, test.repoURLs...),
			)
		})
	}
}
