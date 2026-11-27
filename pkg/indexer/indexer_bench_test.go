package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// BenchmarkRunningPromotionsByArgoCDApplications covers a Running Promotion
// at its argocd-update step, by app count and app name form, and a
// non-Running one.
func BenchmarkRunningPromotionsByArgoCDApplications(b *testing.B) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
	}
	// Get mirrors a cache read: a deep copy, no serialization.
	c := interceptor.NewClient(fake.NewClientBuilder().Build(), interceptor.Funcs{
		Get: func(
			_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption,
		) error {
			stage.DeepCopyInto(obj.(*kargoapi.Stage)) //nolint:forcetypeassert
			return nil
		},
	})
	idx := RunningPromotionsByArgoCDApplications(b.Context(), c, "", true)

	// promotion returns a Running Promotion at the argocd-update step of a
	// clone, update, commit, push, argocd-update template.
	promotion := func(
		argoCDConfig []byte,
		promoVars []kargoapi.ExpressionVariable,
		stepVars []kargoapi.ExpressionVariable,
	) *kargoapi.Promotion {
		freight := kargoapi.FreightReference{
			Name:   "fake-freight",
			Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"},
			Images: []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1.2.3"}},
		}
		col := &kargoapi.FreightCollection{}
		col.UpdateOrPush(freight)
		return &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-promo"},
			Spec: kargoapi.PromotionSpec{
				Stage:   "fake-stage",
				Freight: freight.Name,
				Vars:    promoVars,
				Steps: []kargoapi.PromotionStep{
					{Uses: "git-clone", As: "clone", Config: &apiextensionsv1.JSON{Raw: []byte(
						`{"repoURL":"https://github.com/example/repo.git","checkout":[` +
							`{"branch":"main","path":"./src"},` +
							`{"branch":"stage/${{ ctx.stage }}","create":true,"path":"./out"}]}`,
					)}},
					{Uses: "yaml-update", As: "update", Config: &apiextensionsv1.JSON{Raw: []byte(
						`{"path":"./out/values.yaml","updates":[` +
							`{"key":"image.tag","value":"${{ imageFrom('example.com/app').Tag }}"}]}`,
					)}},
					{Uses: "git-commit", As: "commit", Config: &apiextensionsv1.JSON{Raw: []byte(
						`{"path":"./out","message":"${{ outputs.update.commitMessage }}"}`,
					)}},
					{Uses: "git-push", As: "push", Config: &apiextensionsv1.JSON{Raw: []byte(
						`{"path":"./out"}`,
					)}},
					{Uses: "argocd-update", Vars: stepVars, Config: &apiextensionsv1.JSON{Raw: argoCDConfig}},
				},
			},
			Status: kargoapi.PromotionStatus{
				Phase:             kargoapi.PromotionPhaseRunning,
				CurrentStep:       4,
				Freight:           &freight,
				FreightCollection: col,
				State: &apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf(
					`{"update":{"commitMessage":"Updated values.yaml"},`+
						`"commit":{"commit":"%040d"},`+
						`"push":{"branch":"stage/fake-stage","commit":"%040d"}}`,
					1, 1,
				))},
			},
		}
	}

	// Each form resolves app i to "fake-stage-app-<i>".
	forms := []struct {
		name      string
		app       string
		promoVars []kargoapi.ExpressionVariable
		stepVars  []kargoapi.ExpressionVariable
	}{
		{name: "literal", app: "fake-stage-app-%d"},
		{name: "templated", app: "${{ ctx.stage }}-app-%d"},
		{
			name:      "vars",
			app:       "${{ vars.app }}-%d",
			promoVars: []kargoapi.ExpressionVariable{{Name: "env", Value: "${{ ctx.stage }}"}},
			stepVars:  []kargoapi.ExpressionVariable{{Name: "app", Value: "${{ vars.env }}-app"}},
		},
	}
	for _, n := range []int{1, 10} {
		for _, f := range forms {
			b.Run(fmt.Sprintf("apps-%d/%s", n, f.name), func(b *testing.B) {
				apps := make([]map[string]string, n)
				want := make([]string, n)
				for i := range n {
					apps[i] = map[string]string{"namespace": "argocd", "name": fmt.Sprintf(f.app, i)}
					want[i] = fmt.Sprintf("argocd:fake-stage-app-%d", i)
				}
				cfg, err := json.Marshal(map[string]any{"apps": apps})
				require.NoError(b, err)
				promo := promotion(cfg, f.promoVars, f.stepVars)
				require.Equal(b, want, idx(promo))
				b.ReportAllocs()
				for b.Loop() {
					idx(promo)
				}
			})
		}
	}
	b.Run("not-running", func(b *testing.B) {
		promo := promotion([]byte(`{"apps":[{"name":"fake-stage-app-0"}]}`), nil, nil)
		promo.Status.Phase = kargoapi.PromotionPhaseSucceeded
		require.Nil(b, idx(promo))
		b.ReportAllocs()
		for b.Loop() {
			idx(promo)
		}
	})
}
