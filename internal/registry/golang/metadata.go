package golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"valisgo/internal/domain"
)

func (p *GoProtocol) handleListVersions(w http.ResponseWriter, req *http.Request, modulePath string) {
	repo := domain.RepositoryFromContext(req.Context())
	reg := domain.RegistryFromContext(req.Context())

	slog.Info("Handling Go module list request", "registry", reg.Name, "repository", repo.Name, "module", modulePath, "type", repo.Type)

	var content []byte
	var err error

	switch repo.Type {
	case domain.RepositoryTypeProxy:
		content, err = p.proxyListVersions(req, repo, modulePath)
	case domain.RepositoryTypeLocal:
		content, err = p.localListVersions(req, repo, modulePath)
	case domain.RepositoryTypeVirtual:
		content, err = p.virtualListVersions(req, reg, repo, modulePath)
	default:
		err = errors.New("unsupported repository type")
	}

	if err != nil && err.Error() == "not found" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func (p *GoProtocol) proxyListVersions(req *http.Request, repo *domain.Repository, modulePath string) ([]byte, error) {
	upstreamURL := fmt.Sprintf("%s/%s/@v/list", strings.TrimSuffix(repo.UpstreamURL, "/"), modulePath)
	cacheKey := fmt.Sprintf("metadata/%d/golang/%s/list", repo.ID, modulePath)

	content, err := p.cacheService.FetchMetadata(req.Context(), cacheKey, upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("bad gateway")
	}

	return content, nil
}

func (p *GoProtocol) localListVersions(req *http.Request, repo *domain.Repository, modulePath string) ([]byte, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(modulePath, repo.ID)
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

	versionsMap := make(map[string]bool)
	for _, f := range files {
		versionsMap[f.Version] = true
	}

	var versions []string
	for v := range versionsMap {
		versions = append(versions, v)
	}

	return []byte(strings.Join(versions, "\n")), nil
}

func parseVersionsToMap(content []byte, versionsMap map[string]bool) {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		versionsMap[line] = true
	}
}

func (p *GoProtocol) virtualListVersions(req *http.Request, reg *domain.Registry, repo *domain.Repository, modulePath string) ([]byte, error) {
	versionsMap := make(map[string]bool)

	for _, member := range repo.VirtualMembers {
		var content []byte
		var err error

		if member.MemberRepo.Type == domain.RepositoryTypeProxy {
			content, err = p.proxyListVersions(req, &member.MemberRepo, modulePath)
		} else {
			content, err = p.localListVersions(req, &member.MemberRepo, modulePath)
		}

		if err != nil {
			continue
		}

		parseVersionsToMap(content, versionsMap)
	}

	var versions []string
	for v := range versionsMap {
		versions = append(versions, v)
	}

	if len(versions) == 0 {
		return nil, errors.New("not found")
	}

	return []byte(strings.Join(versions, "\n")), nil
}

func (p *GoProtocol) handleVersionInfo(w http.ResponseWriter, req *http.Request, modulePath, version string) {
	repo := domain.RepositoryFromContext(req.Context())

	var content []byte
	var err error

	switch repo.Type {
	case domain.RepositoryTypeProxy:
		content, err = p.proxyVersionInfo(req, repo, modulePath, version)
	case domain.RepositoryTypeLocal:
		content, err = p.localVersionInfo(req, repo, modulePath, version)
	case domain.RepositoryTypeVirtual:
		content, err = p.virtualVersionInfo(req, repo, modulePath, version)
	default:
		err = errors.New("unsupported repository type")
	}

	if err != nil && err.Error() == "not found" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func (p *GoProtocol) proxyVersionInfo(req *http.Request, repo *domain.Repository, modulePath, version string) ([]byte, error) {
	upstreamURL := fmt.Sprintf("%s/%s/@v/%s.info", strings.TrimSuffix(repo.UpstreamURL, "/"), modulePath, version)
	cacheKey := fmt.Sprintf("metadata/%d/golang/%s/%s.info", repo.ID, modulePath, version)

	content, err := p.cacheService.FetchMetadata(req.Context(), cacheKey, upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("bad gateway")
	}

	return content, nil
}

func (p *GoProtocol) localVersionInfo(req *http.Request, repo *domain.Repository, modulePath, version string) ([]byte, error) {
	pkg, err := p.packageStore.GetByNormalizedNameAndRepository(modulePath, repo.ID)
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

	var modFile *domain.PackageFile
	for _, f := range files {
		if f.Version == version && strings.HasSuffix(f.Filename, ".mod") {
			modFile = f
			break
		}
	}

	if modFile == nil {
		return nil, errors.New("not found")
	}

	info := VersionInfo{
		Version: version,
		Time:    modFile.CreatedAt,
	}

	return json.Marshal(info)
}

func (p *GoProtocol) virtualVersionInfo(req *http.Request, repo *domain.Repository, modulePath, version string) ([]byte, error) {
	for _, member := range repo.VirtualMembers {
		var content []byte
		var err error

		if member.MemberRepo.Type == domain.RepositoryTypeProxy {
			content, err = p.proxyVersionInfo(req, &member.MemberRepo, modulePath, version)
		} else {
			content, err = p.localVersionInfo(req, &member.MemberRepo, modulePath, version)
		}

		if err == nil {
			return content, nil
		}
	}

	return nil, errors.New("not found")
}
