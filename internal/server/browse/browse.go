package browse

import (
	"embed"
	"html/template"
	"io"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	registriesTemplate   = template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/registries.html"))
	repositoriesTemplate = template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/repositories.html"))
	packagesTemplate     = template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/packages.html"))
	filesTemplate        = template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/files.html"))
)

type API struct {
	db      *gorm.DB
	storage storage.Storage
}

func NewAPI(db *gorm.DB, storage storage.Storage) *API {
	return &API{db: db, storage: storage}
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", a.HandleRegistries)
	r.Get("/{registry}", a.HandleRepositories)
	r.Get("/{registry}/{repository}", a.HandlePackages)
	r.Get("/{registry}/{repository}/{package}", a.HandleFiles)
	r.Get("/{registry}/{repository}/{package}/{filename}", a.HandleDownload)
	return r
}

func (a *API) HandleRegistries(w http.ResponseWriter, r *http.Request) {
	var registries []domain.Registry
	if err := a.db.Find(&registries).Error; err != nil {
		http.Error(w, "Failed to load registries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := registriesTemplate.ExecuteTemplate(w, "base", registries); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (a *API) HandleRepositories(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "registry")

	var registry domain.Registry
	if err := a.db.Where("name = ?", registryName).First(&registry).Error; err != nil {
		http.Error(w, "Registry not found", http.StatusNotFound)
		return
	}

	var repos []domain.Repository
	if err := a.db.Where("registry_id = ?", registry.ID).Find(&repos).Error; err != nil {
		http.Error(w, "Failed to load repositories", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName string
		Repositories []domain.Repository
	}{
		RegistryName: registry.Name,
		Repositories: repos,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := repositoriesTemplate.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (a *API) HandlePackages(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "registry")
	repositoryName := chi.URLParam(r, "repository")

	var registry domain.Registry
	if err := a.db.Where("name = ?", registryName).First(&registry).Error; err != nil {
		http.Error(w, "Registry not found", http.StatusNotFound)
		return
	}

	var repo domain.Repository
	if err := a.db.Where("name = ? AND registry_id = ?", repositoryName, registry.ID).First(&repo).Error; err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var pkgs []domain.Package
	if err := a.db.Where("repository_id = ?", repo.ID).Find(&pkgs).Error; err != nil {
		http.Error(w, "Failed to load packages", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName   string
		RepositoryName string
		Packages       []domain.Package
	}{
		RegistryName:   registry.Name,
		RepositoryName: repo.Name,
		Packages:       pkgs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := packagesTemplate.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (a *API) HandleFiles(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "registry")
	repositoryName := chi.URLParam(r, "repository")
	packageName := chi.URLParam(r, "package")

	var registry domain.Registry
	if err := a.db.Where("name = ?", registryName).First(&registry).Error; err != nil {
		http.Error(w, "Registry not found", http.StatusNotFound)
		return
	}

	var repo domain.Repository
	if err := a.db.Where("name = ? AND registry_id = ?", repositoryName, registry.ID).First(&repo).Error; err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var pkg domain.Package
	if err := a.db.Where("normalized_name = ? AND repository_id = ?", packageName, repo.ID).First(&pkg).Error; err != nil {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	var files []domain.PackageFile
	if err := a.db.Where("package_id = ?", pkg.ID).Find(&files).Error; err != nil {
		http.Error(w, "Failed to load files", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName   string
		RepositoryName string
		PackageName    string
		Files          []domain.PackageFile
	}{
		RegistryName:   registry.Name,
		RepositoryName: repo.Name,
		PackageName:    pkg.Name,
		Files:          files,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := filesTemplate.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (a *API) HandleDownload(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "registry")
	repositoryName := chi.URLParam(r, "repository")
	packageName := chi.URLParam(r, "package")
	filename := chi.URLParam(r, "filename")

	var registry domain.Registry
	if err := a.db.Where("name = ?", registryName).First(&registry).Error; err != nil {
		http.Error(w, "Registry not found", http.StatusNotFound)
		return
	}

	var repo domain.Repository
	if err := a.db.Where("name = ? AND registry_id = ?", repositoryName, registry.ID).First(&repo).Error; err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var pkg domain.Package
	if err := a.db.Where("normalized_name = ? AND repository_id = ?", packageName, repo.ID).First(&pkg).Error; err != nil {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	var pkgFile domain.PackageFile
	if err := a.db.Where("filename = ? AND package_id = ?", filename, pkg.ID).First(&pkgFile).Error; err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	reader, err := a.storage.Get(r.Context(), pkgFile.BlobKey)
	if err != nil {
		http.Error(w, "File not found in storage", http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+pkgFile.Filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}
