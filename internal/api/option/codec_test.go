package option

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestMarshal(t *testing.T) {
	testCases := map[string]struct {
		input    proto.Message
		expected string
	}{
		"google proto message": {
			input: &svcv1alpha1.GetVersionInfoResponse{
				VersionInfo: &svcv1alpha1.VersionInfo{
					Version:      "devel+unknown.dirty",
					GitTreeDirty: true,
				},
			},
			expected: `{
	"versionInfo":{
		"version": "devel+unknown.dirty",
		"gitTreeDirty":true
	}
}`,
		},
		"google proto message mixed with gogo proto message": {
			input: &svcv1alpha1.GetStageResponse{
				Stage: &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-stage",
						CreationTimestamp: metav1.NewTime(
							time.Date(2024, 2, 21, 15, 0, 0, 0, time.UTC),
						),
					},
					Spec: &kargoapi.StageSpec{
						Subscriptions: &kargoapi.Subscriptions{
							Warehouse: "fake-warehouse",
						},
					},
					Status: kargoapi.StageStatus{},
				},
			},
			expected: `{
	"stage": {
		"metadata": {
			"namespace": "fake-namespace",
			"name": "fake-stage",
			"creationTimestamp": "2024-02-21T15:00:00Z"
		},
		"spec": {
			"subscriptions": {
				"warehouse": "fake-warehouse"
			}
		},
		"status": {}
	}
}`,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			codec := newJSONCodec("json")
			actual, err := codec.Marshal(tc.input)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(actual))
		})
	}
}

func TestUnmarshal(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected proto.Message
	}{
		"google proto message": {
			input: `{
	"versionInfo":{
		"version": "devel+unknown.dirty",
		"gitTreeDirty":true
	}
}`,
			expected: &svcv1alpha1.GetVersionInfoResponse{
				VersionInfo: &svcv1alpha1.VersionInfo{
					Version:      "devel+unknown.dirty",
					GitTreeDirty: true,
				},
			},
		},
		"google proto message mixed with gogo proto message": {
			input: `{
	"stage": {
		"metadata": {
			"namespace": "fake-namespace",
			"name": "fake-stage",
			"creationTimestamp": "2024-02-21T15:00:00Z"
		},
		"spec": {
			"subscriptions": {
				"warehouse": "fake-warehouse"
			}
		},
		"status": {}
	}
}`,
			expected: &svcv1alpha1.GetStageResponse{
				Stage: &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-namespace",
						Name:      "fake-stage",
						CreationTimestamp: metav1.NewTime(
							time.Date(2024, 2, 21, 15, 0, 0, 0, time.UTC),
						),
					},
					Spec: &kargoapi.StageSpec{
						Subscriptions: &kargoapi.Subscriptions{
							Warehouse: "fake-warehouse",
						},
					},
					Status: kargoapi.StageStatus{},
				},
			},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := proto.Clone(tc.expected)
			proto.Reset(actual)

			codec := newJSONCodec("json")
			require.NoError(t, codec.Unmarshal([]byte(tc.input), actual))
			require.Empty(t, cmp.Diff(tc.expected, actual, protocmp.Transform()))
		})
	}
}
