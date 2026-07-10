package npm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/registry"
	"valisgo/internal/storage"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type NPMProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
	cacheService     *proxy.CacheService
	downloadSF       singleflight.Group
}

func NewNPMProtocol(packageStore domain.PackageStore, packageFileStore domain.PackageFileStore, storage storage.Storage, cacheService *proxy.CacheService) *NPMProtocol {
	return &NPMProtocol{
		packageStore:     packageStore,
		packageFileStore: packageFileStore,
		storage:          storage,
		cacheService:     cacheService,
	}
}

func (p *NPMProtocol) MountRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/{package}", p.handleMetadata)
	r.Get("/@{scope}/{package}", p.handleMetadata)
	r.Get("/{package}/-/{filename}", p.handleDownload)
	r.Get("/@{scope}/{package}/-/{filename}", p.handleDownload)

	r.Put("/{package}", p.handleUpload)
	r.Put("/@{scope}/{package}", p.handleUpload)

	return r
}

func (p *NPMProtocol) handleMetadata(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	pkgName := getPackageName(req)
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling NPM package metadata request", "registry", reg.Name, "repository", repo.Name, "package", pkgName, "type", repo.Type)

	var content []byte
	var err error

	switch repo.Type {
	case domain.RepositoryTypeProxy:
		content, err = p.proxyMetadata(req, reg, repo, pkgName)
	case domain.RepositoryTypeLocal:
		content, err = p.localMetadata(req, reg, repo, pkgName)
	case domain.RepositoryTypeVirtual:
		content, err = p.virtualMetadata(req, reg, repo, pkgName)
	default:
		err = errors.New("unsupported repository type")
	}

	if err != nil {
		slog.Error("Failed to fetch package metadata", "error", err, "package", pkgName, "repository", repo.Name)
		if err.Error() == "not found" {
			http.Error(w, `{"error": "not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func (p *NPMProtocol) proxyMetadata(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName string) ([]byte, error) {
	upstreamURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(repo.UpstreamURL, "/"), pkgName)
	cacheKey := fmt.Sprintf("metadata/%d/npm/%s", repo.ID, pkgName)

	slog.Info("Fetching package metadata from upstream", "upstreamURL", upstreamURL)

	headers := map[string]string{
		"Accept": "application/json",
	}

	content, err := p.cacheService.FetchMetadata(req.Context(), cacheKey, upstreamURL, headers)
	if err != nil {
		slog.Error("Failed to fetch upstream metadata", "error", err, "upstreamURL", upstreamURL)
		return nil, fmt.Errorf("bad gateway")
	}

	// Rewrite dist.tarball URLs
	var meta map[string]interface{}
	if err := json.Unmarshal(content, &meta); err != nil {
		return nil, fmt.Errorf("bad upstream json")
	}

	versions, ok := meta["versions"].(map[string]interface{})
	if ok {
		for _, vInfoRaw := range versions {
			vInfo, ok := vInfoRaw.(map[string]interface{})
			if !ok {
				continue
			}
			dist, ok := vInfo["dist"].(map[string]interface{})
			if !ok {
				continue
			}
			tarballRaw, ok := dist["tarball"].(string)
			if !ok {
				continue
			}

			// Parse upstream tarball URL to get the filename
			parsedURL, err := url.Parse(tarballRaw)
			if err != nil {
				continue
			}
			filename := path.Base(parsedURL.Path)

			scheme := "http"
			if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}

			proxyTarballURL := url.URL{
				Scheme: scheme,
				Host:   req.Host,
				Path:   path.Join("/registries", reg.Name, "repositories", repo.Name, pkgName, "-", filename),
			}

			q := proxyTarballURL.Query()
			q.Set("pkg", pkgName)
			q.Set("upstream", tarballRaw)
			proxyTarballURL.RawQuery = q.Encode()

			dist["tarball"] = proxyTarballURL.String()
		}
	}

	return json.Marshal(meta)
}

func (p *NPMProtocol) localMetadata(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName string) ([]byte, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(pkgName, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("not found")
		}
		return nil, err
	}

	files, err := p.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("not found")
	}

	scheme := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	versions := make(map[string]VersionMetadata)
	latestVersion := ""
	for _, f := range files {
		vURL := url.URL{
			Scheme: scheme,
			Host:   req.Host,
			Path:   path.Join("/registries", reg.Name, "repositories", repo.Name, pkgName, "-", f.Filename),
		}

		var integrity string
		hashBytes, err := hex.DecodeString(f.Hash)
		if err == nil {
			integrity = "sha256-" + base64.StdEncoding.EncodeToString(hashBytes)
		}

		versions[f.Version] = VersionMetadata{
			Name:    pkgName,
			Version: f.Version,
			Dist: Dist{
				Tarball:   vURL.String(),
				Integrity: integrity,
			},
		}
		latestVersion = f.Version // simple approach: last processed is latest
	}

	meta := PackageMetadata{
		Name: pkgName,
		DistTags: map[string]string{
			"latest": latestVersion,
		},
		Versions: versions,
	}

	return json.Marshal(meta)
}

func (p *NPMProtocol) virtualMetadata(req *http.Request, reg *domain.Registry, repo *domain.Repository, pkgName string) ([]byte, error) {
	var mergedMeta map[string]interface{}
	found := false

	for _, member := range repo.VirtualMembers {
		var content []byte
		var err error

		if member.MemberRepo.Type == domain.RepositoryTypeProxy {
			content, err = p.proxyMetadata(req, reg, &member.MemberRepo, pkgName)
		} else {
			content, err = p.localMetadata(req, reg, &member.MemberRepo, pkgName)
		}

		if err == nil {
			var meta map[string]interface{}
			if err := json.Unmarshal(content, &meta); err == nil {
				found = true
				if mergedMeta == nil {
					mergedMeta = meta
				} else {
					// Merge versions
					if newVersions, ok := meta["versions"].(map[string]interface{}); ok {
						if existingVersions, ok := mergedMeta["versions"].(map[string]interface{}); ok {
							for k, v := range newVersions {
								if _, exists := existingVersions[k]; !exists {
									existingVersions[k] = v
								}
							}
						} else {
							mergedMeta["versions"] = newVersions
						}
					}
					// Merge dist-tags (newer member dist-tags don't override older ones for simplicity)
					if newTags, ok := meta["dist-tags"].(map[string]interface{}); ok {
						if existingTags, ok := mergedMeta["dist-tags"].(map[string]interface{}); ok {
							for k, v := range newTags {
								if _, exists := existingTags[k]; !exists {
									existingTags[k] = v
								}
							}
						} else {
							mergedMeta["dist-tags"] = newTags
						}
					}
				}
			}
		}
	}

	if !found {
		return nil, errors.New("not found")
	}

	return json.Marshal(mergedMeta)
}

func (p *NPMProtocol) handleDownload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	filename := chi.URLParam(req, "filename")

	slog.Info("Handling NPM file download", "repository", repo.Name, "filename", filename, "type", repo.Type)

	if repo.Type == domain.RepositoryTypeVirtual {
		registry.DispatchVirtualDownload(w, req, repo, p.MountRoutes())
		return
	}

	pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if repo.Type == domain.RepositoryTypeProxy {
				p.proxyDownload(w, req, repo, filename)
				return
			}
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	p.serveFileFromStorage(w, req, pkgFile)
}

func (p *NPMProtocol) proxyDownload(w http.ResponseWriter, req *http.Request, repo *domain.Repository, filename string) {
	pkgName := req.URL.Query().Get("pkg")
	upstreamURL := req.URL.Query().Get("upstream")

	if pkgName == "" || upstreamURL == "" {
		http.Error(w, "missing proxy parameters", http.StatusBadRequest)
		return
	}

	blobKey := fmt.Sprintf("%d/%s", repo.ID, filename)

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, pkgName, pkgName)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tw := &trackedWriter{ResponseWriter: w}

	_, err, _ = p.downloadSF.Do(blobKey, func() (interface{}, error) {
		_, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err == nil {
			return nil, nil
		}

		err = p.cacheService.StreamAndSave(req.Context(), tw, upstreamURL, func(r io.Reader, size int64) error {
			return p.saveProxiedFile(req.Context(), r, size, repo.ID, pkg, filename, blobKey)
		})
		return nil, err
	})

	if err != nil {
		slog.Error("Error during proxy download", "error", err)
		return
	}

	if !tw.written {
		pkgFile, err := p.packageFileStore.GetByFilenameAndRepository(filename, repo.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		p.serveFileFromStorage(w, req, pkgFile)
	}
}

func (p *NPMProtocol) saveProxiedFile(ctx context.Context, r io.Reader, size int64, repoID uint, pkg *domain.Package, filename, blobKey string) error {
	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	if err := p.storage.Put(ctx, blobKey, tee); err != nil {
		return err
	}

	hashString := fmt.Sprintf("%x", hasher.Sum(nil))

	pkgFile := &domain.PackageFile{
		PackageID: pkg.ID,
		Version:   extractVersion(filename, pkg.Name),
		Filename:  filename,
		Hash:      hashString,
		Size:      size,
		BlobKey:   blobKey,
	}

	return p.packageFileStore.Create(pkgFile)
}

func (p *NPMProtocol) serveFileFromStorage(w http.ResponseWriter, req *http.Request, pkgFile *domain.PackageFile) {
	reader, err := p.storage.Get(req.Context(), pkgFile.BlobKey)
	if err != nil {
		http.Error(w, "failed to read file from storage", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}

func (p *NPMProtocol) getOrCreatePackage(ctx context.Context, repoID uint, name, normalizedName string) (*domain.Package, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		pkg = &domain.Package{
			Name:           name,
			NormalizedName: normalizedName,
			RepositoryID:   repoID,
		}
		if err := p.packageStore.Create(pkg); err != nil {
			if existingPkg, errGet := p.packageStore.GetByNormalizedNameAndRepository(normalizedName, repoID); errGet == nil {
				return existingPkg, nil
			}
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *NPMProtocol) handleUpload(w http.ResponseWriter, req *http.Request) {
	repo := domain.RepositoryFromContext(req.Context())
	slog.Info("Handling NPM package upload", "repository", repo.Name)

	if repo.Type == domain.RepositoryTypeVirtual {
		http.Error(w, "cannot upload to virtual repository", http.StatusBadRequest)
		return
	}

	var payload PublishPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		slog.Warn("Failed to parse NPM publish payload", "error", err)
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	pkgName := payload.Name
	normalized := pkgName // For NPM we can just use the name as normalized

	pkg, err := p.getOrCreatePackage(req.Context(), repo.ID, pkgName, normalized)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for filename, attachment := range payload.Attachments {
		data, err := base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			slog.Warn("Failed to decode attachment data", "filename", filename)
			continue
		}

		// Save to storage
		blobKey := fmt.Sprintf("%d/%s", repo.ID, filename)
		reader := bytes.NewReader(data)
		hasher := sha256.New()
		tee := io.TeeReader(reader, hasher)

		if err := p.storage.Put(req.Context(), blobKey, tee); err != nil {
			slog.Error("Failed to store file", "filename", filename, "error", err)
			continue
		}

		hashString := fmt.Sprintf("%x", hasher.Sum(nil))

		// Find version for this attachment by checking payload.Versions
		// This is simplified. Usually filename contains the version.
		version := "unknown"
		for _, vData := range payload.Versions {
			var vMeta VersionMetadata
			if err := json.Unmarshal(vData, &vMeta); err == nil {
				// Assuming filename is something like <pkg>-<version>.tgz
				if strings.Contains(filename, vMeta.Version) {
					version = vMeta.Version
					break
				}
			}
		}

		pkgFile := &domain.PackageFile{
			PackageID: pkg.ID,
			Version:   version,
			Filename:  filename,
			Hash:      hashString,
			Size:      int64(len(data)),
			BlobKey:   blobKey,
		}

		if err := p.packageFileStore.Create(pkgFile); err != nil {
			slog.Error("Failed to save file metadata", "error", err)
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"success": true}`))
}

func getPackageName(req *http.Request) string {
	scope := chi.URLParam(req, "scope")
	pkg := chi.URLParam(req, "package")
	if scope != "" {
		return fmt.Sprintf("%s/%s", scope, pkg)
	}
	return pkg
}

type trackedWriter struct {
	http.ResponseWriter
	written bool
}

func (tw *trackedWriter) Write(b []byte) (int, error) {
	tw.written = true
	return tw.ResponseWriter.Write(b)
}

func (tw *trackedWriter) WriteHeader(statusCode int) {
	tw.written = true
	tw.ResponseWriter.WriteHeader(statusCode)
}

func extractVersion(filename, pkgName string) string {
	base := filename
	if strings.HasSuffix(base, ".tgz") {
		base = strings.TrimSuffix(base, ".tgz")
	} else {
		return "unknown"
	}

	parts := strings.Split(pkgName, "/")
	shortName := parts[len(parts)-1]

	prefix := shortName + "-"
	if strings.HasPrefix(base, prefix) {
		return strings.TrimPrefix(base, prefix)
	}

	return "unknown"
}
