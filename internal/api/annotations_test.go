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

func TestHasMigrationAnnotationValue(t *testing.T) {
	mockObj := &kargoapi.Project{}

	t.Run("has migration annotation with migration type true", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `{"migration1":true,"migration2":false}`,
		})
		result := HasMigrationAnnotationValue(obj, "migration1")
		require.True(t, result)
	})

	t.Run("has migration annotation with migration type false", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `{"migration1":true,"migration2":false}`,
		})
		result := HasMigrationAnnotationValue(obj, "migration2")
		require.False(t, result)
	})

	t.Run("has migration annotation but migration type not present", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `{"migration1":true}`,
		})
		result := HasMigrationAnnotationValue(obj, "migration2")
		require.False(t, result)
	})

	t.Run("does not have migration annotation", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			"other-annotation": "value",
		})
		result := HasMigrationAnnotationValue(obj, "migration1")
		require.False(t, result)
	})

	t.Run("annotations map is nil", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(nil)
		result := HasMigrationAnnotationValue(obj, "migration1")
		require.False(t, result)
	})

	t.Run("has migration annotation with invalid JSON", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `invalid-json`,
		})
		result := HasMigrationAnnotationValue(obj, "migration1")
		require.False(t, result)
	})

	t.Run("has migration annotation with empty JSON object", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `{}`,
		})
		result := HasMigrationAnnotationValue(obj, "migration1")
		require.False(t, result)
	})
}

func TestAddMigrationAnnotationValue(t *testing.T) {
	mockObj := &kargoapi.Project{}

	t.Run("adds migration to empty annotations map", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(make(map[string]string))
		AddMigrationAnnotationValue(obj, "migration1")

		result := HasMigrationAnnotationValue(obj, "migration1")
		require.True(t, result)
		require.Contains(t, obj.GetAnnotations(), kargoapi.AnnotationKeyMigrated)
	})

	t.Run("adds migration to existing annotations without migration annotation", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			"other-annotation": "value",
		})
		AddMigrationAnnotationValue(obj, "migration1")

		result := HasMigrationAnnotationValue(obj, "migration1")
		require.True(t, result)
		require.Contains(t, obj.GetAnnotations(), kargoapi.AnnotationKeyMigrated)
		require.Equal(t, "value", obj.GetAnnotations()["other-annotation"])
	})

	t.Run("adds migration to existing migration annotation", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `{"migration1":true}`,
		})
		AddMigrationAnnotationValue(obj, "migration2")

		result1 := HasMigrationAnnotationValue(obj, "migration1")
		result2 := HasMigrationAnnotationValue(obj, "migration2")
		require.True(t, result1)
		require.True(t, result2)
	})

	t.Run("adds migration when existing annotation has invalid JSON", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(map[string]string{
			kargoapi.AnnotationKeyMigrated: `invalid-json`,
		})
		AddMigrationAnnotationValue(obj, "migration1")

		result := HasMigrationAnnotationValue(obj, "migration1")
		require.True(t, result)
	})

	t.Run("adds same migration type multiple times", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(make(map[string]string))
		AddMigrationAnnotationValue(obj, "migration1")
		AddMigrationAnnotationValue(obj, "migration1")

		result := HasMigrationAnnotationValue(obj, "migration1")
		require.True(t, result)
	})

	t.Run("adds multiple different migrations", func(t *testing.T) {
		obj := mockObj.DeepCopy()
		obj.SetAnnotations(make(map[string]string))
		AddMigrationAnnotationValue(obj, "migration1")
		AddMigrationAnnotationValue(obj, "migration2")
		AddMigrationAnnotationValue(obj, "migration3")

		result1 := HasMigrationAnnotationValue(obj, "migration1")
		result2 := HasMigrationAnnotationValue(obj, "migration2")
		result3 := HasMigrationAnnotationValue(obj, "migration3")
		require.True(t, result1)
		require.True(t, result2)
		require.True(t, result3)
	})
}
