package pypi

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// NormalizeName implements PEP 503 normalization.
func NormalizeName(name string) string {
	re := regexp.MustCompile(`[-_.]+`)
	return strings.ToLower(re.ReplaceAllString(name, "-"))
}

func acceptsJSON(req *http.Request) bool {
	accept := req.Header.Get("Accept")
	return strings.Contains(accept, "application/vnd.pypi.simple.v1+json") ||
		strings.Contains(accept, "application/vnd.pypi.simple.latest+json")
}

// extractVersion attempts to parse the package version from a PyPI filename.
func extractVersion(filename, pkgName string) string {
	base := filename
	knownExt := false
	for _, ext := range []string{".whl", ".tar.gz", ".zip", ".tar.bz2", ".egg"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			knownExt = true
			break
		}
	}

	if !knownExt {
		return "unknown"
	}

	if strings.HasSuffix(filename, ".whl") {
		parts := strings.Split(base, "-")
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	lowerBase := strings.ToLower(base)
	lowerPkg := strings.ToLower(pkgName)
	
	if strings.HasPrefix(lowerBase, lowerPkg+"-") {
		return base[len(lowerPkg)+1:]
	}
	
	lowerPkgUnderscore := strings.ReplaceAll(lowerPkg, "-", "_")
	if strings.HasPrefix(lowerBase, lowerPkgUnderscore+"-") {
		return base[len(lowerPkgUnderscore)+1:]
	}

	return "unknown"
}

// deduplicateFiles removes duplicate template files based on their Filename.
// The first occurrence of a filename is kept.
func deduplicateFiles(files []templateFile) []templateFile {
	var result []templateFile
	seen := make(map[string]bool)

	for _, tf := range files {
		if !seen[tf.Filename] {
			seen[tf.Filename] = true
			result = append(result, tf)
		}
	}

	return result
}

func parseUploadForm(req *http.Request) (*uploadMetadata, error) {
	err := req.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		return nil, fmt.Errorf("invalid form: %w", err)
	}

	if req.FormValue(":action") != "file_upload" {
		return nil, errors.New("unsupported action")
	}

	name := req.FormValue("name")
	version := req.FormValue("version")
	if name == "" || version == "" {
		return nil, errors.New("missing name or version")
	}

	file, header, err := req.FormFile("content")
	if err != nil {
		return nil, fmt.Errorf("missing file content: %w", err)
	}

	return &uploadMetadata{
		Name:           name,
		NormalizedName: NormalizeName(name),
		Version:        version,
		Filename:       header.Filename,
		Size:           header.Size,
		File:           file,
	}, nil
}
