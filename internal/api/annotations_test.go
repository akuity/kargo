package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestRefreshAnnotationValue(t *testing.T) {
	t.Run("has refresh annotation", func(t *testing.T) {
		result, ok := RefreshAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyRefresh: "foo",
		})
		require.True(t, ok)
		require.Equal(t, "foo", result)
	})

	t.Run("does not have refresh annotation", func(t *testing.T) {
		result, ok := RefreshAnnotationValue(nil)
		require.False(t, ok)
		require.Empty(t, result)
	})

	t.Run("has refresh annotation with empty value", func(t *testing.T) {
		result, ok := RefreshAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyRefresh: "",
		})
		require.True(t, ok)
		require.Empty(t, result)
	})
}

func TestReverifyAnnotationValue(t *testing.T) {
	t.Run("has reverify annotation with valid JSON", func(t *testing.T) {
		result, ok := ReverifyAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyReverify: `{"id":"foo"}`,
		})
		require.True(t, ok)
		require.Equal(t, "foo", result.ID)
	})

	t.Run("has reverify annotation with ID string", func(t *testing.T) {
		result, ok := ReverifyAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyReverify: "foo",
		})
		require.True(t, ok)
		require.Equal(t, "foo", result.ID)
	})

	t.Run("does not have reverify annotation", func(t *testing.T) {
		result, ok := ReverifyAnnotationValue(nil)
		require.False(t, ok)
		require.Nil(t, result)
	})

	t.Run("has reverify annotation with empty ID", func(t *testing.T) {
		result, ok := ReverifyAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: "",
		})
		require.False(t, ok)
		require.Nil(t, result)
	})
}

func TestAbortVerificationAnnotationValue(t *testing.T) {
	t.Run("has abort annotation with valid JSON", func(t *testing.T) {
		result, ok := AbortVerificationAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: `{"id":"foo"}`,
		})
		require.True(t, ok)
		require.Equal(t, "foo", result.ID)
	})

	t.Run("has abort annotation with ID string", func(t *testing.T) {
		result, ok := AbortVerificationAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: "foo",
		})
		require.True(t, ok)
		require.Equal(t, "foo", result.ID)
	})

	t.Run("does not have abort annotation", func(t *testing.T) {
		result, ok := AbortVerificationAnnotationValue(nil)
		require.False(t, ok)
		require.Nil(t, result)
	})

	t.Run("has abort annotation with empty ID", func(t *testing.T) {
		result, ok := AbortVerificationAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: "",
		})
		require.False(t, ok)
		require.Nil(t, result)
	})
}

func TestAbortPromotionAnnotationValue(t *testing.T) {
	t.Run("has abort annotation with valid JSON", func(t *testing.T) {
		result, ok := AbortPromotionAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: fmt.Sprintf(`{"action":"%s"}`, kargoapi.AbortActionTerminate),
		})
		require.True(t, ok)
		require.Equal(t, kargoapi.AbortActionTerminate, result.Action)
	})

	t.Run("has abort annotation with action string", func(t *testing.T) {
		result, ok := AbortPromotionAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: string(kargoapi.AbortActionTerminate),
		})
		require.True(t, ok)
		require.Equal(t, kargoapi.AbortActionTerminate, result.Action)
	})

	t.Run("does not have abort annotation", func(t *testing.T) {
		result, ok := AbortPromotionAnnotationValue(nil)
		require.False(t, ok)
		require.Nil(t, result)
	})

	t.Run("has abort annotation with empty action", func(t *testing.T) {
		result, ok := AbortPromotionAnnotationValue(map[string]string{
			kargoapi.AnnotationKeyAbort: "",
		})
		require.False(t, ok)
		require.Nil(t, result)
	})
}
