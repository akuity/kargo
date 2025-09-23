package ecr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tokenCacheKey(t *testing.T) {
	testCases := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "single part",
			parts: []string{"region1"},
			want:  "7507acda9c58034d4f38545edd121b4c8572483cbc5c7dc40f3daa2c74d8430a",
		},
		{
			name:  "multiple parts",
			parts: []string{"region1", "key1", "secret1"},
			want:  "559495d4cca6055810e755d40dfdeeb1aa0a937f3030463be970b9cd2d586002",
		},
		{
			name:  "no parts",
			parts: []string{},
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100000; i++ {
				result := tokenCacheKey(tt.parts...)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}
