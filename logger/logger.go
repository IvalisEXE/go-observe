// Package logger adalah central logger untuk semua service.
// Semua log di-output dalam format JSON (structured) supaya gampang
// ditarik ke ELK / Loki / Datadog dsb.
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// EventType dipakai buat nge-tag jenis aktivitas di field "type",
// biar gampang di-filter di Kibana/Grafana.
type EventType string

const (
	EventHTTPRequest EventType = "http_request"  // incoming request masuk ke service kita
	EventExternalAPI EventType = "external_api"  // outgoing call ke service lain
	EventDBQuery      EventType = "db_query"
	EventRedis        EventType = "redis"
	EventMinIO        EventType = "minio"
	EventPanic        EventType = "panic"
	EventGeneric      EventType = "log"
)

// Config buat inisialisasi logger sekali di awal aplikasi (main.go).
type Config struct {
	ServiceName string // wajib, contoh: "order-service"
	Env         string // production | staging | development
	Level       string // debug | info | warn | error
	Output      io.Writer // default os.Stdout, bisa diarahkan ke file/pipe
	Pretty      bool      // true = human-readable console (dev only), false = JSON (prod)
}

// Logger adalah wrapper tipis di atas zerolog.Logger.
// Semua helper library lain (httpmw, gormlog, redislog, dst) mengandalkan
// tipe ini supaya konsisten formatnya.
type Logger struct {
	zl zerolog.Logger
}

var std *Logger // instance global, di-set lewat Init()

// Init menginisialisasi logger global. Panggil ini sekali di main.go
// sebelum service jalan.
func Init(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano

	var w io.Writer = cfg.Output
	if cfg.Pretty {
		w = zerolog.ConsoleWriter{Out: cfg.Output, TimeFormat: time.RFC3339}
	}

	level := parseLevel(cfg.Level)

	zl := zerolog.New(w).
		Level(level).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("env", cfg.Env).
		Caller().
		Logger()

	std = &Logger{zl: zl}
	return std
}

// L mengembalikan instance logger global. Panggil setelah Init().
func L() *Logger {
	if std == nil {
		// fallback biar ga nil-panic kalau ada yang lupa Init()
		std = Init(Config{ServiceName: "unknown", Env: "development", Level: "info", Pretty: true})
	}
	return std
}

func parseLevel(lvl string) zerolog.Level {
	switch lvl {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithFields bikin child logger dengan field tambahan yang nempel terus
// di setiap log berikutnya (misal request_id, user_id).
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zl.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{zl: ctx.Logger()}
}

// Event mulai satu log entry terstruktur dengan "type" tertentu.
// Ini yang dipakai internal oleh httpmw/gormlog/redislog/miniolog/httpclient.
func (l *Logger) Event(evt EventType) *zerolog.Event {
	return l.zl.Info().Str("type", string(evt))
}

func (l *Logger) Info() *zerolog.Event  { return l.zl.Info() }
func (l *Logger) Debug() *zerolog.Event { return l.zl.Debug() }
func (l *Logger) Warn() *zerolog.Event  { return l.zl.Warn() }
func (l *Logger) Error() *zerolog.Event { return l.zl.Error() }
func (l *Logger) Fatal() *zerolog.Event { return l.zl.Fatal() }

// Raw dipakai kalau ada squad yang butuh akses langsung ke zerolog.Logger.
func (l *Logger) Raw() zerolog.Logger { return l.zl }
