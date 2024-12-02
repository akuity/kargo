package promotiontemplate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_webhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		objects    []client.Object
		template   *kargoapi.PromotionTemplate
		assertions func(*testing.T, admission.Warnings, error)
	}{
		{
			name: "project does not exist",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
					},
				},
			},
			template: &kargoapi.PromotionTemplate{
				ObjectMeta: v1.ObjectMeta{
					Name:      "fake-template",
					Namespace: "fake-project",
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.ErrorContains(t, err, "namespace \"fake-project\" is not a project")
			},
		},
		{
			name: "project exists",
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
						Labels: map[string]string{
							kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
						},
					},
				},
				&kargoapi.Project{
					ObjectMeta: v1.ObjectMeta{
						Name: "fake-project",
					},
				},
			},
			template: &kargoapi.PromotionTemplate{
				ObjectMeta: v1.ObjectMeta{
					Name:      "fake-template",
					Namespace: "fake-project",
				},
			},
			assertions: func(t *testing.T, warnings admission.Warnings, err error) {
				assert.Empty(t, warnings)
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithScheme(scheme).
				Build()

			w := &webhook{
				client: c,
			}

			got, err := w.ValidateCreate(context.Background(), tt.template)
			tt.assertions(t, got, err)
		})
	}
}
