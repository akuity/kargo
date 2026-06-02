package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestAuthorizedStages(t *testing.T) {
	testCases := []struct {
		name   string
		value  string
		assert func(*testing.T, []types.NamespacedName, error)
	}{
		{
			name:  "single entry",
			value: "proj:stage",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]types.NamespacedName{{Namespace: "proj", Name: "stage"}},
					stages,
				)
			},
		},
		{
			name:  "multiple entries",
			value: "proj:stage-a,proj:stage-b,other:stage-c",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]types.NamespacedName{
						{Namespace: "proj", Name: "stage-a"},
						{Namespace: "proj", Name: "stage-b"},
						{Namespace: "other", Name: "stage-c"},
					},
					stages,
				)
			},
		},
		{
			name:  "whitespace is trimmed and empty entries are ignored",
			value: " proj : stage-a , , other:stage-b ,",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]types.NamespacedName{
						{Namespace: "proj", Name: "stage-a"},
						{Namespace: "other", Name: "stage-b"},
					},
					stages,
				)
			},
		},
		{
			name:  "missing separator",
			value: "bogus",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "expected format")
				require.Nil(t, stages)
			},
		},
		{
			name:  "empty stage",
			value: "proj:",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "expected format")
				require.Nil(t, stages)
			},
		},
		{
			name:  "empty project",
			value: ":stage",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "expected format")
				require.Nil(t, stages)
			},
		},
		{
			name:  "empty value",
			value: "",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "no authorized Stages")
				require.Nil(t, stages)
			},
		},
		{
			name:  "only commas",
			value: ",, ,",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "no authorized Stages")
				require.Nil(t, stages)
			},
		},
		{
			name:  "wildcard project is rejected",
			value: "*:stage",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "deprecated glob expressions")
				require.Nil(t, stages)
			},
		},
		{
			name:  "wildcard stage is rejected",
			value: "proj:*",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "deprecated glob expressions")
				require.Nil(t, stages)
			},
		},
		{
			name:  "wildcard in one entry of a list is rejected",
			value: "proj:stage,proj:*",
			assert: func(t *testing.T, stages []types.NamespacedName, err error) {
				require.ErrorContains(t, err, "deprecated glob expressions")
				require.Nil(t, stages)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stages, err := AuthorizedStages(testCase.value)
			testCase.assert(t, stages, err)
		})
	}
}
