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
