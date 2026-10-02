package can

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestVerbs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		verb     Verb
		expected Verb
	}{
		{verb: Get(), expected: "get"},
		{verb: List(), expected: "list"},
		{verb: Watch(), expected: "watch"},
		{verb: Create(), expected: "create"},
		{verb: Update(), expected: "update"},
		{verb: Patch(), expected: "patch"},
		{verb: Delete(), expected: "delete"},
		{verb: DeleteCollection(), expected: "deletecollection"},
		{verb: Promote(), expected: "promote"},
		{verb: Do("escalate"), expected: "escalate"},
	}
	for _, testCase := range testCases {
		t.Run(string(testCase.expected), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.verb)
		})
	}
}

func TestAccess(t *testing.T) {
	t.Parallel()
	kargoResource := kargoapi.GroupVersion.WithResource
	testCases := []struct {
		name     string
		access   Access
		expected Access
	}{
		{
			name: "any resource",
			access: Get().Resource(
				schema.GroupVersionResource{Version: "v1", Resource: "secrets"},
				types.NamespacedName{Namespace: "kargo-demo", Name: "creds"},
			),
			expected: Access{
				Verb:     "get",
				Resource: schema.GroupVersionResource{Version: "v1", Resource: "secrets"},
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "creds"},
			},
		},
		{
			name:   "Kargo collection",
			access: List().Kargo("freights"),
			expected: Access{
				Verb:     "list",
				Resource: kargoResource("freights"),
			},
		},
		{
			name:   "Kargo collection in a namespace",
			access: List().Kargo("stages").In("kargo-demo"),
			expected: Access{
				Verb:     "list",
				Resource: kargoResource("stages"),
				Key:      types.NamespacedName{Namespace: "kargo-demo"},
			},
		},
		{
			name:   "Stage",
			access: Promote().Stage("kargo-demo", "uat"),
			expected: Access{
				Verb:     "promote",
				Resource: kargoResource("stages"),
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "uat"},
			},
		},
		{
			name:   "Promotion",
			access: Get().Promotion("kargo-demo", "uat.abc123"),
			expected: Access{
				Verb:     "get",
				Resource: kargoResource("promotions"),
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "uat.abc123"},
			},
		},
		{
			name:   "PromotionRequest",
			access: Get().PromotionRequest("kargo-demo", "uat.abc123"),
			expected: Access{
				Verb:     "get",
				Resource: kargoResource("promotionrequests"),
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "uat.abc123"},
			},
		},
		{
			name:   "Warehouse",
			access: Get().Warehouse("kargo-demo", "images"),
			expected: Access{
				Verb:     "get",
				Resource: kargoResource("warehouses"),
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "images"},
			},
		},
		{
			name:   "ProjectConfig is named after its Project",
			access: Get().ProjectConfig("kargo-demo"),
			expected: Access{
				Verb:     "get",
				Resource: kargoResource("projectconfigs"),
				Key:      types.NamespacedName{Namespace: "kargo-demo", Name: "kargo-demo"},
			},
		},
		{
			name:   "ClusterConfig is cluster-scoped",
			access: Get().ClusterConfig("cluster"),
			expected: Access{
				Verb:     "get",
				Resource: kargoResource("clusterconfigs"),
				Key:      types.NamespacedName{Name: "cluster"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.access)
		})
	}
}
