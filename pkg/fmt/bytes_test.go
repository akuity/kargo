package fmt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatByteCount(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  int64
		expect string
	}{
		{
			name:   "zero bytes",
			input:  0,
			expect: "0 bytes",
		},
		{
			name:   "small byte count",
			input:  512,
			expect: "512 bytes",
		},
		{
			name:   "exactly 1 KiB",
			input:  1 << 10,
			expect: "1.0 KiB",
		},
		{
			name:   "fractional KiB",
			input:  1536, // 1.5 KiB
			expect: "1.5 KiB",
		},
		{
			name:   "exactly 1 MiB",
			input:  1 << 20,
			expect: "1.0 MiB",
		},
		{
			name:   "fractional MiB",
			input:  3 * (1 << 19), // 1.5 MiB
			expect: "1.5 MiB",
		},
		{
			name:   "exactly 1 GiB",
			input:  1 << 30,
			expect: "1.0 GiB",
		},
		{
			name:   "large GiB value",
			input:  6 * (1 << 30), // 6 GiB
			expect: "6.0 GiB",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expect, FormatByteCount(tc.input))
		})
	}
}
