package kargo

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestNewPromotion(t *testing.T) {
	const (
		testFreight          = "f08b2e72c9b2b7b263da6d55f9536e49b5ce972c"
		veryLongResourceName = "the-kubernetes-maximum-length-of-a-label-value-is-only-sixty-" +
			"three-characters-meanwhile-the-maximum-length-of-a-kubernetes-resource-name-" +
			"is-two-hundred-and-fifty-three-characters-but-this-string-is-two-hundred-" +
			"and-thirty-seven-characters"
	)
	t.Parallel()
	testCases := []struct {
		name       string
		template   kargoapi.PromotionTemplate
		namespace  string
		stage      string
		freight    string
		assertions func(*testing.T, kargoapi.Promotion)
	}{
		{
			name: "Promote stage",
			template: kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Vars: []kargoapi.PromotionVariable{
						{
							Name:  "foo",
							Value: "bar",
						},
					},
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "test-step",
							As: "test-step",
						},
					},
				},
			},
			namespace: "kargo-demo",
			stage:     "test",
			freight:   testFreight,
			assertions: func(t *testing.T, promo kargoapi.Promotion) {
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, "test", parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
			},
		},
		{
			name: "Promote stage with very long name",
			namespace: "kargo-demo",
			stage: veryLongResourceName,
			freight: testFreight,
			assertions: func(t *testing.T, promo kargoapi.Promotion) {
				require.Len(t, promo.Name, 253)
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, veryLongResourceName[0:maxStageNamePrefixLength], parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			promo := NewPromotion(context.TODO(), tc.template, tc.namespace, tc.stage, tc.freight)
			require.Equal(t, tc.freight, promo.Spec.Freight)
			require.Equal(t, tc.stage, promo.Spec.Stage)
			require.Equal(t, tc.freight, promo.Spec.Freight)
			require.LessOrEqual(t, len(promo.Name), 253)
			tc.assertions(t, promo)
		})
	}
}

func TestPromoPhaseChanged_Update(t *testing.T) {
	tests := []struct {
		name      string
		oldObject *kargoapi.Promotion
		newObject *kargoapi.Promotion
		want      bool
	}{
		{
			name:      "no old or new object",
			oldObject: nil,
			newObject: nil,
			want:      false,
		},
		{
			name:      "no old object",
			oldObject: nil,
			newObject: &kargoapi.Promotion{},
			want:      false,
		},
		{
			name:      "no new object",
			oldObject: &kargoapi.Promotion{},
			newObject: nil,
			want:      false,
		},
		{
			name: "no phase change",
			oldObject: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			newObject: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			want: false,
		},
		{
			name: "phase changed",
			oldObject: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			newObject: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseErrored,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPromoPhaseChangedPredicate(logging.NewLogger(logging.InfoLevel))
			require.Equal(t, tt.want, p.Update(event.TypedUpdateEvent[*kargoapi.Promotion]{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}))
		})
	}
}

func TestRefreshRequested_Update(t *testing.T) {
	tests := []struct {
		name      string
		oldObject client.Object
		newObject client.Object
		want      bool
	}{
		{
			name:      "no old or new object",
			oldObject: nil,
			newObject: nil,
			want:      false,
		},
		{
			name:      "no old object",
			oldObject: nil,
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "no new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			newObject: nil,
			want:      false,
		},
		{
			name: "no refresh annotation",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other": "annotation",
					},
				},
			},
			want: false,
		},
		{
			name: "refresh annotation set on new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			want: true,
		},
		{
			name: "refresh annotation removed from new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "refresh annotation value changed",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "bar",
					},
				},
			},
			want: true,
		},
		{
			name: "refresh annotation value equal",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "foo",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := RefreshRequested{}
			require.Equal(t, tt.want, p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}))
		})
	}
}

func TestReverifyRequested_Update(t *testing.T) {
	tests := []struct {
		name      string
		oldObject client.Object
		newObject client.Object
		want      bool
	}{
		{
			name:      "no old or new object",
			oldObject: nil,
			newObject: nil,
			want:      false,
		},
		{
			name:      "no old object",
			oldObject: nil,
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "no new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "foo",
					},
				},
			},
			newObject: nil,
			want:      false,
		},
		{
			name: "no reverify annotation",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other": "annotation",
					},
				},
			},
			want: false,
		},
		{
			name: "reverify annotation set on new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "foo",
					},
				},
			},
			want: true,
		},
		{
			name: "reverify annotation removed from new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: "foo",
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "empty reverify annotation value",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "",
					},
				},
			},
			want: false,
		},
		{
			name: "reverify annotation ID changed",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "bar",
						}).String(),
					},
				},
			},
			want: true,
		},
		{
			name: "reverify annotation actor changed with same ID",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "foo",
							Actor: "fake-actor",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID:    "foo",
							Actor: "real-actor",
						}).String(),
					},
				},
			},
			want: false,
		},
		{
			name: "reverify annotation ID equal",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ReverifyRequested{}
			require.Equal(t, tt.want, p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}))
		})
	}
}

func TestVerificationAbortRequested_Update(t *testing.T) {
	tests := []struct {
		name      string
		oldObject client.Object
		newObject client.Object
		want      bool
	}{
		{
			name:      "no old or new object",
			oldObject: nil,
			newObject: nil,
			want:      false,
		},
		{
			name:      "no old object",
			oldObject: nil,
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "no new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			newObject: nil,
			want:      false,
		},
		{
			name: "no abort annotation",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other": "annotation",
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation set on new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			want: true,
		},
		{
			name: "abort annotation removed from new object",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "empty abort annotation value",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "",
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation ID changed",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "bar",
						}).String(),
					},
				},
			},
			want: true,
		},
		{
			name: "abort annotation actor changed with same ID",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "foo",
							Actor: "fake-actor",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID:    "foo",
							Actor: "real-actor",
						}).String(),
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation ID equal",
			oldObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.VerificationRequest{
							ID: "foo",
						}).String(),
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := VerificationAbortRequested{}
			require.Equal(t, tt.want, p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}))
		})
	}
}

func TestPromotionAbortRequested_Update(t *testing.T) {
	tests := []struct {
		name      string
		oldObject client.Object
		newObject client.Object
		want      bool
	}{
		{
			name:      "no old or new object",
			oldObject: nil,
			newObject: nil,
			want:      false,
		},
		{
			name:      "no old object",
			oldObject: nil,
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "no new object",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			newObject: nil,
			want:      false,
		},
		{
			name: "no abort annotation",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other": "annotation",
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation set on new object",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			want: true,
		},
		{
			name: "abort annotation removed from new object",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "foo",
					},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "empty abort annotation value",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "",
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation action changed",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "bar",
						}).String(),
					},
				},
			},
			want: true,
		},
		{
			name: "abort annotation actor changed with same ID",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
							Actor:  "fake-actor",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
							Actor:  "real-actor",
						}).String(),
					},
				},
			},
			want: false,
		},
		{
			name: "abort annotation ID equal",
			oldObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
						}).String(),
					},
				},
			},
			newObject: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: (&kargoapi.AbortPromotionRequest{
							Action: "foo",
						}).String(),
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PromotionAbortRequested{}
			require.Equal(t, tt.want, p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}))
		})
	}
}
