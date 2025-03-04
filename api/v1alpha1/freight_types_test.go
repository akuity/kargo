package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGitCommitDeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *GitCommit
		b              *GitCommit
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoURLs differ",
			a: &GitCommit{
				RepoURL: "foo",
			},
			b: &GitCommit{
				RepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "branch names differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Branch:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "tags differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Tag:     "bar",
			},
			expectedResult: false,
		},
		{
			name: "messages differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Message: "bar",
			},
			expectedResult: false,
		},
		{
			name: "authors differ",
			a: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "foo",
			},
			b: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
				Author:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "committers differ",
			a: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "foo",
			},
			b: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Committer: "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			a: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Branch:    "fake-branch",
				Tag:       "fake-tag",
				Message:   "fake-message",
				Author:    "fake-author",
				Committer: "fake-committer",
			},
			b: &GitCommit{
				RepoURL:   "fake-url",
				ID:        "fake-commit-id",
				Branch:    "fake-branch",
				Tag:       "fake-tag",
				Message:   "fake-message",
				Author:    "fake-author",
				Committer: "fake-committer",
			},
			expectedResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.a.DeepEquals(testCase.b))
			require.Equal(t, testCase.expectedResult, testCase.b.DeepEquals(testCase.a))
		})
	}
}

func TestGitCommitEquals(t *testing.T) {
	testCases := []struct {
		name           string
		lhs            *GitCommit
		rhs            *GitCommit
		expectedResult bool
	}{
		{
			name:           "lhs and rhs both nil",
			expectedResult: true,
		},
		{
			name:           "only lhs is nil",
			rhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name:           "only rhs is nil",
			lhs:            &GitCommit{},
			expectedResult: false,
		},
		{
			name: "repoUrls differ",
			lhs: &GitCommit{
				RepoURL: "foo",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "bar",
				ID:      "fake-commit-id",
			},
			expectedResult: false,
		},
		{
			name: "commit IDs differ",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "foo",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			lhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			rhs: &GitCommit{
				RepoURL: "fake-url",
				ID:      "fake-commit-id",
			},
			expectedResult: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedResult,
				testCase.lhs.Equals(testCase.rhs),
			)
		})
	}
}

func TestFreightGenerateID(t *testing.T) {
	freight := Freight{
		Origin: FreightOrigin{
			Kind: "fake-kind",
			Name: "fake-name",
		},
		Commits: []GitCommit{
			{
				RepoURL: "fake-git-repo",
				ID:      "fake-commit-id",
			},
		},
		Images: []Image{
			{
				RepoURL: "fake-image-repo",
				Tag:     "fake-image-tag",
			},
		},
		Charts: []Chart{
			{
				RepoURL: "fake-chart-repo",
				Name:    "fake-chart",
				Version: "fake-chart-version",
			},
		},
	}
	id := freight.GenerateID()
	expected := id
	// Doing this any number of times should yield the same ID
	for i := 0; i < 100; i++ {
		require.Equal(t, expected, freight.GenerateID())
	}
	// Changing anything should change the result
	freight.Commits[0].ID = "a-different-fake-commit"
	require.NotEqual(t, expected, freight.GenerateID())
}

func TestFreight_IsCurrentlyIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsCurrentlyIn(testStage))
	freight.Status.CurrentlyIn = map[string]CurrentStage{testStage: {}}
	require.True(t, freight.IsCurrentlyIn(testStage))
}

func TestFreight_IsVerifiedIn(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsVerifiedIn(testStage))
	freight.Status.VerifiedIn = map[string]VerifiedStage{testStage: {}}
	require.True(t, freight.IsVerifiedIn(testStage))
}

func TestFreight_IsApprovedFor(t *testing.T) {
	const testStage = "fake-stage"
	freight := &Freight{}
	require.False(t, freight.IsApprovedFor(testStage))
	freight.Status.ApprovedFor = map[string]ApprovedStage{testStage: {}}
	require.True(t, freight.IsApprovedFor(testStage))
}

func TestFreight_GetLongestSoak(t *testing.T) {
	testStage := "fake-stage"
	testCases := []struct {
		name       string
		status     FreightStatus
		assertions func(t *testing.T, status FreightStatus, longestSoak time.Duration)
	}{
		{
			name: "Freight is not currently in the Stage and was never verified there",
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is not currently in the Stage but was verified there",
			status: FreightStatus{
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Equal(t, time.Hour, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage but was never verified there",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				require.Zero(t, longestSoak)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; current soak is longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
		{
			name: "Freight is currently in the Stage and has been verified there; a previous soak was longer",
			status: FreightStatus{
				CurrentlyIn: map[string]CurrentStage{
					testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
				},
				VerifiedIn: map[string]VerifiedStage{
					testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
				},
			},
			assertions: func(t *testing.T, _ FreightStatus, longestSoak time.Duration) {
				// Expect these to be equal within a second. TODO(krancour): There's probably a
				// more elegant way to do this, but I consider good enough.
				require.GreaterOrEqual(t, longestSoak, 2*time.Hour)
				require.LessOrEqual(t, longestSoak, 2*time.Hour+time.Second)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight := &Freight{
				Status: testCase.status,
			}
			testCase.assertions(t, freight.Status, freight.GetLongestSoak(testStage))
		})
	}
}

func TestFreightStatus_AddCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already in current", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddCurrentStage(testStage, newTime)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, oldTime, record.Since.Time)
	})
	t.Run("not already in current", func(t *testing.T) {
		status := FreightStatus{}
		status.AddCurrentStage(testStage, now)
		require.NotNil(t, status.CurrentlyIn)
		record, in := status.CurrentlyIn[testStage]
		require.True(t, in)
		require.Equal(t, now, record.Since.Time)
	})
}

func TestFreightStatus_RemoveCurrentStage(t *testing.T) {
	const testStage = "fake-stage"
	t.Run("not verified", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
	})
	t.Run("verified; old soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, 2*time.Hour, record.LongestCompletedSoak.Duration)
	})
	t.Run("verified; new soak is longer", func(t *testing.T) {
		status := FreightStatus{
			CurrentlyIn: map[string]CurrentStage{
				testStage: {Since: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}},
			},
			VerifiedIn: map[string]VerifiedStage{
				testStage: {LongestCompletedSoak: &metav1.Duration{Duration: time.Hour}},
			},
		}
		status.RemoveCurrentStage(testStage)
		require.NotContains(t, status.CurrentlyIn, testStage)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		// Expect these to be equal within a second. TODO(krancour): There's probably a
		// more elegant way to do this, but I consider good enough.
		require.GreaterOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour)
		require.LessOrEqual(t, record.LongestCompletedSoak.Duration, 2*time.Hour+time.Second)
	})
}

func TestFreightStatus_AddVerifiedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already verified", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			VerifiedIn: map[string]VerifiedStage{
				testStage: {VerifiedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddVerifiedStage(testStage, newTime)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, oldTime, record.VerifiedAt.Time)
	})
	t.Run("not already verified", func(t *testing.T) {
		status := FreightStatus{}
		testTime := time.Now()
		status.AddVerifiedStage(testStage, testTime)
		require.NotNil(t, status.VerifiedIn)
		record, verified := status.VerifiedIn[testStage]
		require.True(t, verified)
		require.Equal(t, testTime, record.VerifiedAt.Time)
	})
}

func TestFreightStatus_AddApprovedStage(t *testing.T) {
	const testStage = "fake-stage"
	now := time.Now()
	t.Run("already approved", func(t *testing.T) {
		oldTime := now.Add(-time.Hour)
		newTime := now
		status := FreightStatus{
			ApprovedFor: map[string]ApprovedStage{
				testStage: {ApprovedAt: &metav1.Time{Time: oldTime}},
			},
		}
		status.AddApprovedStage(testStage, newTime)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, oldTime, record.ApprovedAt.Time)
	})
	t.Run("not already approved", func(t *testing.T) {
		status := FreightStatus{}
		status.AddApprovedStage(testStage, now)
		require.NotNil(t, status.ApprovedFor)
		record, approved := status.ApprovedFor[testStage]
		require.True(t, approved)
		require.Equal(t, now, record.ApprovedAt.Time)
	})
}
