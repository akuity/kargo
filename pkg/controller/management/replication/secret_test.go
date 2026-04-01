package replication

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestComputeSecretHash(t *testing.T) {
	mk := func(labels, annotations map[string]string, data map[string][]byte) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
			Data:       data,
		}
	}

	t.Run("deterministic", func(t *testing.T) {
		s := mk(nil, nil, map[string][]byte{"k": []byte("v")})
		require.Equal(t, computeSecretHash(s), computeSecretHash(s))
		require.Len(t, computeSecretHash(s), 16)
	})

	t.Run("data key order independent", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"a": []byte("1"), "b": []byte("2")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"b": []byte("2"), "a": []byte("1")}))
		require.Equal(t, h1, h2)
	})

	t.Run("different data produces different hashes", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"k": []byte("v1")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"k": []byte("v2")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("empty secret", func(t *testing.T) {
		require.Len(t, computeSecretHash(&corev1.Secret{}), 16)
	})

	t.Run("label change produces different hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{"env": "prod"}, nil, nil))
		h2 := computeSecretHash(mk(map[string]string{"env": "staging"}, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("annotation change produces different hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{"owner": "team-a"}, nil))
		h2 := computeSecretHash(mk(nil, map[string]string{"owner": "team-b"}, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("replicate-to annotation excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{kargoapi.AnnotationKeyReplicateTo: "*"}, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("last-applied-configuration excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{lastAppliedConfigAnnotation: `{"big":"json"}`}, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("replication labels excluded from hash", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{
			kargoapi.LabelKeyReplicatedFrom: "src",
			kargoapi.LabelKeyReplicatedSHA:  "abc123",
		}, nil, nil))
		h2 := computeSecretHash(mk(nil, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("label order independent", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{"a": "1", "b": "2"}, nil, nil))
		h2 := computeSecretHash(mk(map[string]string{"b": "2", "a": "1"}, nil, nil))
		require.Equal(t, h1, h2)
	})

	t.Run("no key-value boundary collision", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"a": []byte("bc")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"ab": []byte("c")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no cross-pair boundary collision", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, nil, map[string][]byte{"a": []byte("bc"), "d": []byte("ef")}))
		h2 := computeSecretHash(mk(nil, nil, map[string][]byte{"a": []byte("bcd"), "e": []byte("f")}))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no label key-value boundary collision", func(t *testing.T) {
		h1 := computeSecretHash(mk(map[string]string{"a": "bc"}, nil, nil))
		h2 := computeSecretHash(mk(map[string]string{"ab": "c"}, nil, nil))
		require.NotEqual(t, h1, h2)
	})

	t.Run("no annotation key-value boundary collision", func(t *testing.T) {
		h1 := computeSecretHash(mk(nil, map[string]string{"a": "bc"}, nil))
		h2 := computeSecretHash(mk(nil, map[string]string{"ab": "c"}, nil))
		require.NotEqual(t, h1, h2)
	})
}
