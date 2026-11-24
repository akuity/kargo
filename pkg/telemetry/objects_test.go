package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func testFreight() *kargoapi.Freight {
	return &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc123"},
		Origin:     kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "wh"},
		Commits:    []kargoapi.GitCommit{{ID: "deadbeef"}},
	}
}

func TestFreightDetailAttributes(t *testing.T) {
	t.Parallel()
	f := testFreight()
	require.ElementsMatch(
		t,
		FreightDetailAttributes(f),
		[]any{
			FreightOriginKey.String("Warehouse/wh"),
			WarehouseKey.String("wh"),
			VCSCommitKey.String("deadbeef"),
		},
	)
	// With commits from several repositories, none is the Freight's.
	f.Commits = append(f.Commits, kargoapi.GitCommit{ID: "cafef00d"})
	require.NotContains(t, FreightDetailAttributes(f), VCSCommitKey.String("deadbeef"))
}
