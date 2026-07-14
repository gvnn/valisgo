package npm

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
)

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
	}

	if err != nil && err.Error() == "not found" {
		http.Error(w, `{"error": "not found"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
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

			scheme := getScheme(req)

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
		return nil, err
	}
	if pkg == nil {
		return nil, errors.New("not found")
	}

	files, err := p.packageFileStore.ListByPackage(pkg.ID)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("not found")
	}

	scheme := getScheme(req)

	versions := make(map[string]VersionMetadata)
	latestVersion := ""
	for _, f := range files {
		vURL := url.URL{
			Scheme: scheme,
			Host:   req.Host,
			Path:   path.Join("/registries", reg.Name, "repositories", repo.Name, pkgName, "-", f.Filename),
		}

		integrity := formatIntegrity(f.Hash)

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
				mergedMeta = mergeMetadata(mergedMeta, meta)
			}
		}
	}

	if !found {
		return nil, errors.New("not found")
	}

	return json.Marshal(mergedMeta)
}
