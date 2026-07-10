package gate

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDefaultEvaluate(t *testing.T) {
	t.Parallel()

	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}
	directStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-project",
			Name:      "test-stage",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin:  origin,
				Sources: kargoapi.FreightSources{Direct: true},
			}},
		},
	}
	freight := func(namespace string) *kargoapi.Freight {
		return &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "test-freight",
			},
			Origin: origin,
		}
	}

	t.Run("eligible direct Freight is allowed", func(t *testing.T) {
		t.Parallel()
		decision, err := DefaultEvaluate(
			t.Context(),
			directStage,
			freight("test-project"),
		)
		require.NoError(t, err)
		require.NotNil(t, decision)
		require.True(t, decision.Allow)
	})

	t.Run("Freight in another namespace is denied", func(t *testing.T) {
		t.Parallel()
		decision, err := DefaultEvaluate(
			t.Context(),
			directStage,
			freight("other-project"),
		)
		require.NoError(t, err)
		require.NotNil(t, decision)
		require.False(t, decision.Allow)
	})

	t.Run("unrequested origin is denied", func(t *testing.T) {
		t.Parallel()
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-project",
				Name:      "test-stage",
			},
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "other-warehouse",
					},
					Sources: kargoapi.FreightSources{Direct: true},
				}},
			},
		}
		decision, err := DefaultEvaluate(t.Context(), stage, freight("test-project"))
		require.NoError(t, err)
		require.NotNil(t, decision)
		require.False(t, decision.Allow)
	})

	t.Run("nil Stage yields an error", func(t *testing.T) {
		t.Parallel()
		_, err := DefaultEvaluate(t.Context(), nil, freight("test-project"))
		require.Error(t, err)
	})
}
