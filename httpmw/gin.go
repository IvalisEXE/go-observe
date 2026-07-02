// Package httpmw nyediain middleware Gin buat log setiap incoming
// request & response secara otomatis, termasuk body (dengan masking
// field sensitif) dan stack trace kalau ada panic.
package httpmw

import (
	"bytes"
	"io"
	"time"

	"github.com/IvalisEXE/go-observe/errors"
	"github.com/IvalisEXE/go-observe/logger"
	"github.com/gin-gonic/gin"
)

// Options buat konfigurasi middleware.
type Options struct {
	// MaxBodyLogSize batas ukuran body (bytes) yang di-log, sisanya di-truncate.
	// Default 4096 kalau 0.
	MaxBodyLogSize int
	// SkipPaths, misal health check, ga usah di-log biar ga berisik.
	SkipPaths []string
	// SensitiveFields nama field JSON yang bakal di-mask jadi "***",
	// contoh: password, token, otp, card_number.
	SensitiveFields []string
}

func defaultOptions(o Options) Options {
	if o.MaxBodyLogSize == 0 {
		o.MaxBodyLogSize = 4096
	}
	if len(o.SensitiveFields) == 0 {
		o.SensitiveFields = []string{"password", "token", "access_token", "refresh_token", "otp", "card_number", "cvv", "pin"}
	}
	return o
}

// responseWriter nangkep body response tanpa ganggu response asli ke client.
type responseWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.buf.Write(b) // simpan copy buat di-log
	return w.ResponseWriter.Write(b)
}

// RequestLogger dipasang paling atas: router.Use(httpmw.RequestLogger(opts))
func RequestLogger(opts Options) gin.HandlerFunc {
	opts = defaultOptions(opts)
	skip := make(map[string]bool, len(opts.SkipPaths))
	for _, p := range opts.SkipPaths {
		skip[p] = true
	}

	return func(c *gin.Context) {
		if skip[c.FullPath()] {
			c.Next()
			return
		}

		start := time.Now()

		// request_id: pakai dari header kalau upstream udah kirim (buat trace lintas service),
		// kalau ga ada, generate baru.
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = logger.NewRequestID()
		}
		c.Writer.Header().Set("X-Request-ID", reqID)

		ctx := logger.WithRequestID(c.Request.Context(), reqID)
		reqLogger := logger.L().WithFields(map[string]interface{}{
			"request_id": reqID,
		})
		ctx = logger.WithLogger(ctx, reqLogger)
		c.Request = c.Request.WithContext(ctx)

		// baca & simpan ulang request body (biar handler asli tetep bisa baca)
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// wrap response writer buat nangkep body response
		rw := &responseWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = rw

		defer func() {
			if rec := recover(); rec != nil {
				reqLogger.Event(logger.EventPanic).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Interface("recover", rec).
					Str("stack_trace", errors.CaptureFromRecover()).
					Msg("panic recovered in http handler")

				c.AbortWithStatus(500)
			}
		}()

		c.Next() // jalanin handler asli

		latency := time.Since(start)
		status := c.Writer.Status()

		evt := reqLogger.Event(logger.EventHTTPRequest).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int("status", status).
			Dur("latency", latency).
			Str("request_body", truncate(string(maskJSON(reqBody, opts.SensitiveFields)), opts.MaxBodyLogSize)).
			Str("response_body", truncate(string(maskJSON(rw.buf.Bytes(), opts.SensitiveFields)), opts.MaxBodyLogSize))

		if len(c.Errors) > 0 {
			evt = evt.Str("errors", c.Errors.String()).Str("stack_trace", errors.Capture(2))
		}

		if status >= 500 {
			evt.Msg("http request completed with server error")
		} else if status >= 400 {
			evt.Msg("http request completed with client error")
		} else {
			evt.Msg("http request completed")
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
