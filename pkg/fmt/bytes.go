package fmt

import "fmt"

// FormatByteCount formats a byte count using the largest appropriate IEC
// binary unit (KiB, MiB, GiB).
func FormatByteCount(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}
