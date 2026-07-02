# corelogger

Central logging library buat semua squad. Structured JSON log (via `zerolog`),
gampang ditarik ke ELK/Loki/Grafana. Semua log punya `request_id` yang sama
dari masuk request sampe keluar ke DB/Redis/API/MinIO, jadi satu request bisa
di-trace penuh cukup filter `request_id`.

## Install

```bash
go get github.com/IvalisEXE/go-observe
```

## Fitur

| Modul | Fungsi |
|---|---|
| `logger` | core logger + context propagation |
| `httpmw` | Gin middleware: log incoming request/response + panic recovery |
| `gormlog` | GORM logger: log tiap query, durasi, slow query, error |
| `redislog` | go-redis hook: log tiap command Redis |
| `httpclient` | http.Client wrapper: log tiap outgoing API call (hit API) |
| `miniolog` | wrapper minio-go: log upload/download/delete object |
| `errors` | capture stack trace, otomatis nempel di log kalau ada error |

## Quick Start

Liat `example/main.go` buat contoh lengkap wiring semua modul.

```go
corelogger.Init(corelogger.Config{
    ServiceName: "order-service",
    Env:         "production",
    Level:       "info",
    Pretty:      false, // JSON di prod
})
```

Semua log otomatis punya field:
```json
{
  "level": "info",
  "service": "order-service",
  "env": "production",
  "request_id": "b1e2...",
  "type": "http_request",
  "method": "GET",
  "path": "/orders/123",
  "status": 200,
  "latency": "12ms",
  "time": "2026-07-02T10:00:00Z"
}
```

## Contoh log per aktivitas

**HTTP request masuk** → `type: http_request`
**Hit API keluar** → `type: external_api`
**Query DB** → `type: db_query`
**Command Redis** → `type: redis`
**Upload/download MinIO** → `type: minio`
**Panic** → `type: panic` (dengan `stack_trace`)

## Aturan buat squad

1. **Wajib** panggil `corelogger.Init()` sekali di `main.go`.
2. **Wajib** pasang `httpmw.RequestLogger()` sebagai middleware paling atas.
3. Kalau butuh logging manual di service/repo layer, pakai
   `corelogger.FromContext(ctx)` — jangan bikin logger baru sendiri,
   biar `request_id` tetep nyambung.
4. Field sensitif (password, token, otp, dll) **otomatis di-mask**, tapi
   kalau ada field baru yang sensitif, tambahin ke `SensitiveFields` di config.
5. Jangan log body yang gede banget (file upload dsb) — `MaxBodyLogSize`
   udah handle truncate otomatis, default 4KB.

## Roadmap (belum termasuk sekarang)

- [ ] Kafka/RabbitMQ consumer & producer logging
- [ ] gRPC interceptor
- [ ] OpenTelemetry trace_id integration (biar nyambung ke Jaeger/Tempo)
- [ ] Sampling buat high-traffic endpoint
