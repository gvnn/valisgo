package pypi

import (
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
	for _, ext := range []string{".whl", ".tar.gz", ".zip", ".tar.bz2", ".egg"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
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
