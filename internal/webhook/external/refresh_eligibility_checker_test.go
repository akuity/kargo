package external

import (
	"testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_needsRefresh_Git(t *testing.T) {
	for _, test := range []struct {
		name string
		rc   *refreshEligibilityChecker
		rs   kargoapi.RepoSubscription
	}{
		{
			name: "semver selection - invalid semver constraint",
			rc: &refreshEligibilityChecker{
				Git: &struct {
					Tag    *git.TagMetadata
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.rc.needsRefresh(t.Context(), test.rs)
		})
	}
}

func Test_needsRefresh_Image(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "test case 1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
		})
	}
}

func Test_needsRefresh_Chart(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{
			name: "test case 1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
		})
	}
}
