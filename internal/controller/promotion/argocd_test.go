package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestNewArgoCDMechanism(t *testing.T) {
	pm := newArgoCDMechanism(
		fake.NewClientBuilder().Build(),
	)
	apm, ok := pm.(*argoCDMechanism)
	require.True(t, ok)
	require.NotNil(t, apm.doSingleUpdateFn)
	require.NotNil(t, apm.getArgoCDAppFn)
	require.NotNil(t, apm.applyArgoCDSourceUpdateFn)
	require.NotNil(t, apm.argoCDAppPatchFn)
}

func TestArgoCDGetName(t *testing.T) {
	require.NotEmpty(t, (&argoCDMechanism{}).GetName())
}

func TestArgoCDPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *argoCDMechanism
		stage      *kargoapi.Stage
		newFreight kargoapi.FreightReference
		assertions func(
			t *testing.T,
			newStatus *kargoapi.PromotionStatus,
			newFreightIn kargoapi.FreightReference,
			newFreightOut kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name:      "no updates",
			promoMech: &argoCDMechanism{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name:      "argo cd integration disabled",
			promoMech: &argoCDMechanism{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				_ kargoapi.FreightReference,
				_ kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"Argo CD integration is disabled on this controller",
				)
			},
		},
		{
			name: "error applying update",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				doSingleUpdateFn: func(
					context.Context,
					metav1.ObjectMeta,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) error {
					return errors.New("something went wrong")
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(
					t,
					"something went wrong",
					err.Error(),
				)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &argoCDMechanism{
				argocdClient: fake.NewClientBuilder().Build(),
				doSingleUpdateFn: func(
					context.Context,
					metav1.ObjectMeta,
					kargoapi.ArgoCDAppUpdate,
					kargoapi.FreightReference,
				) error {
					return nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, newFreightOut, err := testCase.promoMech.Promote(
				context.Background(),
				testCase.stage,
				&kargoapi.Promotion{},
				testCase.newFreight,
			)
			testCase.assertions(t, newStatus, testCase.newFreight, newFreightOut, err)
		})
	}
}

func TestArgoCDDoSingleUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *argoCDMechanism
		stageMeta  metav1.ObjectMeta
		update     kargoapi.ArgoCDAppUpdate
		assertions func(*testing.T, error)
	}{
		{
			name: "error getting Argo CD App",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error finding Argo CD Application")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "Argo CD App not found",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to find Argo CD Application")
			},
		},
		{
			name: "update not authorized",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							// The annotations that would permit this are missing
						},
					}, nil
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "does not permit mutation by")
			},
		},
		{
			name: "error updating app.Spec.Source",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
						Spec: argocd.ApplicationSpec{
							Source: &argocd.ApplicationSource{},
						},
					}, nil
				},
				applyArgoCDSourceUpdateFn: func(
					argocd.ApplicationSource,
					kargoapi.FreightReference,
					kargoapi.ArgoCDSourceUpdate,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error updating source of Argo CD Application",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error updating app.Spec.Sources",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
						Spec: argocd.ApplicationSpec{
							Sources: []argocd.ApplicationSource{
								{},
							},
						},
					}, nil
				},
				applyArgoCDSourceUpdateFn: func(
					argocd.ApplicationSource,
					kargoapi.FreightReference,
					kargoapi.ArgoCDSourceUpdate,
				) (argocd.ApplicationSource, error) {
					return argocd.ApplicationSource{}, errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			update: kargoapi.ArgoCDAppUpdate{
				SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
					{},
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error updating source(s) of Argo CD Application",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error patching Application",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
					}, nil
				},
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error patching Argo CD Application",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			promoMech: &argoCDMechanism{
				getArgoCDAppFn: func(
					context.Context,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: "fake-namespace",
							Annotations: map[string]string{
								authorizedStageAnnotationKey: "fake-namespace:fake-name",
							},
						},
					}, nil
				},
				argoCDAppPatchFn: func(
					context.Context,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return nil
				},
			},
			stageMeta: metav1.ObjectMeta{
				Name:      "fake-name",
				Namespace: "fake-namespace",
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.promoMech.doSingleUpdate(
					context.Background(),
					testCase.stageMeta,
					testCase.update,
					kargoapi.FreightReference{},
				),
			)
		})
	}
}

func TestAuthorizeArgoCDAppUpdate(t *testing.T) {
	permErr := "does not permit mutation"
	parseErr := "unable to parse"
	invalidGlobErr := "invalid glob expression"
	testCases := []struct {
		name    string
		appMeta metav1.ObjectMeta
		errMsg  string
	}{
		{
			name:    "annotations are nil",
			appMeta: metav1.ObjectMeta{},
			errMsg:  permErr,
		},
		{
			name: "annotation is missing",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			errMsg: permErr,
		},
		{
			name: "annotation cannot be parsed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "bogus",
				},
			},
			errMsg: parseErr,
		},
		{
			name: "mutation is not allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-nope:name-nope",
				},
			},
			errMsg: permErr,
		},
		{
			name: "mutation is allowed",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-yep:name-yep",
				},
			},
		},
		{
			name: "wildcard namespace with full name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*:name-yep",
				},
			},
		},
		{
			name: "full namespace with wildcard name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "ns-yep:*",
				},
			},
		},
		{
			name: "partial wildcards in namespace and name",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*-ye*:*-y*",
				},
			},
		},
		{
			name: "wildcards do not match",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*-nope:*-nope",
				},
			},
			errMsg: permErr,
		},
		{
			name: "invalid namespace glob",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*[:*",
				},
			},
			errMsg: invalidGlobErr,
		},
		{
			name: "invalid name glob",
			appMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					authorizedStageAnnotationKey: "*:*[",
				},
			},
			errMsg: invalidGlobErr,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := authorizeArgoCDAppUpdate(
				metav1.ObjectMeta{
					Name:      "name-yep",
					Namespace: "ns-yep",
				},
				testCase.appMeta,
			)
			if testCase.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, testCase.errMsg)
			}
		})
	}
}

func TestApplyArgoCDSourceUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		source     argocd.ApplicationSource
		newFreight kargoapi.FreightReference
		update     kargoapi.ArgoCDSourceUpdate
		assertions func(
			t *testing.T,
			originalSource argocd.ApplicationSource,
			updatedSource argocd.ApplicationSource,
			err error,
		)
	}{
		{
			name: "update doesn't apply to this source",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "different-fake-url",
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Source should be entirely unchanged
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (git)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				UpdateTargetRevision: true,
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-commit", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update target revision (helm chart)",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
				Chart:   "fake-chart",
			},
			newFreight: kargoapi.FreightReference{
				Charts: []kargoapi.Chart{
					{
						RepoURL: "fake-url",
						Name:    "fake-chart",
						Version: "fake-version",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL:              "fake-url",
				Chart:                "fake-chart",
				UpdateTargetRevision: true,
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// TargetRevision should be updated
				require.Equal(t, "fake-version", updatedSource.TargetRevision)
				// Everything else should be unchanged
				updatedSource.TargetRevision = originalSource.TargetRevision
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with kustomize",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Kustomize: &kargoapi.ArgoCDKustomize{
					Images: []kargoapi.ArgoCDKustomizeImageUpdate{
						{
							Image: "fake-image-url",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Kustomize attributes should be updated
				require.NotNil(t, updatedSource.Kustomize)
				require.Equal(
					t,
					argocd.KustomizeImages{
						argocd.KustomizeImage("fake-image-url=fake-image-url:fake-tag"),
					},
					updatedSource.Kustomize.Images,
				)
				// Everything else should be unchanged
				updatedSource.Kustomize = originalSource.Kustomize
				require.Equal(t, originalSource, updatedSource)
			},
		},

		{
			name: "update images with helm",
			source: argocd.ApplicationSource{
				RepoURL: "fake-url",
			},
			newFreight: kargoapi.FreightReference{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
					{
						// This one should not be updated because it's not a match for
						// anything in the update instructions
						RepoURL: "another-fake-image-url",
						Tag:     "another-fake-tag",
					},
				},
			},
			update: kargoapi.ArgoCDSourceUpdate{
				RepoURL: "fake-url",
				Helm: &kargoapi.ArgoCDHelm{
					Images: []kargoapi.ArgoCDHelmImageUpdate{
						{
							Image: "fake-image-url",
							Key:   "image",
							Value: kargoapi.ImageUpdateValueTypeImageAndTag,
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				originalSource argocd.ApplicationSource,
				updatedSource argocd.ApplicationSource,
				err error,
			) {
				require.NoError(t, err)
				// Helm attributes should be updated
				require.NotNil(t, updatedSource.Helm)
				require.NotNil(t, updatedSource.Helm.Parameters)
				require.Equal(
					t,
					[]argocd.HelmParameter{
						{
							Name:  "image",
							Value: "fake-image-url:fake-tag",
						},
					},
					updatedSource.Helm.Parameters,
				)
				// Everything else should be unchanged
				updatedSource.Helm = originalSource.Helm
				require.Equal(t, originalSource, updatedSource)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedSource, err := applyArgoCDSourceUpdate(
				testCase.source,
				testCase.newFreight,
				testCase.update,
			)
			testCase.assertions(t, testCase.source, updatedSource, err)
		})
	}
}

func TestBuildKustomizeImagesForArgoCDAppSource(t *testing.T) {
	images := []kargoapi.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
			Digest:  "fake-digest",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
			Digest:  "another-fake-digest",
		},
	}
	imageUpdates := []kargoapi.ArgoCDKustomizeImageUpdate{
		{Image: "fake-url"},
		{
			Image:     "another-fake-url",
			UseDigest: true,
		},
		{Image: "image-that-is-not-in-list"},
	}
	result := buildKustomizeImagesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"fake-url=fake-url:fake-tag",
			"another-fake-url=another-fake-url@another-fake-digest",
		},
		result,
	)
}

func TestBuildHelmParamChangesForArgoCDAppSource(t *testing.T) {
	images := []kargoapi.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
			Digest:  "fake-digest",
		},
		{
			RepoURL: "second-fake-url",
			Tag:     "second-fake-tag",
			Digest:  "second-fake-digest",
		},
		{
			RepoURL: "third-fake-url",
			Tag:     "third-fake-tag",
			Digest:  "third-fake-digest",
		},
		{
			RepoURL: "fourth-fake-url",
			Tag:     "fourth-fake-tag",
			Digest:  "fourth-fake-digest",
		},
	}
	imageUpdates := []kargoapi.ArgoCDHelmImageUpdate{
		{
			Image: "fake-url",
			Key:   "fake-key",
			Value: kargoapi.ImageUpdateValueTypeImageAndTag,
		},
		{
			Image: "second-fake-url",
			Key:   "second-fake-key",
			Value: kargoapi.ImageUpdateValueTypeTag,
		},
		{
			Image: "third-fake-url",
			Key:   "third-fake-key",
			Value: kargoapi.ImageUpdateValueTypeImageAndDigest,
		},
		{
			Image: "fourth-fake-url",
			Key:   "fourth-fake-key",
			Value: kargoapi.ImageUpdateValueTypeDigest,
		},
		{
			Image: "image-that-is-not-in-list",
			Key:   "fake-key",
			Value: "Tag",
		},
	}
	result := buildHelmParamChangesForArgoCDAppSource(images, imageUpdates)
	require.Equal(
		t,
		map[string]string{
			"fake-key":        "fake-url:fake-tag",
			"second-fake-key": "second-fake-tag",
			"third-fake-key":  "third-fake-url@third-fake-digest",
			"fourth-fake-key": "fourth-fake-digest",
		},
		result,
	)
}
