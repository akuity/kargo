package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStage_IsFreightAvailable(t *testing.T) {
	const testNamespace = "fake-namespace"
	const testWarehouse = "fake-warehouse"
	const testStage = "fake-stage"
	const testFreight = "fake-freight"
	testStageMeta := metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testStage,
	}
	testFreightMeta := metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      testFreight,
	}
	testOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: testWarehouse,
	}

	testCases := []struct {
		name     string
		stage    *Stage
		freight  *Freight
		expected bool
	}{
		{
			name:     "stage is nil",
			freight:  &Freight{ObjectMeta: testFreightMeta},
			expected: false,
		},
		{
			name:     "freight is nil",
			stage:    &Stage{ObjectMeta: testStageMeta},
			expected: false,
		},
		{
			name:  "stage and freight are in different namespaces",
			stage: &Stage{ObjectMeta: testStageMeta},
			freight: &Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: "wrong-namespace"},
			},
			expected: false,
		},
		{
			name:  "freight is approved for stage",
			stage: &Stage{ObjectMeta: testStageMeta},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Status: FreightStatus{
					ApprovedFor: map[string]ApprovedStage{
						testStage: {},
					},
				},
			},
			expected: true,
		},
		{
			name: "stage accepts freight direct from origin",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Direct: true,
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream; soak not required",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight is verified in an upstream stage with no longestCompletedSoak; soak required",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: false,
		},
		{
			name: "freight is verified in an upstream stage with longestCompletedSoak; soak required but not elapsed",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: 2 * time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {
							LongestCompletedSoak: &metav1.Duration{Duration: time.Hour},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "freight is verified in an upstream stage with longestCompletedSoak; soak required and is elapsed",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages:           []string{"upstream-stage"},
							RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin:     testOrigin,
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {
							LongestCompletedSoak: &metav1.Duration{Duration: time.Hour},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "freight from origin not requested",
			stage: &Stage{
				ObjectMeta: testStageMeta,
				Spec: StageSpec{
					RequestedFreight: []FreightRequest{{
						Origin: testOrigin,
						Sources: FreightSources{
							Stages: []string{"upstream-stage"},
						},
					}},
				},
			},
			freight: &Freight{
				ObjectMeta: testFreightMeta,
				Origin: FreightOrigin{
					Kind: FreightOriginKindWarehouse,
					Name: "wrong-warehouse",
				},
				Status: FreightStatus{
					VerifiedIn: map[string]VerifiedStage{
						"upstream-stage": {},
					},
				},
			},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				testCase.stage.IsFreightAvailable(testCase.freight),
			)
		})
	}
}

func TestVerificationInfo_HasAnalysisRun(t *testing.T) {
	testCases := []struct {
		name           string
		info           *VerificationInfo
		expectedResult bool
	}{
		{
			name:           "VerificationInfo is nil",
			info:           nil,
			expectedResult: false,
		},
		{
			name:           "AnalysisRun is nil",
			info:           &VerificationInfo{},
			expectedResult: false,
		},
		{
			name: "AnalysisRun is not nil",
			info: &VerificationInfo{
				AnalysisRun: &AnalysisRunReference{},
			},
			expectedResult: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.info.HasAnalysisRun())
		})
	}
}

func TestFreightCollectionIncludes(t *testing.T) {
	const testFreight = "test-freight"
	testCases := []struct {
		name       string
		collection *FreightCollection
		expected   bool
	}{
		{
			name:       "collection is nil",
			collection: nil,
			expected:   false,
		},
		{
			name:       "collection.Freight is nil",
			collection: &FreightCollection{},
			expected:   false,
		},
		{
			name: "collection does not include Freight",
			collection: &FreightCollection{
				Freight: map[string]FreightReference{
					"fake-warehouse":         {Name: "wrong-freight"},
					"another-fake-warehouse": {Name: "another-wrong-freight"},
				},
			},
			expected: false,
		},
		{
			name: "collection includes Freight",
			collection: &FreightCollection{
				Freight: map[string]FreightReference{
					"fake-warehouse": {Name: testFreight},
				},
			},
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, testCase.collection.Includes(testFreight))
		})
	}
}

func TestFreightCollectionUpdateOrPush(t *testing.T) {
	fooOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "foo",
	}
	barOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "bar",
	}
	bazOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "baz",
	}
	testCases := []struct {
		name            string
		freight         map[string]FreightReference
		newFreight      []FreightReference
		expectedFreight map[string]FreightReference
	}{
		{
			name:    "initial list is nil",
			freight: nil,
			newFreight: []FreightReference{
				{Origin: fooOrigin},
				{Origin: bazOrigin},
			},
			expectedFreight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
				bazOrigin.String(): {Origin: bazOrigin},
			},
		},
		{
			name: "update existing FreightReference from same Warehouse",
			freight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
				barOrigin.String(): {Origin: barOrigin},
			},
			newFreight: []FreightReference{
				{Origin: fooOrigin},
				{Origin: barOrigin, Name: "update"},
			},
			expectedFreight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
				barOrigin.String(): {Origin: barOrigin, Name: "update"},
			},
		},
		{
			name: "append new FreightReference",
			freight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
			},
			newFreight: []FreightReference{
				{Origin: barOrigin},
				{Origin: bazOrigin},
			},
			expectedFreight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
				barOrigin.String(): {Origin: barOrigin},
				bazOrigin.String(): {Origin: bazOrigin},
			},
		},
		{
			name: "update existing FreightReference and append new FreightReference",
			freight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin},
				barOrigin.String(): {Origin: barOrigin},
			},
			newFreight: []FreightReference{
				{Origin: fooOrigin, Name: "update"},
				{Origin: bazOrigin},
			},
			expectedFreight: map[string]FreightReference{
				fooOrigin.String(): {Origin: fooOrigin, Name: "update"},
				barOrigin.String(): {Origin: barOrigin},
				bazOrigin.String(): {Origin: bazOrigin},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			entry := &FreightCollection{Freight: testCase.freight}
			entry.UpdateOrPush(testCase.newFreight...)
			require.Equal(t, testCase.expectedFreight, entry.Freight)
		})
	}
}

func TestFreightCollectionReferences(t *testing.T) {
	fooOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "foo",
	}
	barOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "bar",
	}
	bazOrigin := FreightOrigin{
		Kind: FreightOriginKindWarehouse,
		Name: "baz",
	}

	testCases := []struct {
		name           string
		freight        FreightCollection
		expectedResult []FreightReference
	}{
		{
			name: "freight is nil",
			freight: FreightCollection{
				Freight: nil,
			},
			expectedResult: nil,
		},
		{
			name: "freight is empty",
			freight: FreightCollection{
				Freight: map[string]FreightReference{},
			},
		},
		{
			name: "freight has one element",
			freight: FreightCollection{
				Freight: map[string]FreightReference{
					fooOrigin.String(): {Origin: fooOrigin},
				},
			},
			expectedResult: []FreightReference{{Origin: fooOrigin}},
		},
		{
			name: "freight has multiple elements",
			freight: FreightCollection{
				Freight: map[string]FreightReference{
					fooOrigin.String(): {Origin: fooOrigin},
					barOrigin.String(): {Origin: barOrigin},
					bazOrigin.String(): {Origin: bazOrigin},
				},
			},
			expectedResult: []FreightReference{
				{Origin: barOrigin},
				{Origin: bazOrigin},
				{Origin: fooOrigin},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Run the test multiple times to ensure the result is consistent.
			for i := 0; i < 100; i++ {
				require.Equal(t, testCase.expectedResult, testCase.freight.References())
			}
		})
	}
}

func TestFreightHistoryCurrent(t *testing.T) {
	testCases := []struct {
		name           string
		history        FreightHistory
		expectedResult *FreightCollection
	}{
		{
			name:           "history is nil",
			history:        nil,
			expectedResult: nil,
		},
		{
			name:           "history is empty",
			history:        FreightHistory{},
			expectedResult: nil,
		},
		{
			name: "history has one element",
			history: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
			},
			expectedResult: &FreightCollection{
				Freight: map[string]FreightReference{
					"foo": {
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: "foo",
						},
					},
				},
			},
		},
		{
			name: "history has multiple elements",
			history: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"baz": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "baz",
							},
						},
					},
				},
				{
					Freight: map[string]FreightReference{
						"bar": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "bar",
							},
						},
					},
				},
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
			},
			expectedResult: &FreightCollection{
				Freight: map[string]FreightReference{
					"baz": {
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: "baz",
						},
					},
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.history.Current())
		})
	}
}

func TestFreightHistoryRecord(t *testing.T) {
	testCases := []struct {
		name            string
		history         FreightHistory
		newEntry        FreightCollection
		expectedHistory FreightHistory
	}{
		{
			name:    "initial history is nil",
			history: nil,
			newEntry: FreightCollection{
				Freight: map[string]FreightReference{
					"foo": {
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: "foo",
						},
					},
				},
			},
			expectedHistory: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
			},
		},
		{
			name: "initial history is not nil",
			history: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
			},
			newEntry: FreightCollection{
				Freight: map[string]FreightReference{
					"bar": {
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: "bar",
						},
					},
				},
			},
			expectedHistory: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"bar": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "bar",
							},
						},
					},
				},
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
			},
		},
		{
			name: "initial history is full",
			history: FreightHistory{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newEntry: FreightCollection{
				Freight: map[string]FreightReference{
					"foo": {
						Origin: FreightOrigin{
							Kind: FreightOriginKindWarehouse,
							Name: "foo",
						},
					},
				},
			},
			expectedHistory: FreightHistory{
				{
					Freight: map[string]FreightReference{
						"foo": {
							Origin: FreightOrigin{
								Kind: FreightOriginKindWarehouse,
								Name: "foo",
							},
						},
					},
				},
				{}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.history.Record(testCase.newEntry.DeepCopy())
			require.Equal(t, testCase.expectedHistory, testCase.history)
		})
	}
}

func TestVerificationInfoStack_Current(t *testing.T) {
	testCases := []struct {
		name           string
		stack          VerificationInfoStack
		expectedResult *VerificationInfo
	}{
		{
			name:           "stack is nil",
			stack:          nil,
			expectedResult: nil,
		},
		{
			name:           "stack is empty",
			stack:          VerificationInfoStack{},
			expectedResult: nil,
		},
		{
			name: "stack has one element",
			stack: VerificationInfoStack{
				{ID: "foo"},
			},
			expectedResult: &VerificationInfo{ID: "foo"},
		},
		{
			name: "stack has multiple elements",
			stack: VerificationInfoStack{
				{ID: "foo"},
				{ID: "bar"},
				{ID: "baz"},
			},
			expectedResult: &VerificationInfo{ID: "foo"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.stack.Current())
		})
	}
}

func TestVerificationInfoStack_UpdateOrPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         VerificationInfoStack
		newInfo       []VerificationInfo
		expectedStack VerificationInfoStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newInfo:       []VerificationInfo{{ID: "foo"}, {ID: "bar"}},
			expectedStack: VerificationInfoStack{{ID: "foo"}, {ID: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         VerificationInfoStack{{ID: "foo"}},
			newInfo:       []VerificationInfo{{ID: "bar"}, {ID: "baz"}},
			expectedStack: VerificationInfoStack{{ID: "bar"}, {ID: "baz"}, {ID: "foo"}},
		},
		{
			name:    "initial stack has matching IDs",
			stack:   VerificationInfoStack{{ID: "foo"}, {ID: "bar"}},
			newInfo: []VerificationInfo{{ID: "bar", Phase: VerificationPhaseFailed}, {ID: "baz"}, {ID: "zab"}},
			expectedStack: VerificationInfoStack{
				{ID: "baz"},
				{ID: "zab"},
				{ID: "foo"},
				{ID: "bar", Phase: VerificationPhaseFailed},
			},
		},
		{
			name: "initial stack is full",
			stack: VerificationInfoStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newInfo: []VerificationInfo{{ID: "foo"}},
			expectedStack: VerificationInfoStack{
				{ID: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.stack.UpdateOrPush(testCase.newInfo...)
			require.Equal(t, testCase.expectedStack, testCase.stack)
		})
	}
}

func TestImageDeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *Image
		b              *Image
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &Image{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &Image{},
			expectedResult: false,
		},
		{
			name: "repo URLs differ",
			a: &Image{
				RepoURL: "foo",
			},
			b: &Image{
				RepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "git repo URLs differ",
			a: &Image{
				RepoURL:    "fake-url",
				GitRepoURL: "foo",
			},
			b: &Image{
				RepoURL:    "fake-url",
				GitRepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "image tags differ",
			a: &Image{
				RepoURL: "fake-url",
				Tag:     "foo",
			},
			b: &Image{
				RepoURL: "fake-url",
				Tag:     "bar",
			},
			expectedResult: false,
		},
		{
			name: "image digests differ",
			a: &Image{
				RepoURL: "fake-url",
				Digest:  "foo",
			},
			b: &Image{
				RepoURL: "fake-url",
				Digest:  "bar",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			a: &Image{
				RepoURL:    "fake-url",
				GitRepoURL: "fake-repo-url",
				Tag:        "fake-tag",
				Digest:     "fake-digest",
			},
			b: &Image{
				RepoURL:    "fake-url",
				GitRepoURL: "fake-repo-url",
				Tag:        "fake-tag",
				Digest:     "fake-digest",
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

func TestChartDeepEquals(t *testing.T) {
	testCases := []struct {
		name           string
		a              *Chart
		b              *Chart
		expectedResult bool
	}{
		{
			name:           "a and b both nil",
			expectedResult: true,
		},
		{
			name:           "only a is nil",
			b:              &Chart{},
			expectedResult: false,
		},
		{
			name:           "only b is nil",
			a:              &Chart{},
			expectedResult: false,
		},
		{
			name: "repo URLs differ",
			a: &Chart{
				RepoURL: "foo",
			},
			b: &Chart{
				RepoURL: "bar",
			},
			expectedResult: false,
		},
		{
			name: "chart names differ",
			a: &Chart{
				RepoURL: "fake-url",
				Name:    "foo",
			},
			b: &Chart{
				RepoURL: "fake-url",
				Name:    "bar",
			},
			expectedResult: false,
		},
		{
			name: "chart versions differ",
			a: &Chart{
				RepoURL: "fake-url",
				Name:    "fake-name",
				Version: "v1.0.0",
			},
			b: &Chart{
				RepoURL: "fake-url",
				Name:    "fake-name",
				Version: "v2.0.0",
			},
			expectedResult: false,
		},
		{
			name: "perfect match",
			a: &Chart{
				RepoURL: "fake-url",
				Name:    "fake-name",
				Version: "v1.0.0",
			},
			b: &Chart{
				RepoURL: "fake-url",
				Name:    "fake-name",
				Version: "v1.0.0",
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
