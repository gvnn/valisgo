package main

import (
	"flag"
	"log"
	"net/http"

	"valisgo/internal/database"
	"valisgo/internal/env"
	"valisgo/internal/server"
	"valisgo/internal/server/management"
	"valisgo/internal/server/registries"

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

	enforcer := server.SetupCasbin(db)

	srv := server.NewServer(enforcer)

	r := srv.SetupRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mgmtAPI := management.NewAPI(db)
	r.Mount("/manage", mgmtAPI.MountRoutes())

	registriesAPI := registries.NewAPI(db)
	r.Mount("/registries", registriesAPI.MountRoutes())

	log.Println("Server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
