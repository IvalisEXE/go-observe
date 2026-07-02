// Package gormlog implement gorm/logger.Interface supaya semua query
// (SELECT/INSERT/UPDATE/DELETE) otomatis ke-log terstruktur, termasuk
// durasi, jumlah row, dan stack trace kalau error/slow query.
package gormlog

import (
	"context"
	"errors"
	"time"

	stacktrace "github.com/IvalisEXE/go-observe/errors"
	corelogger "github.com/IvalisEXE/go-observe/logger"
	gormlogger "gorm.io/gorm/logger"
)

// Config buat threshold slow query dsb.
type Config struct {
	SlowQueryThreshold time.Duration // default 200ms
}

type GormLogger struct {
	cfg Config
}

// New bikin instance logger buat dipasang di gorm.Open(dsn, &gorm.Config{Logger: gormlog.New(...)})
func New(cfg Config) *GormLogger {
	if cfg.SlowQueryThreshold == 0 {
		cfg.SlowQueryThreshold = 200 * time.Millisecond
	}
	return &GormLogger{cfg: cfg}
}

func (g *GormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return g // level di-handle sendiri lewat context logger tiap request
}

func (g *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	corelogger.FromContext(ctx).Event(corelogger.EventDBQuery).Msgf(msg, args...)
}

func (g *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	corelogger.FromContext(ctx).Warn().Msgf(msg, args...)
}

func (g *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	corelogger.FromContext(ctx).Error().Msgf(msg, args...)
}

// Trace dipanggil GORM otomatis setiap abis eksekusi query.
func (g *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	l := corelogger.FromContext(ctx)

	evt := l.Event(corelogger.EventDBQuery).
		Str("sql", sql).
		Int64("rows_affected", rows).
		Dur("duration", elapsed)

	switch {
	case err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound):
		evt.Str("error", err.Error()).
			Str("stack_trace", stacktrace.Capture(3)).
			Msg("db query failed")
	case elapsed > g.cfg.SlowQueryThreshold:
		evt.Bool("slow_query", true).Msg("db query completed (slow)")
	default:
		evt.Msg("db query completed")
	}
}
