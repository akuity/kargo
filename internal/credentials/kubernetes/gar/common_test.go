package gar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tokenCacheKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{
			key:  "test123",
			want: "ecd71870d1963316a97e3ac3408c9835ad8cf0f3c1bc703527c30265534f75ae",
		},
		{
			key:  "123test",
			want: "a8327d4a49d4631631d090a72297d8d749337a30e6eb0416bd3655b71e36481b",
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			for i := 0; i < 100000; i++ {
				assert.Equal(t, tt.want, tokenCacheKey(tt.key))
			}
		})
	}
}
