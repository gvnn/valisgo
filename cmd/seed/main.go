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

	// Create PyPI proxy repository
	var proxyRepo domain.Repository
	if err := db.FirstOrCreate(&proxyRepo, domain.Repository{
		Name:        "pypi-proxy",
		RegistryID:  pypiReg.ID,
		Type:        domain.RepositoryTypeProxy,
		UpstreamURL: "https://pypi.org",
	}).Error; err != nil {
		log.Fatalf("failed to seed pypi proxy repository: %v", err)
	}
	log.Printf("Repository 'pypi-proxy' for PyPI (ID: %d) seeded.", proxyRepo.ID)

	// Create PyPI virtual repository
	var virtualRepo domain.Repository
	if err := db.FirstOrCreate(&virtualRepo, domain.Repository{
		Name:       "pypi-virtual",
		RegistryID: pypiReg.ID,
		Type:       domain.RepositoryTypeVirtual,
	}).Error; err != nil {
		log.Fatalf("failed to seed pypi virtual repository: %v", err)
	}
	log.Printf("Repository 'pypi-virtual' for PyPI (ID: %d) seeded.", virtualRepo.ID)

	// Add members to virtual repository
	members := []domain.VirtualRepoMember{
		{VirtualRepoID: virtualRepo.ID, MemberRepoID: defaultRepo.ID, Priority: 1}, // Local repo first
		{VirtualRepoID: virtualRepo.ID, MemberRepoID: proxyRepo.ID, Priority: 2},   // Proxy repo second
	}

	for _, m := range members {
		if err := db.FirstOrCreate(&domain.VirtualRepoMember{}, m).Error; err != nil {
			log.Fatalf("failed to seed virtual repo member: %v", err)
		}
	}
	log.Printf("Seeded members for virtual repository 'pypi-virtual'.")

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

	// Create NPM registry if it doesn't exist
	var npmReg domain.Registry
	if err := db.FirstOrCreate(&npmReg, domain.Registry{Name: "mynpm", Format: domain.FormatNPM}).Error; err != nil {
		log.Fatalf("failed to seed npm registry: %v", err)
	}
	log.Printf("Registry NPM (ID: %d) seeded.", npmReg.ID)

	// Create NPM proxy repository
	var npmProxyRepo domain.Repository
	if err := db.FirstOrCreate(&npmProxyRepo, domain.Repository{
		Name:        "npm-proxy",
		RegistryID:  npmReg.ID,
		Type:        domain.RepositoryTypeProxy,
		UpstreamURL: "https://registry.npmjs.org",
	}).Error; err != nil {
		log.Fatalf("failed to seed npm proxy repository: %v", err)
	}
	log.Printf("Repository 'npm-proxy' for NPM (ID: %d) seeded.", npmProxyRepo.ID)

	// Create NPM local repository
	var npmLocalRepo domain.Repository
	if err := db.FirstOrCreate(&npmLocalRepo, domain.Repository{
		Name:       "npm-local",
		RegistryID: npmReg.ID,
		Type:       domain.RepositoryTypeLocal,
	}).Error; err != nil {
		log.Fatalf("failed to seed npm local repository: %v", err)
	}
	log.Printf("Repository 'npm-local' for NPM (ID: %d) seeded.", npmLocalRepo.ID)

	// Create NPM virtual repository
	var npmVirtualRepo domain.Repository
	if err := db.FirstOrCreate(&npmVirtualRepo, domain.Repository{
		Name:       "npm-virtual",
		RegistryID: npmReg.ID,
		Type:       domain.RepositoryTypeVirtual,
	}).Error; err != nil {
		log.Fatalf("failed to seed npm virtual repository: %v", err)
	}
	log.Printf("Repository 'npm-virtual' for NPM (ID: %d) seeded.", npmVirtualRepo.ID)

	// Add members to NPM virtual repository
	npmMembers := []domain.VirtualRepoMember{
		{VirtualRepoID: npmVirtualRepo.ID, MemberRepoID: npmLocalRepo.ID, Priority: 1},
		{VirtualRepoID: npmVirtualRepo.ID, MemberRepoID: npmProxyRepo.ID, Priority: 2},
	}
	for _, m := range npmMembers {
		if err := db.FirstOrCreate(&domain.VirtualRepoMember{}, m).Error; err != nil {
			log.Fatalf("failed to seed virtual repo member: %v", err)
		}
	}
	log.Printf("Seeded members for virtual repository 'npm-virtual'.")

	log.Println("Seeding completed successfully.")
}
