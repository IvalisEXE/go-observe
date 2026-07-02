// Package miniolog wrap minio-go client buat log setiap operasi
// object storage: upload, download, delete — termasuk bucket, object key,
// ukuran file, durasi, dan error.
package miniolog

import (
	"context"
	"time"

	stacktrace "github.com/IvalisEXE/go-observe/errors"
	corelogger "github.com/IvalisEXE/go-observe/logger"
	"github.com/minio/minio-go/v7"
)

// Client wrap *minio.Client asli, method signature dibikin mirip
// biar gampang di-swap dari kode yang udah ada.
type Client struct {
	*minio.Client
}

func Wrap(c *minio.Client) *Client {
	return &Client{Client: c}
}

// FPutObject: upload file dari path lokal ke bucket.
func (c *Client) FPutObject(ctx context.Context, bucket, object, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	start := time.Now()
	info, err := c.Client.FPutObject(ctx, bucket, object, filePath, opts)
	c.logOp(ctx, "upload", bucket, object, info.Size, time.Since(start), err)
	return info, err
}

// PutObject: upload dari io.Reader (stream).
func (c *Client) PutObject(ctx context.Context, bucket, object string, reader interface {
	Read(p []byte) (n int, err error)
}, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	start := time.Now()
	info, err := c.Client.PutObject(ctx, bucket, object, reader, objectSize, opts)
	c.logOp(ctx, "upload", bucket, object, objectSize, time.Since(start), err)
	return info, err
}

// FGetObject: download object ke file lokal.
func (c *Client) FGetObject(ctx context.Context, bucket, object, filePath string, opts minio.GetObjectOptions) error {
	start := time.Now()
	err := c.Client.FGetObject(ctx, bucket, object, filePath, opts)
	c.logOp(ctx, "download", bucket, object, -1, time.Since(start), err)
	return err
}

// RemoveObject: hapus object dari bucket.
func (c *Client) RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error {
	start := time.Now()
	err := c.Client.RemoveObject(ctx, bucket, object, opts)
	c.logOp(ctx, "delete", bucket, object, -1, time.Since(start), err)
	return err
}

func (c *Client) logOp(ctx context.Context, op, bucket, object string, size int64, elapsed time.Duration, err error) {
	l := corelogger.FromContext(ctx)
	evt := l.Event(corelogger.EventMinIO).
		Str("operation", op).
		Str("bucket", bucket).
		Str("object", object).
		Dur("duration", elapsed)

	if size >= 0 {
		evt = evt.Int64("size_bytes", size)
	}

	if err != nil {
		evt.Str("error", err.Error()).
			Str("stack_trace", stacktrace.Capture(3)).
			Msg("minio operation failed")
		return
	}
	evt.Msg("minio operation completed")
}
