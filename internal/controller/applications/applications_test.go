package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestAppHealthChangePredicate(t *testing.T) {
	testCases := []struct {
		name    string
		old     map[string]any
		new     map[string]any
		updated bool
	}{
		{
			name: "health changed",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "health did not change",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			updated: false,
		},
		{
			name: "sync status changed",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "sync status did not change",
			old: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"status": "Healthy",
					},
				},
			},
			updated: false,
		},
		{
			name: "revision changed",
			old: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"revision": "fake-revision",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"revision": "different-fake-revision",
					},
				},
			},
			updated: true,
		},
		{
			name: "revision did not change",
			old: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"revision": "fake-revision",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"revision": "fake-revision",
					},
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := AppHealthSyncStatusChangePredicate{}
			newUn := &unstructured.Unstructured{Object: testCase.new}
			oldUn := &unstructured.Unstructured{Object: testCase.old}
			updated := p.Update(event.UpdateEvent{
				ObjectNew: newUn,
				ObjectOld: oldUn,
			})
			require.Equal(t, testCase.updated, updated)
		})
	}
}
