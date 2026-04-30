package replication

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestComputeConfigMapHash(t *testing.T) {
	mk := func(
		labels, annotations map[string]string,
		data map[string]string,
		binaryData map[string][]byte,
	) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Data:       data,
			BinaryData: binaryData,
		}
	}

	t.Run("deterministic", func(t *testing.T) {
		cm := mk(nil, nil, map[string]string{"k": "v"}, nil)
		require.Equal(t, computeConfigMapHash(cm), computeConfigMapHash(cm))
		require.Len(t, computeConfigMapHash(cm), 16)
	})

	t.Run("data key order independent", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"a": "1", "b": "2"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"b": "2", "a": "1"}, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("different data produces different hashes", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v1"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v2"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("binaryData included in hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"bin": []byte("data1")}))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"bin": []byte("data2")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("data and binaryData sections are distinct", func(t *testing.T) {
		// Same key+value in Data vs BinaryData should produce different hashes.
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"k": "v"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"k": []byte("v")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("empty configmap", func(t *testing.T) {
		require.Len(t, computeConfigMapHash(&corev1.ConfigMap{}), 16)
	})

	t.Run("label change produces different hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(map[string]string{"env": "prod"}, nil, nil, nil))
		h2 := computeConfigMapHash(mk(map[string]string{"env": "staging"}, nil, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("replicate-to annotation excluded from hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, map[string]string{kargoapi.AnnotationKeyReplicateTo: "*"}, nil, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("replication labels excluded from hash", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(map[string]string{
			kargoapi.LabelKeyReplicatedFrom: "src",
			kargoapi.LabelKeyReplicatedSHA:  "abc123",
		}, nil, nil, nil))
		h2 := computeConfigMapHash(mk(nil, nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("no data key-value boundary collision", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"a": "bc"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"ab": "c"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no data cross-pair boundary collision", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, map[string]string{"a": "bc", "d": "ef"}, nil))
		h2 := computeConfigMapHash(mk(nil, nil, map[string]string{"a": "bcd", "e": "f"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no binaryData key-value boundary collision", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"a": []byte("bc")}))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"ab": []byte("c")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no binaryData cross-pair boundary collision", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"a": []byte("bc"), "d": []byte("ef")}))
		h2 := computeConfigMapHash(mk(nil, nil, nil, map[string][]byte{"a": []byte("bcd"), "e": []byte("f")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no label key-value boundary collision", func(t *testing.T) {
		h1 := computeConfigMapHash(mk(map[string]string{"a": "bc"}, nil, nil, nil))
		h2 := computeConfigMapHash(mk(map[string]string{"ab": "c"}, nil, nil, nil))
		require.NotEqual(t, h1, h2)
	})
}
