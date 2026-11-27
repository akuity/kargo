package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	corev1 "k8s.io/api/core/v1"
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

func TestObjectAttributes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		obj      metav1.Object
		expected []attribute.KeyValue
	}{
		{
			name: "nil",
			obj:  (*kargoapi.Freight)(nil),
		},
		{
			name: "Freight",
			obj:  testFreight(),
			expected: []attribute.KeyValue{
				KindKey.String("Freight"),
				NameKey.String("abc123"),
				ProjectKey.String("smoke"),
				FreightKey.String("abc123"),
				FreightOriginKey.String("Warehouse/wh"),
				WarehouseKey.String("wh"),
				VCSCommitKey.String("deadbeef"),
			},
		},
		{
			name: "Warehouse",
			obj: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "wh"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Warehouse"),
				NameKey.String("wh"),
				ProjectKey.String("smoke"),
				WarehouseKey.String("wh"),
			},
		},
		{
			name: "Stage",
			obj: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Stage"),
				NameKey.String("dev"),
				ProjectKey.String("smoke"),
				StageKey.String("dev"),
			},
		},
		{
			name: "Promotion",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev.01"},
				Spec:       kargoapi.PromotionSpec{Stage: "dev", Freight: "abc123"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Promotion"),
				NameKey.String("dev.01"),
				ProjectKey.String("smoke"),
				PromotionKey.String("dev.01"),
				StageKey.String("dev"),
				FreightKey.String("abc123"),
			},
		},
		{
			name: "Promotion with only a name, as refreshed from the API",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev.01"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Promotion"),
				NameKey.String("dev.01"),
				ProjectKey.String("smoke"),
				PromotionKey.String("dev.01"),
			},
		},
		{
			name: "Freight with a name not generated yet",
			obj: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Freight"),
				ProjectKey.String("smoke"),
			},
		},
		{
			name: "PromotionRequest",
			obj: &kargoapi.PromotionRequest{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "req"},
				Spec:       kargoapi.PromotionRequestSpec{Stage: "dev", Freight: "abc123"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("PromotionRequest"),
				NameKey.String("req"),
				ProjectKey.String("smoke"),
				PromotionRequestKey.String("req"),
				StageKey.String("dev"),
				FreightKey.String("abc123"),
			},
		},
		{
			name: "Project",
			obj:  &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "smoke"}},
			expected: []attribute.KeyValue{
				KindKey.String("Project"),
				NameKey.String("smoke"),
				ProjectKey.String("smoke"),
			},
		},
		{
			name: "namespaced Kargo kind without its own attribute",
			obj: &kargoapi.PromotionTask{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "task"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("PromotionTask"),
				NameKey.String("task"),
				ProjectKey.String("smoke"),
			},
		},
		{
			name: "cluster-scoped Kargo kind",
			obj:  &kargoapi.ClusterConfig{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}},
			expected: []attribute.KeyValue{
				KindKey.String("ClusterConfig"),
				NameKey.String("cluster"),
			},
		},
		{
			name: "namespaced, not Kargo",
			obj: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kargo", Name: "cfg"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("ConfigMap"),
				NameKey.String("cfg"),
				NamespaceKey.String("kargo"),
			},
		},
		{
			name: "Secret, without its name",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "repo-creds"},
			},
			expected: []attribute.KeyValue{
				KindKey.String("Secret"),
				NamespaceKey.String("smoke"),
			},
		},
		{
			name: "cluster-scoped, not Kargo",
			obj:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "smoke"}},
			expected: []attribute.KeyValue{
				KindKey.String("Namespace"),
				NameKey.String("smoke"),
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, ObjectAttributes(testCase.obj))
		})
	}
	require.Nil(t, KargoAttributes(nil))
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
