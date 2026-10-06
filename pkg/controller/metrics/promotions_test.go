package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_countPromotionsByProject(t *testing.T) {
	t.Parallel()

	promotion := func(project string) kargoapi.Promotion {
		return kargoapi.Promotion{ObjectMeta: metav1.ObjectMeta{Namespace: project}}
	}

	testCases := []struct {
		name       string
		promotions []kargoapi.Promotion
		expected   map[string]float64
	}{
		{
			name:     "no Promotions yields no Projects",
			expected: map[string]float64{},
		},
		{
			name: "Promotions are counted per Project",
			promotions: []kargoapi.Promotion{
				promotion("project-a"),
				promotion("project-a"),
				promotion("project-b"),
			},
			expected: map[string]float64{"project-a": 2, "project-b": 1},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, countPromotionsByProject(testCase.promotions))
		})
	}
}
