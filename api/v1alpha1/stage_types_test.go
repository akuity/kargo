package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleFreightStackEmpty(t *testing.T) {
	testCases := []struct {
		name           string
		stack          SimpleFreightStack
		expectedResult bool
	}{
		{
			name:           "stack is nil",
			stack:          nil,
			expectedResult: true,
		},
		{
			name:           "stack is empty",
			stack:          SimpleFreightStack{},
			expectedResult: true,
		},
		{
			name:           "stack has items",
			stack:          SimpleFreightStack{{ID: "foo"}},
			expectedResult: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expectedResult, testCase.stack.Empty())
		})
	}
}

func TestSimpleFreightStackPop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           SimpleFreightStack
		expectedStack   SimpleFreightStack
		expectedFreight SimpleFreight
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedStack:   nil,
			expectedFreight: SimpleFreight{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           SimpleFreightStack{},
			expectedStack:   SimpleFreightStack{},
			expectedFreight: SimpleFreight{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           SimpleFreightStack{{ID: "foo"}, {ID: "bar"}},
			expectedStack:   SimpleFreightStack{{ID: "bar"}},
			expectedFreight: SimpleFreight{ID: "foo"},
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

func TestSimpleFreightStackTop(t *testing.T) {
	testCases := []struct {
		name            string
		stack           SimpleFreightStack
		expectedFreight SimpleFreight
		expectedOK      bool
	}{
		{
			name:            "stack is nil",
			stack:           nil,
			expectedFreight: SimpleFreight{},
			expectedOK:      false,
		},
		{
			name:            "stack is empty",
			stack:           SimpleFreightStack{},
			expectedFreight: SimpleFreight{},
			expectedOK:      false,
		},
		{
			name:            "stack has items",
			stack:           SimpleFreightStack{{ID: "foo"}, {ID: "bar"}},
			expectedFreight: SimpleFreight{ID: "foo"},
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

func TestSimpleFreightStackPush(t *testing.T) {
	testCases := []struct {
		name          string
		stack         SimpleFreightStack
		newFreight    []SimpleFreight
		expectedStack SimpleFreightStack
	}{
		{
			name:          "initial stack is nil",
			stack:         nil,
			newFreight:    []SimpleFreight{{ID: "foo"}, {ID: "bar"}},
			expectedStack: SimpleFreightStack{{ID: "foo"}, {ID: "bar"}},
		},
		{
			name:          "initial stack is not nil",
			stack:         SimpleFreightStack{{ID: "foo"}},
			newFreight:    []SimpleFreight{{ID: "bar"}},
			expectedStack: SimpleFreightStack{{ID: "bar"}, {ID: "foo"}},
		},
		{
			name: "initial stack is full",
			stack: SimpleFreightStack{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			},
			newFreight: []SimpleFreight{{ID: "foo"}},
			expectedStack: SimpleFreightStack{
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
