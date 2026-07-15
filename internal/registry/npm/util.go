package npm

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func getPackageName(req *http.Request) string {
	scope := chi.URLParam(req, "scope")
	pkg := chi.URLParam(req, "package")
	if scope != "" {
		return fmt.Sprintf("@%s/%s", scope, pkg)
	}
	return pkg
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

// mergeMetadata merges additional NPM metadata into the base metadata.
// It prioritizes existing keys in base (so additional does not overwrite base).
func mergeMetadata(base, additional map[string]interface{}) map[string]interface{} {
	if base == nil {
		return additional
	}
	if additional == nil {
		return base
	}

	// Merge versions
	if addVersions, ok := additional["versions"].(map[string]interface{}); ok {
		if baseVersions, ok := base["versions"].(map[string]interface{}); ok {
			for k, v := range addVersions {
				if _, exists := baseVersions[k]; !exists {
					baseVersions[k] = v
				}
			}
		} else {
			base["versions"] = addVersions
		}
	}

	// Merge dist-tags
	if addTags, ok := additional["dist-tags"].(map[string]interface{}); ok {
		if baseTags, ok := base["dist-tags"].(map[string]interface{}); ok {
			for k, v := range addTags {
				if _, exists := baseTags[k]; !exists {
					baseTags[k] = v
				}
			}
		} else {
			base["dist-tags"] = addTags
		}
	}

	return base
}

// getScheme determines the scheme (http or https) from the request.
func getScheme(req *http.Request) string {
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

// getHost determines the host (taking into account X-Forwarded-Host) from the request.
func getHost(req *http.Request) string {
	if xfh := req.Header.Get("X-Forwarded-Host"); xfh != "" {
		return xfh
	}
	return req.Host
}


// formatIntegrity converts a hex-encoded SHA256 string to an NPM integrity string.
func formatIntegrity(hashHex string) string {
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return ""
	}
	return "sha256-" + base64.StdEncoding.EncodeToString(hashBytes)
}
