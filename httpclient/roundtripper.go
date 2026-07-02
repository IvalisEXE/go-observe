// Package httpclient nyediain http.RoundTripper wrapper buat log
// setiap outgoing call ke API/service lain (hit API), termasuk
// propagate request_id via header X-Request-ID biar bisa di-trace
// lintas service.
package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"time"

	stacktrace "github.com/IvalisEXE/go-observe/errors"
	corelogger "github.com/IvalisEXE/go-observe/logger"
)

// Options sama kayak httpmw, biar konsisten body logging & masking-nya.
type Options struct {
	MaxBodyLogSize  int
	SensitiveFields []string
}

type loggingTransport struct {
	base http.RoundTripper
	opts Options
}

// New bikin *http.Client yang otomatis log setiap request keluar.
// Contoh pakai:
//
//	client := httpclient.New(http.DefaultTransport, httpclient.Options{})
//	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
//	client.Do(req)
func New(base http.RoundTripper, opts Options) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}
	if opts.MaxBodyLogSize == 0 {
		opts.MaxBodyLogSize = 4096
	}
	return &http.Client{Transport: &loggingTransport{base: base, opts: opts}}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	l := corelogger.FromContext(ctx)

	// propagate request_id ke service tujuan
	if reqID := corelogger.RequestIDFromContext(ctx); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}

	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	elapsed := time.Since(start)

	evt := l.Event(corelogger.EventExternalAPI).
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Dur("duration", elapsed).
		Str("request_body", truncate(string(reqBody), t.opts.MaxBodyLogSize))

	if err != nil {
		evt.Str("error", err.Error()).
			Str("stack_trace", stacktrace.Capture(2)).
			Msg("external api call failed")
		return resp, err
	}

	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	evt = evt.Int("status", resp.StatusCode).
		Str("response_body", truncate(string(respBody), t.opts.MaxBodyLogSize))

	if resp.StatusCode >= 500 {
		evt.Str("stack_trace", stacktrace.Capture(2)).Msg("external api call returned server error")
	} else if resp.StatusCode >= 400 {
		evt.Msg("external api call returned client error")
	} else {
		evt.Msg("external api call completed")
	}

	return resp, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
