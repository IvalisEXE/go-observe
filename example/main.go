package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/IvalisEXE/go-observe/gormlog"
	"github.com/IvalisEXE/go-observe/httpclient"
	"github.com/IvalisEXE/go-observe/httpmw"
	corelogger "github.com/IvalisEXE/go-observe/logger"
	"github.com/IvalisEXE/go-observe/miniolog"
	"github.com/IvalisEXE/go-observe/redislog"
)

func main() {
	env := os.Getenv("APP_ENV") // "production" / "development"

	// 1. Init logger sekali di awal
	corelogger.Init(corelogger.Config{
		ServiceName: "order-service",
		Env:         env,
		Level:       "info",
		Pretty:      env == "development", // JSON di prod, pretty di lokal
	})

	// 2. GORM: pasang gormlog sebagai Logger
	db, err := gorm.Open(postgres.Open("host=localhost port=5432 user=youruser dbname=yourdb password=yourpassword sslmode=disable"), &gorm.Config{
		Logger: gormlog.New(gormlog.Config{SlowQueryThreshold: 200 * time.Millisecond}),
	})
	if err != nil {
		corelogger.L().Fatal().Err(err).Msg("failed connect db")
	}
	_ = db

	// 3. Redis: pasang hook
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	rdb.AddHook(redislog.New())

	// 4. HTTP client buat hit API service lain, otomatis ke-log
	apiClient := httpclient.New(http.DefaultTransport, httpclient.Options{})
	_ = apiClient // pakai: apiClient.Do(req)

	// 5. MinIO client, wrap biar setiap upload/download ke-log
	minioClient, err := minio.New("minio:9000", &minio.Options{
		Creds: credentials.NewStaticV4("access", "secret", ""),
	})
	if err != nil {
		corelogger.L().Fatal().Err(err).Msg("failed connect minio")
	}
	storage := miniolog.Wrap(minioClient)
	_ = storage // pakai: storage.FPutObject(ctx, bucket, object, path, opts)

	// 6. Gin router + middleware logger
	r := gin.New()
	r.Use(httpmw.RequestLogger(httpmw.Options{
		SkipPaths:       []string{"/health"},
		SensitiveFields: []string{"password", "token", "otp"},
	}))

	r.GET("/orders/:id", func(c *gin.Context) {
		ctx := c.Request.Context()

		// logger dari context udah otomatis ada request_id-nya,
		// tinggal dipanggil di layer manapun (service/repo)
		corelogger.FromContext(ctx).Info().Msg("fetching order detail")

		c.JSON(200, gin.H{"id": c.Param("id"), "status": "ok"})
	})

	r.Run(":8080")
}
