package npm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		pkgName  string
		expected string
	}{
		{
			name:     "standard package",
			filename: "lodash-4.17.21.tgz",
			pkgName:  "lodash",
			expected: "4.17.21",
		},
		{
			name:     "scoped package",
			filename: "core-1.2.3.tgz",
			pkgName:  "@babel/core",
			expected: "1.2.3",
		},
		{
			name:     "pre-release version",
			filename: "react-18.0.0-rc.3.tgz",
			pkgName:  "react",
			expected: "18.0.0-rc.3",
		},
		{
			name:     "invalid extension",
			filename: "lodash-4.17.21.zip",
			pkgName:  "lodash",
			expected: "unknown",
		},
		{
			name:     "invalid prefix",
			filename: "otherpkg-1.0.0.tgz",
			pkgName:  "mypkg",
			expected: "unknown",
		},
		{
			name:     "no version",
			filename: "lodash-.tgz",
			pkgName:  "lodash",
			expected: "", // TrimPrefix will return empty string here, which is fine
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.filename, tt.pkgName)
			if result != tt.expected {
				t.Errorf("extractVersion(%q, %q) = %q; want %q", tt.filename, tt.pkgName, result, tt.expected)
			}
		})
	}
}
func TestGetPackageName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard package",
			url:      "/lodash",
			expected: "lodash",
		},
		{
			name:     "scoped package",
			url:      "/@babel/core",
			expected: "@babel/core",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			r := chi.NewRouter()
			var result string
			
			r.Get("/{package}", func(w http.ResponseWriter, r *http.Request) {
				result = getPackageName(r)
			})
			r.Get("/@{scope}/{package}", func(w http.ResponseWriter, r *http.Request) {
				result = getPackageName(r)
			})
			
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			
			if result != tt.expected {
				t.Errorf("getPackageName() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestMergeMetadata(t *testing.T) {
	base := map[string]interface{}{
		"versions": map[string]interface{}{
			"1.0.0": map[string]interface{}{"name": "pkg", "version": "1.0.0"},
			"1.1.0": map[string]interface{}{"name": "pkg", "version": "1.1.0"},
		},
		"dist-tags": map[string]interface{}{
			"latest": "1.1.0",
			"stable": "1.0.0",
		},
	}

	additional := map[string]interface{}{
		"versions": map[string]interface{}{
			"1.1.0": map[string]interface{}{"name": "pkg", "version": "1.1.0-override"}, // should not overwrite
			"1.2.0": map[string]interface{}{"name": "pkg", "version": "1.2.0"},          // should be added
		},
		"dist-tags": map[string]interface{}{
			"latest": "1.2.0", // should not overwrite
			"next":   "2.0.0", // should be added
		},
	}

	merged := mergeMetadata(base, additional)

	versions := merged["versions"].(map[string]interface{})
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	v110 := versions["1.1.0"].(map[string]interface{})
	if v110["version"] != "1.1.0" {
		t.Errorf("Expected version 1.1.0 to remain unchanged, got %s", v110["version"])
	}

	if _, ok := versions["1.2.0"]; !ok {
		t.Error("Expected version 1.2.0 to be added")
	}

	tags := merged["dist-tags"].(map[string]interface{})
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	if tags["latest"] != "1.1.0" {
		t.Errorf("Expected latest tag to remain unchanged, got %s", tags["latest"])
	}

	if _, ok := tags["next"]; !ok {
		t.Error("Expected next tag to be added")
	}
}
