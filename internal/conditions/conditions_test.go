package conditions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGet(t *testing.T) {
	const (
		mockType1 = "MockType1"
		mockType2 = "MockType2"
	)

	tests := []struct {
		name          string
		getter        Getter
		conditionType string
		want          *metav1.Condition
	}{
		{
			name:          "nil getter",
			getter:        nil,
			conditionType: mockType1,
			want:          nil,
		},
		{
			name: "empty conditions",
			getter: &mockSetter{
				conditions: []metav1.Condition{},
			},
			conditionType: mockType1,
			want:          nil,
		},
		{
			name: "condition found",
			getter: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:   mockType1,
						Status: metav1.ConditionTrue,
					},
				},
			},
			conditionType: mockType1,
			want: &metav1.Condition{
				Type:   mockType1,
				Status: metav1.ConditionTrue,
			},
		},
		{
			name: "condition not found",
			getter: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:   mockType1,
						Status: metav1.ConditionTrue,
					},
				},
			},
			conditionType: mockType2,
			want:          nil,
		},
		{
			name: "multiple conditions",
			getter: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:   mockType1,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   mockType2,
						Status: metav1.ConditionFalse,
					},
				},
			},
			conditionType: mockType2,
			want: &metav1.Condition{
				Type:   mockType2,
				Status: metav1.ConditionFalse,
			},
		},
		{
			name: "empty condition type",
			getter: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:   "",
						Status: metav1.ConditionTrue,
					},
				},
			},
			conditionType: "",
			want: &metav1.Condition{
				Type:   "",
				Status: metav1.ConditionTrue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(tt.getter, tt.conditionType)
			if tt.want == nil {
				require.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			require.Equal(t, tt.want.Type, got.Type)
			require.Equal(t, tt.want.Status, got.Status)
		})
	}
}

func TestSet(t *testing.T) {
	const (
		mockType   = "MockType"
		mockReason = "MockReason"
	)

	frozenTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		on         Setter
		conditions []*metav1.Condition
		assertions func(t *testing.T, existing, updated Setter)
	}{
		{
			name: "sets new condition",
			on:   &mockSetter{},
			conditions: []*metav1.Condition{
				{
					Type:               mockType,
					Status:             metav1.ConditionTrue,
					Reason:             mockReason,
					Message:            "MockMessage",
					ObservedGeneration: 12,
				},
			},
			assertions: func(t *testing.T, _, updated Setter) {
				require.NotNil(t, updated)

				conditions := updated.GetConditions()
				require.Len(t, conditions, 1)

				require.Equal(t, mockType, conditions[0].Type)
				require.Equal(t, metav1.ConditionTrue, conditions[0].Status)
				require.Equal(t, mockReason, conditions[0].Reason)
				require.Equal(t, "MockMessage", conditions[0].Message)
				require.NotNil(t, conditions[0].LastTransitionTime)
				require.Equal(t, int64(12), conditions[0].ObservedGeneration)
			},
		},
		{
			name: "replaces existing condition",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:    mockType,
						Status:  metav1.ConditionTrue,
						Reason:  mockReason,
						Message: "MockMessage",
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionFalse,
					Reason:  mockReason,
					Message: "MockMessage",
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				require.NotNil(t, updated)

				conditions := updated.GetConditions()
				require.Len(t, conditions, 1)

				oldCondition := Get(existing, mockType)
				require.NotNil(t, oldCondition)

				newCondition := Get(updated, mockType)
				require.NotNil(t, newCondition)

				require.NotEqual(t, oldCondition, newCondition)
				require.NotEqual(t, oldCondition.Status, newCondition.Status)
				require.NotEqual(t, oldCondition.LastTransitionTime, newCondition.LastTransitionTime)
			},
		},
		{
			name: "appends new condition",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:    "OtherType",
						Status:  metav1.ConditionTrue,
						Reason:  "MockReason1",
						Message: "MockMessage1",
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionTrue,
					Reason:  mockReason,
					Message: "MockMessage",
				},
			},
			assertions: func(t *testing.T, _, updated Setter) {
				require.NotNil(t, updated)

				conditions := updated.GetConditions()
				require.Len(t, conditions, 2)

				condition := Get(updated, mockType)
				require.NotNil(t, condition)
			},
		},
		{
			name: "sets multiple conditions",
			on:   &mockSetter{},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionTrue,
					Reason:  mockReason,
					Message: "MockMessage1",
				},
				{
					Type:    "OtherType",
					Status:  metav1.ConditionFalse,
					Reason:  "OtherReason",
					Message: "MockMessage2",
				},
			},
			assertions: func(t *testing.T, _, updated Setter) {
				require.NotNil(t, updated)

				conditions := updated.GetConditions()
				require.Len(t, conditions, 2)

				condition1 := Get(updated, mockType)
				require.NotNil(t, condition1)
				require.Equal(t, metav1.ConditionTrue, condition1.Status)

				condition2 := Get(updated, "OtherType")
				require.NotNil(t, condition2)
				require.Equal(t, metav1.ConditionFalse, condition2.Status)
			},
		},
		{
			name: "sets condition on Object with generation",
			on: &mockObject{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]any{
						"metadata": map[string]any{
							"generation": int64(42),
						},
					},
				},
				mockSetter: &mockSetter{},
			},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionTrue,
					Reason:  mockReason,
					Message: "MockMessage",
				},
				{
					Type:               "OtherType",
					Status:             metav1.ConditionFalse,
					Reason:             "OtherReason",
					Message:            "MockMessage2",
					ObservedGeneration: 41, // Should not be overwritten
				},
			},
			assertions: func(t *testing.T, _, updated Setter) {
				require.NotNil(t, updated)

				conditions := updated.GetConditions()
				require.Len(t, conditions, 2)

				condition := Get(updated, mockType)
				require.NotNil(t, condition)
				require.Equal(t, int64(42), condition.ObservedGeneration)

				otherCondition := Get(updated, "OtherType")
				require.NotNil(t, otherCondition)
				require.Equal(t, int64(41), otherCondition.ObservedGeneration)
			},
		},
		{
			name: "does not update when condition is equal and ObservedGeneration is not higher",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:               mockType,
						Status:             metav1.ConditionTrue,
						Reason:             mockReason,
						Message:            "MockMessage",
						ObservedGeneration: 42,
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:               mockType,
					Status:             metav1.ConditionTrue,
					Reason:             mockReason,
					Message:            "MockMessage",
					ObservedGeneration: 42,
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				require.NotNil(t, updated)

				oldCondition := Get(existing, mockType)
				newCondition := Get(updated, mockType)

				require.Equal(t, oldCondition, newCondition)
			},
		},
		{
			name: "updates when condition is equal but ObservedGeneration is higher",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:               mockType,
						Status:             metav1.ConditionTrue,
						Reason:             mockReason,
						Message:            "MockMessage",
						ObservedGeneration: 42,
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:               mockType,
					Status:             metav1.ConditionTrue,
					Reason:             mockReason,
					Message:            "MockMessage",
					ObservedGeneration: 43,
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				require.NotNil(t, updated)

				oldCondition := Get(existing, mockType)
				require.NotNil(t, oldCondition)

				newCondition := Get(updated, mockType)
				require.NotNil(t, newCondition)

				require.NotEqual(t, oldCondition, newCondition)
				require.Equal(t, int64(43), newCondition.ObservedGeneration)
			},
		},
		{
			name: "preserves LastTransitionTime for identical condition",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:               mockType,
						Status:             metav1.ConditionTrue,
						Reason:             mockReason,
						Message:            "MockMessage",
						LastTransitionTime: metav1.Time{Time: frozenTime.Add(-1 * time.Hour)},
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionTrue,
					Reason:  mockReason,
					Message: "MockMessage",
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				oldCondition := Get(existing, mockType)
				require.NotNil(t, oldCondition)

				newCondition := Get(updated, mockType)
				require.NotNil(t, newCondition)

				require.Equal(t, oldCondition.LastTransitionTime, newCondition.LastTransitionTime)
			},
		},
		{
			name: "uses existing non-zero LastTransitionTime when input LastTransitionTime is zero",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:               mockType,
						Status:             metav1.ConditionTrue,
						Reason:             mockReason,
						Message:            "MockMessage",
						LastTransitionTime: metav1.Time{Time: frozenTime.Add(-1 * time.Hour)},
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:    mockType,
					Status:  metav1.ConditionFalse,
					Reason:  mockReason,
					Message: "UpdatedMessage",
					// LastTransitionTime is intentionally left as zero value
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				oldCondition := Get(existing, mockType)
				require.NotNil(t, oldCondition)

				newCondition := Get(updated, mockType)
				require.NotNil(t, newCondition)

				require.NotEqual(t, oldCondition.Status, newCondition.Status)
				require.NotEqual(t, oldCondition.Message, newCondition.Message)

				// The key assertion: LastTransitionTime should be preserved from the old condition
				require.Equal(t, oldCondition.LastTransitionTime, newCondition.LastTransitionTime)
				require.False(t, newCondition.LastTransitionTime.IsZero())

				require.Equal(t, metav1.ConditionFalse, newCondition.Status)
				require.Equal(t, "UpdatedMessage", newCondition.Message)
			},
		},
		{
			name: "respects non-zero LastTransitionTime in input condition",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:               mockType,
						Status:             metav1.ConditionTrue,
						Reason:             mockReason,
						Message:            "MockMessage",
						LastTransitionTime: metav1.Time{Time: frozenTime.Add(-2 * time.Hour)},
					},
				},
			},
			conditions: []*metav1.Condition{
				{
					Type:               mockType,
					Status:             metav1.ConditionFalse,
					Reason:             mockReason,
					Message:            "UpdatedMessage",
					LastTransitionTime: metav1.Time{Time: frozenTime.Add(-1 * time.Hour)},
				},
			},
			assertions: func(t *testing.T, existing, updated Setter) {
				oldCondition := Get(existing, mockType)
				require.NotNil(t, oldCondition)

				newCondition := Get(updated, mockType)
				require.NotNil(t, newCondition)

				require.NotEqual(t, oldCondition.LastTransitionTime, newCondition.LastTransitionTime)
				require.Equal(t, newCondition.LastTransitionTime, metav1.Time{Time: frozenTime.Add(-1 * time.Hour)})
				require.Equal(t, metav1.ConditionFalse, newCondition.Status)
				require.Equal(t, "UpdatedMessage", newCondition.Message)
			},
		},
		{
			name: "handles nil condition",
			on: &mockSetter{
				conditions: []metav1.Condition{
					{
						Type:    mockType,
						Status:  metav1.ConditionTrue,
						Reason:  mockReason,
						Message: "MockMessage",
					},
				},
			},
			conditions: nil,
			assertions: func(t *testing.T, existing, updated Setter) {
				require.NotNil(t, updated)
				require.Equal(t, existing.GetConditions(), updated.GetConditions())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := tt.on.(deepCopyable).DeepCopy() // nolint: forcetypeassert
			Set(tt.on, tt.conditions...)
			tt.assertions(t, old, tt.on)
		})
	}
}

func TestDelete(t *testing.T) {
	const (
		mockType1 = "MockType1"
		mockType2 = "MockType2"
	)

	tests := []struct {
		name           string
		setter         *mockSetter
		conditionType  string
		wantConditions []metav1.Condition
	}{
		{
			name:           "nil setter",
			setter:         nil,
			conditionType:  mockType1,
			wantConditions: nil,
		},
		{
			name: "empty condition type",
			setter: &mockSetter{
				conditions: []metav1.Condition{
					{Type: mockType1, Status: metav1.ConditionTrue},
				},
			},
			conditionType: "",
			wantConditions: []metav1.Condition{
				{Type: mockType1, Status: metav1.ConditionTrue},
			},
		},
		{
			name: "empty conditions",
			setter: &mockSetter{
				conditions: []metav1.Condition{},
			},
			conditionType:  mockType1,
			wantConditions: []metav1.Condition{},
		},
		{
			name: "condition found and deleted",
			setter: &mockSetter{
				conditions: []metav1.Condition{
					{Type: mockType1, Status: metav1.ConditionTrue},
				},
			},
			conditionType:  mockType1,
			wantConditions: []metav1.Condition{},
		},
		{
			name: "condition not found",
			setter: &mockSetter{
				conditions: []metav1.Condition{
					{Type: mockType1, Status: metav1.ConditionTrue},
				},
			},
			conditionType: mockType2,
			wantConditions: []metav1.Condition{
				{Type: mockType1, Status: metav1.ConditionTrue},
			},
		},
		{
			name: "multiple conditions with target at start",
			setter: &mockSetter{
				conditions: []metav1.Condition{
					{Type: mockType1, Status: metav1.ConditionTrue},
					{Type: mockType2, Status: metav1.ConditionFalse},
				},
			},
			conditionType: mockType1,
			wantConditions: []metav1.Condition{
				{Type: mockType2, Status: metav1.ConditionFalse},
			},
		},
		{
			name: "multiple conditions with target at end",
			setter: &mockSetter{
				conditions: []metav1.Condition{
					{Type: mockType1, Status: metav1.ConditionTrue},
					{Type: mockType2, Status: metav1.ConditionFalse},
				},
			},
			conditionType: mockType2,
			wantConditions: []metav1.Condition{
				{Type: mockType1, Status: metav1.ConditionTrue},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Delete(tt.setter, tt.conditionType)
			require.Equal(t, tt.wantConditions, tt.setter.GetConditions())
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a    metav1.Condition
		b    metav1.Condition
		want bool
	}{
		{
			name: "identical conditions",
			a: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message",
			},
			b: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message",
			},
			want: true,
		},
		{
			name: "different type",
			a: metav1.Condition{
				Type:    "TestType1",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message",
			},
			b: metav1.Condition{
				Type:    "TestType2",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message",
			},
			want: false,
		},
		{
			name: "different status",
			a: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message",
			},
			b: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionFalse,
				Reason:  "TestReason",
				Message: "Test message",
			},
			want: false,
		},
		{
			name: "different reason",
			a: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason1",
				Message: "Test message",
			},
			b: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason2",
				Message: "Test message",
			},
			want: false,
		},
		{
			name: "different message",
			a: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message 1",
			},
			b: metav1.Condition{
				Type:    "TestType",
				Status:  metav1.ConditionTrue,
				Reason:  "TestReason",
				Message: "Test message 2",
			},
			want: false,
		},
		{
			name: "different LastTransitionTime",
			a: metav1.Condition{
				Type:               "TestType",
				Status:             metav1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "Test message",
				LastTransitionTime: metav1.Time{Time: time.Now()},
			},
			b: metav1.Condition{
				Type:               "TestType",
				Status:             metav1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "Test message",
				LastTransitionTime: metav1.Time{Time: time.Now().Add(1 * time.Hour)},
			},
			want: true,
		},
		{
			name: "different ObservedGeneration",
			a: metav1.Condition{
				Type:               "TestType",
				Status:             metav1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "Test message",
				ObservedGeneration: 1,
			},
			b: metav1.Condition{
				Type:               "TestType",
				Status:             metav1.ConditionTrue,
				Reason:             "TestReason",
				Message:            "Test message",
				ObservedGeneration: 2,
			},
			want: true,
		},
		{
			name: "empty conditions",
			a:    metav1.Condition{},
			b:    metav1.Condition{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Equal(tt.a, tt.b)
			require.Equal(t, tt.want, got)
		})
	}
}

type deepCopyable interface {
	DeepCopy() Setter
}

type mockSetter struct {
	conditions []metav1.Condition
}

func (m *mockSetter) GetConditions() []metav1.Condition {
	if m == nil {
		return nil
	}

	return m.conditions
}

func (m *mockSetter) SetConditions(conditions []metav1.Condition) {
	m.conditions = conditions
}

func (m *mockSetter) DeepCopy() Setter {
	if m == nil {
		return nil
	}

	conditions := make([]metav1.Condition, len(m.conditions))
	copy(conditions, m.conditions)

	return &mockSetter{
		conditions: conditions,
	}
}

type mockObject struct {
	*unstructured.Unstructured
	*mockSetter
}

func (m *mockObject) DeepCopy() Setter {
	if m == nil {
		return nil
	}

	return &mockObject{
		Unstructured: m.Unstructured.DeepCopy(),
		mockSetter:   m.mockSetter.DeepCopy().(*mockSetter), // nolint: forcetypeassert
	}
}
