package logger

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyLogger    ctxKey = "logger"
)

// NewRequestID generate request id baru (dipanggil di middleware HTTP).
func NewRequestID() string {
	return uuid.NewString()
}

// WithRequestID nyimpen request_id ke context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestIDFromContext ambil request_id dari context. Kosong kalau ga ada.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// WithLogger nyimpen logger (yang sudah ke-attach request_id dkk) ke context,
// supaya bisa dipanggil ulang di layer manapun (service, repository, dst)
// tanpa perlu passing logger manual sebagai parameter.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, l)
}

// FromContext ambil logger dari context. Kalau belum pernah di-set,
// fallback ke logger global L().
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*Logger); ok && l != nil {
		return l
	}
	return L()
}
