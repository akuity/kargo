package glob

import (
	"context"

	"github.com/gobwas/glob"

	"github.com/akuity/kargo/internal/logging"
)

func Match(pattern, text string, separators ...rune) bool {
	compiledGlob, err := glob.Compile(pattern, separators...)
	if err != nil {
		logging.LoggerFromContext(context.TODO()).Error(
			err, "failed to compile pattern", "pattern", pattern,
		)
		return false
	}
	return compiledGlob.Match(text)
}
