package image

import (
	"github.com/akuity/kargo/pkg/cache"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/types"
)

var imageCache cache.Cache[image]

func init() {
	var err error
	imageCache, err = cache.NewInMemoryCache[image](
		types.MustParseInt(os.GetEnv("MAX_IMAGE_CACHE_ENTRIES", "100000")),
	)
	if err != nil {
		panic("failed to initialize image cache: " + err.Error())
	}
}

func SetCache(c cache.Cache[image]) {
	imageCache = c
}
