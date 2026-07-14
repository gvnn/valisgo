package pypi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"valisgo/internal/domain"
	"valisgo/internal/registry"

	"github.com/go-chi/chi/v5"
)

func (p *PyPIProtocol) handleSimpleIndex(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling PyPI simple index request", "registry", reg.Name, "repository", repo.Name, "type", repo.Type)

	var allPkgs []*domain.Package
	var err error

	if repo.Type == domain.RepositoryTypeVirtual {
		allPkgs, err = p.buildVirtualIndex(repo)
	} else {
		allPkgs, err = p.packageStore.ListByRepository(repo.ID)
	}

	if err != nil {
		slog.Error("Failed to list packages", "error", err, "repository", repo.Name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if acceptsJSON(req) {
		p.serveSimpleIndexJSON(w, allPkgs)
		return
	}

	p.serveSimpleIndexHTML(w, reg, repo, allPkgs)
}

func (p *PyPIProtocol) buildVirtualIndex(repo *domain.Repository) ([]*domain.Package, error) {
	return p.packageStore.ListDistinctByVirtualRepository(repo.ID)
}

func (p *PyPIProtocol) serveSimpleIndexHTML(w http.ResponseWriter, reg *domain.Registry, repo *domain.Repository, pkgs []*domain.Package) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		RepoName     string
		RegistryName string
		Packages     []*domain.Package
	}{
		RepoName:     repo.Name,
		RegistryName: reg.Name,
		Packages:     pkgs,
	}

	if err := indexTemplate.ExecuteTemplate(w, "index.html", data); err != nil {
		slog.Error("Failed to execute index template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) fetchLocalPackageFiles(req *http.Request, reg *domain.Registry, repo *domain.Repository, normalized string) ([]templateFile, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalized, repo.ID)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, errors.New("package not found")
	}
	files, err := p.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		return nil, err
	}
	var tFiles []templateFile
	for _, f := range files {
		u := url.URL{
			Path: path.Join("/registries", reg.Name, "repositories", repo.Name, "packages", f.Filename),
		}
		tFiles = append(tFiles, templateFile{
			Filename:   f.Filename,
			Hash:       f.Hash,
			URL:        u.String(),
			Size:       f.Size,
			UploadTime: f.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			Version:    f.Version,
		})
	}
	return tFiles, nil
}

func (p *PyPIProtocol) handleSimplePackage(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	pkgName := chi.URLParam(req, "package")
	normalized := NormalizeName(pkgName)
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling PyPI simple package request", "registry", reg.Name, "repository", repo.Name, "package", pkgName, "type", repo.Type)

	var allFiles []templateFile
	var err error

	switch repo.Type {
	case domain.RepositoryTypeVirtual:
		allFiles, err = p.buildVirtualPackageFiles(req, reg, repo, pkgName, normalized)
	case domain.RepositoryTypeProxy:
		allFiles, err = p.proxySimplePackage(req, reg, repo, pkgName, normalized)
	default:
		allFiles, err = p.fetchLocalPackageFiles(req, reg, repo, normalized)
	}

	if err != nil {
		slog.Error("Failed to fetch package files", "error", err, "package", pkgName, "repository", repo.Name)
	}

	if err != nil && err.Error() == "package not found" {
		http.Error(w, "package not found", http.StatusNotFound)
		return
	}

	if err != nil && repo.Type == domain.RepositoryTypeProxy && err.Error() == "bad gateway" {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	if registry.HandleInternalError(w, err) {
		return
	}

	if acceptsJSON(req) {
		p.serveSimplePackageJSON(w, pkgName, allFiles)
		return
	}

	p.serveSimplePackageHTML(w, reg, repo, pkgName, allFiles)
}

func (p *PyPIProtocol) buildVirtualPackageFiles(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName, normalized string) ([]templateFile, error) {
	var allFiles []templateFile

	for _, member := range repo.VirtualMembers {
		var tFiles []templateFile
		var err error

		if member.MemberRepo.Type == domain.RepositoryTypeProxy {
			tFiles, err = p.proxySimplePackage(req, reg, &member.MemberRepo, pkgName, normalized)
		} else {
			tFiles, err = p.fetchLocalPackageFiles(req, reg, &member.MemberRepo, normalized)
		}

		if err == nil {
			allFiles = append(allFiles, tFiles...)
		} else {
			slog.Warn("Failed to fetch files from virtual member", "member", member.MemberRepo.Name, "package", pkgName, "error", err)
		}
	}

	allFiles = deduplicateFiles(allFiles)

	if len(allFiles) == 0 {
		return nil, errors.New("package not found")
	}
	return allFiles, nil
}

func (p *PyPIProtocol) serveSimpleIndexJSON(w http.ResponseWriter, pkgs []*domain.Package) {
	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	resp := SimpleIndexResponse{
		Meta: SimpleMeta{APIVersion: "1.1"},
	}
	for _, pkg := range pkgs {
		resp.Projects = append(resp.Projects, SimpleProject{Name: pkg.NormalizedName})
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode simple index JSON", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimplePackageJSON(w http.ResponseWriter, pkgName string, tFiles []templateFile) {
	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	resp := SimplePackageResponse{
		Meta:  SimpleMeta{APIVersion: "1.1"},
		Name:  pkgName,
		Files: []SimpleFile{},
	}

	versionsSet := make(map[string]struct{})
	for _, tf := range tFiles {
		if tf.Version != "" && tf.Version != "unknown" {
			if _, exists := versionsSet[tf.Version]; !exists {
				versionsSet[tf.Version] = struct{}{}
				resp.Versions = append(resp.Versions, tf.Version)
			}
		}

		resp.Files = append(resp.Files, SimpleFile{
			Filename:   tf.Filename,
			URL:        tf.URL,
			Hashes:     SimpleFileHashes{SHA256: tf.Hash},
			Size:       tf.Size,
			UploadTime: tf.UploadTime,
		})
	}

	if resp.Versions == nil {
		resp.Versions = []string{}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode simple package JSON", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) serveSimplePackageHTML(w http.ResponseWriter, reg *domain.Registry, repo *domain.Repository, pkgName string, files []templateFile) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		RepoName     string
		RegistryName string
		PackageName  string
		Files        []templateFile
	}{
		RepoName:     repo.Name,
		RegistryName: reg.Name,
		PackageName:  pkgName,
		Files:        files,
	}

	if err := packageTemplate.ExecuteTemplate(w, "package.html", data); err != nil {
		slog.Error("Failed to execute package template", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (p *PyPIProtocol) proxySimplePackage(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName, normalized string) ([]templateFile, error) {
	upstreamURL := fmt.Sprintf("%s/simple/%s/", strings.TrimSuffix(repo.UpstreamURL, "/"), pkgName)
	cacheKey := fmt.Sprintf("metadata/%d/simple/%s?json=true", repo.ID, pkgName)

	slog.Info("Fetching simple package from upstream", "upstreamURL", upstreamURL)

	headers := map[string]string{
		"Accept": "application/vnd.pypi.simple.v1+json",
	}

	content, err := p.cacheService.FetchMetadata(req.Context(), cacheKey, upstreamURL, headers)
	if err != nil {
		slog.Error("Failed to fetch upstream metadata", "error", err, "upstreamURL", upstreamURL)
		return nil, fmt.Errorf("bad gateway")
	}

	var upstreamResp SimplePackageResponse
	if err := json.Unmarshal(content, &upstreamResp); err != nil {
		slog.Error("Failed to unmarshal upstream JSON", "error", err, "upstreamURL", upstreamURL)
		return nil, fmt.Errorf("bad upstream response")
	}

	var tFiles []templateFile
	for _, f := range upstreamResp.Files {
		u := url.URL{
			Path: path.Join("/registries", reg.Name, "repositories", repo.Name, "packages", f.Filename),
		}
		if f.URL != "" {
			q := u.Query()
			q.Set("pkg", pkgName)
			q.Set("upstream", f.URL)
			u.RawQuery = q.Encode()
		}

		tFiles = append(tFiles, templateFile{
			Filename:   f.Filename,
			Hash:       f.Hashes.SHA256,
			URL:        u.String(),
			Size:       f.Size,
			UploadTime: f.UploadTime,
			Version:    extractVersion(f.Filename, pkgName),
		})
	}

	return tFiles, nil
}
