package stages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestIsControlFlowStage_Create(t *testing.T) {
	tests := []struct {
		name string
		s    IsControlFlowStage
		e    event.CreateEvent
		want bool
	}{
		{
			name: "nil object",
			s:    true,
			e: event.CreateEvent{
				Object: nil,
			},
			want: false,
		},
		{
			name: "wrong type",
			s:    true,
			e: event.CreateEvent{
				Object: &corev1.Node{},
			},
			want: false,
		},
		{
			name: "non-control flow stage when looking for control flow",
			s:    true,
			e: event.CreateEvent{
				Object: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "control flow stage when looking for control flow",
			s:    true,
			e: event.CreateEvent{
				Object: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						// No promotion template makes it a control flow stage
					},
				},
			},
			want: true,
		},
		{
			name: "non-control flow stage when looking for non-control flow",
			s:    false,
			e: event.CreateEvent{
				Object: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "control flow stage when looking for non-control flow",
			s:    false,
			e: event.CreateEvent{
				Object: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						// No promotion template makes it a control flow stage
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Create(tt.e)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsControlFlowStage_Update(t *testing.T) {
	tests := []struct {
		name string
		s    IsControlFlowStage
		e    event.UpdateEvent
		want bool
	}{
		{
			name: "nil object",
			s:    true,
			e: event.UpdateEvent{
				ObjectNew: nil,
			},
			want: false,
		},
		{
			name: "wrong type",
			s:    true,
			e: event.UpdateEvent{
				ObjectNew: &corev1.Node{},
			},
			want: false,
		},
		{
			name: "non-control flow stage when looking for control flow",
			s:    true,
			e: event.UpdateEvent{
				ObjectNew: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "control flow stage when looking for control flow",
			s:    true,
			e: event.UpdateEvent{
				ObjectNew: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						// No promotion template makes it a control flow stage
					},
				},
			},
			want: true,
		},
		{
			name: "non-control flow stage when looking for non-control flow",
			s:    false,
			e: event.UpdateEvent{
				ObjectNew: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "control flow stage when looking for non-control flow",
			s:    false,
			e: event.UpdateEvent{
				ObjectNew: &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						// No promotion template makes it a control flow stage
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Update(tt.e)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsControlFlowStage_Delete(t *testing.T) {
	tests := []struct {
		name string
		s    IsControlFlowStage
		e    event.DeleteEvent
		want bool
	}{
		{
			name: "always returns false",
			s:    true,
			e:    event.DeleteEvent{},
			want: false,
		},
		{
			name: "always returns false when looking for non-control flow",
			s:    false,
			e:    event.DeleteEvent{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Delete(tt.e)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsControlFlowStage_Generic(t *testing.T) {
	tests := []struct {
		name string
		s    IsControlFlowStage
		e    event.GenericEvent
		want bool
	}{
		{
			name: "always returns false",
			s:    true,
			e:    event.GenericEvent{},
			want: false,
		},
		{
			name: "always returns false when looking for non-control flow",
			s:    false,
			e:    event.GenericEvent{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Generic(tt.e)
			assert.Equal(t, tt.want, got)
		})
	}
}
