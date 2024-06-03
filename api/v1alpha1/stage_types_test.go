package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestFreightReferenceStackPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         FreightReferenceStack
		newFreight    []FreightReference
		expectedStack FreightReferenceStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newFreight:    []FreightReference{{Name: "foo"}, {Name: "bar"}},
			expectedStack: FreightReferenceStack{{Name: "foo"}, {Name: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         FreightReferenceStack{{Name: "foo"}},
			newFreight:    []FreightReference{{Name: "bar"}},
			expectedStack: FreightReferenceStack{{Name: "bar"}, {Name: "foo"}},
		},
		{
			name:       "initial stack has matching names",
			stack:      FreightReferenceStack{{Name: "foo"}, {Name: "bar"}},
			newFreight: []FreightReference{{Name: "bar"}, {Name: "baz"}, {Name: "zab"}},
			expectedStack: FreightReferenceStack{
				{Name: "bar"},
				{Name: "baz"},
				{Name: "zab"},
				{Name: "foo"},
				{Name: "bar"},
			},
		},
		{
			name: "initial stack is full",
			stack: FreightReferenceStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newFreight: []FreightReference{{Name: "foo"}},
			expectedStack: FreightReferenceStack{
				{Name: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.stack.Push(testCase.newFreight...)
			require.Equal(t, testCase.expectedStack, testCase.stack)
		})
	}
}

func TestFreightReferenceStackUpdateOrPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         FreightReferenceStack
		newFreight    []FreightReference
		expectedStack FreightReferenceStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newFreight:    []FreightReference{{Name: "foo"}, {Name: "bar"}},
			expectedStack: FreightReferenceStack{{Name: "foo"}, {Name: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         FreightReferenceStack{{Name: "foo"}},
			newFreight:    []FreightReference{{Name: "bar"}},
			expectedStack: FreightReferenceStack{{Name: "bar"}, {Name: "foo"}},
		},
		{
			name:       "initial stack has matching names",
			stack:      FreightReferenceStack{{Name: "foo"}, {Name: "bar"}},
			newFreight: []FreightReference{{Name: "bar", Warehouse: "update"}, {Name: "baz"}, {Name: "zab"}},
			expectedStack: FreightReferenceStack{
				{Name: "baz"},
				{Name: "zab"},
				{Name: "foo"},
				{Name: "bar", Warehouse: "update"},
			},
		},
		{
			name: "initial stack is full",
			stack: FreightReferenceStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newFreight: []FreightReference{{Name: "foo"}},
			expectedStack: FreightReferenceStack{
				{Name: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.stack.UpdateOrPush(testCase.newFreight...)
			require.Equal(t, testCase.expectedStack, testCase.stack)
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
