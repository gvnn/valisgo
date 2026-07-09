package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"valisgo/internal/database"
	"valisgo/internal/env"
	"valisgo/internal/server"
	"valisgo/internal/server/browse"
	"valisgo/internal/server/management"
	"valisgo/internal/server/registries"
	"valisgo/internal/storage"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"gorm.io/gorm"
)

var (
	dbDriver   = flag.String("db-driver", env.GetOrDefault("DB_DRIVER", "postgres"), "Database driver (e.g., postgres)")
	dbDsn      = flag.String("db-dsn", env.GetOrDefault("DB_DSN", "postgres://user:pass@localhost:5432/valisgo?sslmode=disable"), "Database connection string")
	storageURL = flag.String("storage-url", env.GetOrDefault("STORAGE_URL", "file://./data/blobs"), "Storage bucket URL")
	logLevel   = flag.String("log-level", env.GetOrDefault("LOG_LEVEL", "debug"), "Logging level (debug, info, warn, error)")
)

func setupDatabase() *gorm.DB {
	db, err := database.NewConnection(*dbDriver, *dbDsn)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func setupStorage() (storage.Storage, func()) {
	if strings.HasPrefix(*storageURL, "file://") {
		dir := strings.TrimPrefix(*storageURL, "file://")
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("failed to create blob dir: %v", err)
		}
	}

	bucket, err := blob.OpenBucket(context.Background(), *storageURL)
	if err != nil {
		log.Fatalf("failed to open bucket: %v", err)
	}

	return storage.NewBlobStorage(bucket), func() {
		bucket.Close()
	}
}

func getLogLevelFromEnv() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")

	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setupLogger() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevelFromEnv(),
	}))

	slog.SetDefault(logger)
}

func main() {
	flag.Parse()

	setupLogger()

	db := setupDatabase()
	blobStorage, cleanup := setupStorage()
	defer cleanup()

	enforcer := server.SetupCasbin(db)

	srv := server.NewServer(enforcer)

	r := srv.SetupRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mgmtAPI := management.NewAPI(db)
	r.Mount("/manage", mgmtAPI.MountRoutes())

	registriesAPI := registries.NewAPI(db, blobStorage)
	r.Mount("/registries", registriesAPI.MountRoutes())

	browseAPI := browse.NewAPI(db, blobStorage)
	r.Mount("/browse", browseAPI.MountRoutes())

	log.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
