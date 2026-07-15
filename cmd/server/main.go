package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"valisgo/internal/auth"
	"valisgo/internal/database"
	"valisgo/internal/env"
	"valisgo/internal/server"
	"valisgo/internal/server/authapi"
	"valisgo/internal/server/browse"
	"valisgo/internal/server/management"
	"valisgo/internal/server/registries"
	"valisgo/internal/storage"
	"valisgo/internal/store"
	"valisgo/internal/strutil"

	"github.com/go-chi/chi/v5"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"gorm.io/gorm"
)

var (
	dbDriver         = flag.String("db-driver", env.GetOrDefault("DB_DRIVER", "postgres"), "Database driver (e.g., postgres)")
	dbDsn            = flag.String("db-dsn", env.GetOrDefault("DB_DSN", "postgres://user:pass@localhost:5432/valisgo?sslmode=disable"), "Database connection string")
	storageURL       = flag.String("storage-url", env.GetOrDefault("STORAGE_URL", "file://./data/blobs"), "Storage bucket URL")
	logLevel         = flag.String("log-level", env.GetOrDefault("LOG_LEVEL", "debug"), "Logging level (debug, info, warn, error)")
	oidcIssuerURL    = flag.String("oidc-issuer", env.GetOrDefault("OIDC_ISSUER_URL", "http://dex:5556/dex"), "OIDC Issuer URL for validation")
	oidcClientID     = flag.String("oidc-client-id", env.GetOrDefault("OIDC_CLIENT_ID", "valisgo-web"), "OIDC Client ID for web app")
	oidcClientSecret = flag.String("oidc-client-secret", env.GetOrDefault("OIDC_CLIENT_SECRET", "web-secret-key"), "OIDC Client Secret for web app")
	oidcRedirectURL  = flag.String("oidc-redirect-url", env.GetOrDefault("OIDC_REDIRECT_URL", "http://localhost:8080/callback"), "OIDC Redirect URL for web app callback")
	oidcScopes       = flag.String("oidc-scopes", env.GetOrDefault("OIDC_SCOPES", "openid,profile,email"), "Comma-separated OIDC Scopes for web app")
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

	parsedScopes := strutil.ParseCommaSeparated(*oidcScopes)

	webOIDCConfig := auth.OIDCConfig{
		IssuerURL:    *oidcIssuerURL,
		ClientID:     *oidcClientID,
		ClientSecret: *oidcClientSecret,
		Scopes:       parsedScopes,
		RedirectURL:  *oidcRedirectURL,
	}
	webAuthenticator, err := auth.NewAuthenticator(context.Background(), webOIDCConfig)
	if err != nil {
		log.Fatalf("failed to initialize OIDC authenticator for %s: %v", *oidcIssuerURL, err)
	}

	srv := server.NewServer(enforcer, webAuthenticator.Verifier())

	r := srv.SetupRouter()

	registryStore := store.NewRegistryStore(db)
	repositoryStore := store.NewRepositoryStore(db)
	packageStore := store.NewPackageStore(db)
	packageFileStore := store.NewPackageFileStore(db)

	r.Group(func(r chi.Router) {
		srv.Protect(r)

		mgmtAPI := management.NewAPI(db)
		r.Mount("/manage", mgmtAPI.MountRoutes())

		registriesAPI := registries.NewAPI(db, blobStorage)
		r.Mount("/registries", registriesAPI.MountRoutes())
	})

	r.Group(func(r chi.Router) {
		srv.ProtectWithRedirect(r, "/login")

		browseAPI := browse.NewAPI(registryStore, repositoryStore, packageStore, packageFileStore, blobStorage)
		r.Mount("/browse", browseAPI.MountRoutes())
	})

	r.Group(func(r chi.Router) {
		authAPI := authapi.NewAPI(webAuthenticator)
		r.Mount("/", authAPI.MountRoutes())

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		})
	})

	log.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
