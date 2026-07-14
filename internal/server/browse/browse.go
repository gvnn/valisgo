package browse

import (
	"embed"
	"html/template"
	"io"
	"net/http"

	"valisgo/internal/domain"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
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
	registryStore    domain.RegistryStore
	repositoryStore  domain.RepositoryStore
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
}

func NewAPI(registryStore domain.RegistryStore, repositoryStore domain.RepositoryStore, packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage) *API {
	return &API{
		registryStore:    registryStore,
		repositoryStore:  repositoryStore,
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
	}
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", a.HandleRegistries)
	r.Get("/{registry}", a.HandleRepositories)
	r.Get("/{registry}/{repository}", a.HandlePackages)
	r.Get("/{registry}/{repository}/*", a.HandlePackageOrFile)
	return r
}

func (a *API) HandleRegistries(w http.ResponseWriter, r *http.Request) {
	registries, err := a.registryStore.All()
	if err != nil {
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

	registry, ok := a.getRegistry(w, registryName)
	if !ok {
		return
	}

	repos, err := a.repositoryStore.ListByRegistryID(registry.ID)
	if err != nil {
		http.Error(w, "Failed to load repositories", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName string
		Repositories []*domain.Repository
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

	registry, ok := a.getRegistry(w, registryName)
	if !ok {
		return
	}

	repo, ok := a.getRepository(w, repositoryName, registry.ID)
	if !ok {
		return
	}

	pkgs, err := a.packageStore.ListByRepository(repo.ID)
	if err != nil {
		http.Error(w, "Failed to load packages", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName   string
		RepositoryName string
		Packages       []*domain.Package
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

func (a *API) HandlePackageOrFile(w http.ResponseWriter, r *http.Request) {
	registryName := chi.URLParam(r, "registry")
	repositoryName := chi.URLParam(r, "repository")
	pathParam := chi.URLParam(r, "*")
	
	if pathParam == "" {
		a.HandlePackages(w, r)
		return
	}

	registry, ok := a.getRegistry(w, registryName)
	if !ok {
		return
	}

	repo, ok := a.getRepository(w, repositoryName, registry.ID)
	if !ok {
		return
	}

	var pkgName, fileName string

	switch registry.Format {
	case domain.FormatGo:
		pkgName, fileName = parseGoPath(pathParam)
	case domain.FormatNPM:
		pkgName, fileName = parseNPMPath(pathParam)
	default:
		pkgName, fileName = parseDefaultPath(pathParam)
	}

	pkg, err := a.packageStore.GetByNormalizedNameAndRepository(pkgName, repo.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if pkg == nil {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	if fileName == "" {
		a.renderFiles(w, r, registry, repo, pkg)
	} else {
		a.renderDownload(w, r, pkg, fileName)
	}
}

func (a *API) renderFiles(w http.ResponseWriter, r *http.Request, registry *domain.Registry, repo *domain.Repository, pkg *domain.Package) {
	files, err := a.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		http.Error(w, "Failed to load files", http.StatusInternalServerError)
		return
	}

	data := struct {
		RegistryName   string
		RepositoryName string
		PackageName    string
		Files          []*domain.PackageFile
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

func (a *API) renderDownload(w http.ResponseWriter, r *http.Request, pkg *domain.Package, filename string) {
	pkgFile, err := a.packageFileStore.GetByFilenameAndPackage(filename, pkg.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if pkgFile == nil {
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

func (a *API) getRegistry(w http.ResponseWriter, name string) (*domain.Registry, bool) {
	registry, err := a.registryStore.GetByName(name)
	if err != nil {
		http.Error(w, "Failed to fetch registry", http.StatusInternalServerError)
		return nil, false
	}
	if registry == nil {
		http.Error(w, "Registry not found", http.StatusNotFound)
		return nil, false
	}
	return registry, true
}

func (a *API) getRepository(w http.ResponseWriter, name string, registryID uint) (*domain.Repository, bool) {
	repo, err := a.repositoryStore.GetByNameAndRegistryID(name, registryID)
	if err != nil {
		http.Error(w, "Failed to fetch repository", http.StatusInternalServerError)
		return nil, false
	}
	if repo == nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return nil, false
	}
	return repo, true
}
