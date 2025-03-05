package option

import (
	"runtime"
	"strconv"
	"strings"
)

const defaultStackLength = 32

func takeStacktrace(n, skip uint32) string {
	var builder strings.Builder
	pcs := make([]uintptr, n)

	// +2 to exclude runtime.Callers and takeStacktrace
	numFrames := runtime.Callers(2+int(skip), pcs)
	if numFrames == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:numFrames])
	for i := 0; ; i++ {
		frame, more := frames.Next()
		if i != 0 {
			_ = builder.WriteByte('\n')
		}
		_, _ = builder.WriteString(frame.Function)
		_ = builder.WriteByte('\n')
		_ = builder.WriteByte('\t')
		_, _ = builder.WriteString(frame.File)
		_ = builder.WriteByte(':')
		_, _ = builder.WriteString(strconv.Itoa(frame.Line))
		if !more {
			break
		}
	}
	return builder.String()
}
