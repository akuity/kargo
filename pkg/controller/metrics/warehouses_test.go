package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_oldestDiscoveryByProject(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	warehouse := func(project string, discovered *time.Time) kargoapi.Warehouse {
		wh := kargoapi.Warehouse{ObjectMeta: metav1.ObjectMeta{Namespace: project}}
		if discovered != nil {
			wh.Status.DiscoveredArtifacts = &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: metav1.Time{Time: *discovered},
			}
		}
		return wh
	}
	at := func(ago time.Duration) *time.Time {
		t := t0.Add(-ago)
		return &t
	}

	require.Empty(t, oldestDiscoveryByProject(nil))
	require.Equal(
		t,
		map[string]time.Time{
			"project-a": t0.Add(-10 * time.Minute),
			"project-b": t0.Add(-time.Minute),
		},
		oldestDiscoveryByProject([]kargoapi.Warehouse{
			warehouse("project-a", at(time.Minute)),
			warehouse("project-a", at(10*time.Minute)),
			// Never discovered: not considered.
			warehouse("project-a", nil),
			warehouse("project-b", at(time.Minute)),
			warehouse("project-c", nil),
		}),
	)
}
