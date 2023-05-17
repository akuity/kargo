package option

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

// TestJsonCodec_MetaV1Time validates that the codec marshals v1.Time correctly into RFC3339 format.
func TestJsonCodec_MetaV1Time(t *testing.T) {
	expected := "2023-05-17T08:00:00Z"
	in, err := time.Parse(time.RFC3339, expected)
	require.NoError(t, err)

	firstSeen := metav1.NewTime(in)
	codec := newJSONCodec("json")
	data, err := codec.Marshal(&v1alpha1.EnvironmentState{
		FirstSeen: &firstSeen,
	})
	require.NoError(t, err)

	var got = struct {
		FirstSeen string `json:"firstSeen"`
	}{}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, expected, got.FirstSeen)
}
