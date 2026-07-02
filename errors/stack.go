// Package errors nyediain helper buat capture stack trace,
// dipakai bareng logger.Error() di semua interceptor.
package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// Capture ngambil stack trace saat function ini dipanggil.
// skip = jumlah frame yang mau di-skip (biasanya 2: Capture itu sendiri + caller langsung).
func Capture(skip int) string {
	var sb strings.Builder
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		// skip frame internal go runtime yang ga penting
		if !strings.Contains(frame.File, "runtime/") {
			sb.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		}
		if !more {
			break
		}
	}
	return sb.String()
}

// CaptureFromRecover khusus dipanggil di dalam recover() block (panic handler).
func CaptureFromRecover() string {
	return Capture(3)
}
