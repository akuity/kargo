package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFreightReferenceStackEmpty(t *testing.T) {
	testCases := []struct {
		name           string
		stack          FreightReferenceStack
		expectedResult bool
	}{
		{
			name:           "stack is nil",
			stack:          nil,
			expectedResult: true,
		},
		{
			name:           "stack is empty",
			stack:          FreightReferenceStack{},
			expectedResult: true,
		},
		{
			name:           "stack has items",
			stack:          FreightReferenceStack{{ID: "foo"}},
			expectedResult: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.stack.Empty())
		})
	}
}

func TestFreightReferenceStackPop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           FreightReferenceStack
		expectedStack   FreightReferenceStack
		expectedFreight FreightReference
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedStack:   nil,
			expectedFreight: FreightReference{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           FreightReferenceStack{},
			expectedStack:   FreightReferenceStack{},
			expectedFreight: FreightReference{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           FreightReferenceStack{{ID: "foo"}, {ID: "bar"}},
			expectedStack:   FreightReferenceStack{{ID: "bar"}},
			expectedFreight: FreightReference{ID: "foo"},
			expectedOK:      true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freight, ok := testCase.stack.Pop()
			require.Equal(t, testCase.expectedStack, testCase.stack)
			require.Equal(t, testCase.expectedFreight, freight)
			require.Equal(t, testCase.expectedOK, ok)
		})
	}
}

func TestFreightReferenceStackTop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           FreightReferenceStack
		expectedFreight FreightReference
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedFreight: FreightReference{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           FreightReferenceStack{},
			expectedFreight: FreightReference{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           FreightReferenceStack{{ID: "foo"}, {ID: "bar"}},
			expectedFreight: FreightReference{ID: "foo"},
			expectedOK:      true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			initialLen := len(testCase.stack)
			freight, ok := testCase.stack.Top()
			require.Len(t, testCase.stack, initialLen)
			require.Equal(t, testCase.expectedFreight, freight)
			require.Equal(t, testCase.expectedOK, ok)
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
			newFreight:    []FreightReference{{ID: "foo"}, {ID: "bar"}},
			expectedStack: FreightReferenceStack{{ID: "foo"}, {ID: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         FreightReferenceStack{{ID: "foo"}},
			newFreight:    []FreightReference{{ID: "bar"}},
			expectedStack: FreightReferenceStack{{ID: "bar"}, {ID: "foo"}},
		},
		{
			name: "initial stack is full",
			stack: FreightReferenceStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newFreight: []FreightReference{{ID: "foo"}},
			expectedStack: FreightReferenceStack{
				{ID: "foo"}, {}, {}, {}, {}, {}, {}, {}, {}, {},
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
