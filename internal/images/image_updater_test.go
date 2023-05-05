package images

import (
	"testing"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are cribbed from Argo CD Image Updater and minimally altered to
// test our workarounds.

func TestGetNewestVersionFromTags(t *testing.T) {
	t.Run("Find the latest version without any constraint", func(t *testing.T) { // nolint: lll
		tagList := newImageTagList([]string{
			"0.1", "0.5.1", "0.9", "1.0", "1.0.1", "1.1.2", "2.0.3",
		})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Equal(t, "2.0.3", newTag)
	})

	t.Run("Find the latest version with a semver constraint on major", func(t *testing.T) { // nolint: lll
		tagList := newImageTagList([]string{
			"0.1", "0.5.1", "0.9", "1.0", "1.0.1", "1.1.2", "2.0.3",
		})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{Constraint: "^1.0"}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Equal(t, "1.1.2", newTag)
	},
	)

	t.Run("Find the latest version with a semver constraint on patch", func(t *testing.T) { // nolint: lll
		tagList := newImageTagList([]string{
			"0.1", "0.5.1", "0.9", "1.0", "1.0.1", "1.1.2", "2.0.3",
		})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{Constraint: "~1.0"}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", newTag)
	})

	t.Run("Find the latest version with a semver constraint that has no match", func(t *testing.T) { // nolint: lll
		tagList := newImageTagList([]string{"0.1", "0.5.1", "0.9", "2.0.3"})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{Constraint: "~1.0"}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		require.Empty(t, newTag)
	})

	t.Run("Find the latest version with a semver constraint that is invalid", func(t *testing.T) { // nolint: lll
		tagList := newImageTagList([]string{"0.1", "0.5.1", "0.9", "2.0.3"})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{Constraint: "latest"}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		assert.Error(t, err)
		assert.Empty(t, newTag)
	})

	t.Run("Find the latest version with no tags", func(t *testing.T) {
		tagList := newImageTagList([]string{})
		img := image.NewFromIdentifier("jannfis/test:1.0")
		vc := image.VersionConstraint{Constraint: "~1.0"}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Empty(t, newTag)
	})

	t.Run("Find the latest version using latest sortmode", func(t *testing.T) {
		tagList := newImageTagListWithDate([]string{
			"zz", "bb", "yy", "cc", "yy", "aa", "ll",
		})
		img := image.NewFromIdentifier("jannfis/test:bb")
		vc := image.VersionConstraint{Strategy: image.StrategyLatest}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Equal(t, "ll", newTag)
	})

	t.Run("Find the latest version using latest sortmode, invalid tags", func(t *testing.T) { // nolint: lll
		tagList := newImageTagListWithDate([]string{
			"zz", "bb", "yy", "cc", "yy", "aa", "ll",
		})
		img := image.NewFromIdentifier("jannfis/test:bb")
		vc := image.VersionConstraint{Strategy: image.StrategySemVer}
		newTag, err := getNewestVersionFromTags(img, &vc, tagList)
		require.NoError(t, err)
		assert.Empty(t, newTag)
	})

}

func newImageTagList(tagNames []string) *tag.ImageTagList {
	tagList := tag.NewImageTagList()
	for _, tagName := range tagNames {
		tagList.Add(tag.NewImageTag(tagName, time.Unix(0, 0), ""))
	}
	return tagList
}

func newImageTagListWithDate(tagNames []string) *tag.ImageTagList {
	tagList := tag.NewImageTagList()
	for i, t := range tagNames {
		tagList.Add(tag.NewImageTag(t, time.Unix(int64(i*5), 0), ""))
	}
	return tagList
}
