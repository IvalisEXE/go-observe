// Package redislog implement redis.Hook (go-redis v9) buat log
// setiap command Redis yang dieksekusi: cmd, args, durasi, error.
package redislog

import (
	"context"
	"net"
	"time"

	stacktrace "github.com/IvalisEXE/go-observe/errors"
	corelogger "github.com/IvalisEXE/go-observe/logger"
	"github.com/redis/go-redis/v9"
)

type Hook struct{}

// New bikin hook baru, pasang ke client: rdb.AddHook(redislog.New())
func New() *Hook { return &Hook{} }

func (h *Hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *Hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		elapsed := time.Since(start)

		l := corelogger.FromContext(ctx)
		evt := l.Event(corelogger.EventRedis).
			Str("command", cmd.Name()).
			Interface("args", maskRedisArgs(cmd.Args())).
			Dur("duration", elapsed)

		if err != nil && err != redis.Nil {
			evt.Str("error", err.Error()).
				Str("stack_trace", stacktrace.Capture(3)).
				Msg("redis command failed")
		} else {
			evt.Msg("redis command executed")
		}
		return err
	}
}

func (h *Hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		elapsed := time.Since(start)

		names := make([]string, 0, len(cmds))
		for _, c := range cmds {
			names = append(names, c.Name())
		}

		l := corelogger.FromContext(ctx)
		evt := l.Event(corelogger.EventRedis).
			Strs("commands", names).
			Int("pipeline_size", len(cmds)).
			Dur("duration", elapsed)

		if err != nil && err != redis.Nil {
			evt.Str("error", err.Error()).Str("stack_trace", stacktrace.Capture(3)).Msg("redis pipeline failed")
		} else {
			evt.Msg("redis pipeline executed")
		}
		return err
	}
}

// maskRedisArgs nyensor value command yang sering isinya sensitif,
// misal SET session:xxx <token>. Cukup ambil arg pertama (nama key) aja,
// sisanya di-redact biar aman.
func maskRedisArgs(args []interface{}) []interface{} {
	if len(args) <= 1 {
		return args
	}
	out := make([]interface{}, len(args))
	out[0] = args[0]
	for i := 1; i < len(args); i++ {
		out[i] = "***"
	}
	return out
}
