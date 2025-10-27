package image

import "github.com/akuity/kargo/pkg/cache"

var imageCache cache.Cache[Image]

func init() {
	var err error
	imageCache, err = cache.NewInMemoryCache[Image](100000)
	if err != nil {
		panic("failed to initialize image cache: " + err.Error())
	}
}

func SetCache(c cache.Cache[Image]) {
	imageCache = c
}
