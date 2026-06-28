package main

import (
	"flag"
	"log"
	"net/http"

	"valisgo/internal/database"
	"valisgo/internal/env"
	"valisgo/internal/registry"
	"valisgo/internal/registry/pypi"
	"valisgo/internal/server"
	"valisgo/internal/server/management"

	"gorm.io/gorm"
)

func setupDatabase() *gorm.DB {
	dbDriver := flag.String("db-driver", env.GetOrDefault("DB_DRIVER", "postgres"), "Database driver (e.g., postgres)")
	dbDsn := flag.String("db-dsn", env.GetOrDefault("DB_DSN", "postgres://user:pass@localhost:5432/valisgo?sslmode=disable"), "Database connection string")
	flag.Parse()

	db, err := database.NewConnection(*dbDriver, *dbDsn)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	db := setupDatabase()

	mgmtAPI := management.NewAPI(db)

	srv := server.NewServer()

	srv.RegisterProtocol("pypi", &pypi.PyPIProtocol{})
	srv.RegisterRepository(registry.Repository{Name: "my-pypi", Format: "pypi"})

	r := srv.SetupRouter()

	r.Mount("/manage", mgmtAPI.MountRoutes())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	log.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
