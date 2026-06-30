package main

import (
	"flag"
	"log"

	"valisgo/internal/database"
	"valisgo/internal/domain"
	"valisgo/internal/env"

	"gorm.io/gorm"
)

var (
	dbDriver = flag.String("db-driver", env.GetOrDefault("DB_DRIVER", "postgres"), "Database driver (e.g., postgres)")
	dbDsn    = flag.String("db-dsn", env.GetOrDefault("DB_DSN", "postgres://user:pass@localhost:5432/valisgo?sslmode=disable"), "Database connection string")
)

func setupDatabase() *gorm.DB {
	db, err := database.NewConnection(*dbDriver, *dbDsn)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	flag.Parse()

	db := setupDatabase()

	log.Println("Seeding database...")

	// Create PyPI registry if it doesn't exist
	var pypiReg domain.Registry
	if err := db.FirstOrCreate(&pypiReg, domain.Registry{Name: "mypypi", Format: domain.FormatPyPI}).Error; err != nil {
		log.Fatalf("failed to seed pypi registry: %v", err)
	}
	log.Printf("Registry PyPI (ID: %d) seeded.", pypiReg.ID)

	// Create a default repository for the pypi registry if it doesn't exist
	var defaultRepo domain.Repository
	if err := db.FirstOrCreate(&defaultRepo, domain.Repository{Name: "myrepo", RegistryID: pypiReg.ID}).Error; err != nil {
		log.Fatalf("failed to seed default repository: %v", err)
	}
	log.Printf("Repository 'default' for PyPI (ID: %d) seeded.", defaultRepo.ID)

	// Create File registry if it doesn't exist
	var fileReg domain.Registry
	if err := db.FirstOrCreate(&fileReg, domain.Registry{Name: "myfile", Format: domain.FormatFile}).Error; err != nil {
		log.Fatalf("failed to seed file registry: %v", err)
	}
	log.Printf("Registry File (ID: %d) seeded.", fileReg.ID)

	var fileRepo domain.Repository
	if err := db.FirstOrCreate(&fileRepo, domain.Repository{Name: "myrepo", RegistryID: fileReg.ID}).Error; err != nil {
		log.Fatalf("failed to seed file repository: %v", err)
	}
	log.Printf("Repository 'myrepo' for File (ID: %d) seeded.", fileRepo.ID)

	log.Println("Seeding completed successfully.")
}
