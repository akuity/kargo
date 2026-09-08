package fidelity

import (
	"testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	gen "github.com/akuity/kargo/pkg/x/client/generated"
)

func TestReleaseContextConfigRoundTrip(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		config *kargoapi.ReleaseContextConfig
	}{
		{name: "omitted"},
		{name: "explicitly empty", config: &kargoapi.ReleaseContextConfig{}},
		{
			name: "custom mappings",
			config: &kargoapi.ReleaseContextConfig{
				ImageAnnotations: kargoapi.ImageAnnotationMappings{
					CommitSubject:   "com.example.image.commit.subject",
					CommitAuthor:    "com.example.image.commit.author",
					CommitCommitter: "com.example.image.commit.committer",
					CommitCreatedAt: "com.example.image.commit.created",
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			roundTrip(
				t,
				kargoapi.ClusterConfigSpec{ReleaseContext: testCase.config},
				&gen.ClusterConfigSpec{},
			)
			roundTrip(
				t,
				kargoapi.ProjectConfigSpec{ReleaseContext: testCase.config},
				&gen.ProjectConfigSpec{},
			)
		})
	}
}
