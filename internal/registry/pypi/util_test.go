package pypi

import (
	"net/http"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "foo-bar",
			expected: "foo-bar",
		},
		{
			name:     "with underscores",
			input:    "foo_bar",
			expected: "foo-bar",
		},
		{
			name:     "with dots",
			input:    "foo.bar",
			expected: "foo-bar",
		},
		{
			name:     "mixed and uppercase",
			input:    "Foo_.-Bar",
			expected: "foo-bar",
		},
		{
			name:     "trailing and leading",
			input:    "_foo.bar-",
			expected: "-foo-bar-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAcceptsJSON(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected bool
	}{
		{
			name:     "v1 json",
			accept:   "application/vnd.pypi.simple.v1+json",
			expected: true,
		},
		{
			name:     "latest json",
			accept:   "application/vnd.pypi.simple.latest+json",
			expected: true,
		},
		{
			name:     "multiple including valid json",
			accept:   "text/html, application/vnd.pypi.simple.v1+json",
			expected: true,
		},
		{
			name:     "html only",
			accept:   "text/html",
			expected: false,
		},
		{
			name:     "empty",
			accept:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			
			result := acceptsJSON(req)
			if result != tt.expected {
				t.Errorf("acceptsJSON() with Accept: %q = %v, want %v", tt.accept, result, tt.expected)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		pkgName  string
		expected string
	}{
		{
			name:     "wheel format",
			filename: "requests-2.28.1-py3-none-any.whl",
			pkgName:  "requests",
			expected: "2.28.1",
		},
		{
			name:     "tar.gz format",
			filename: "requests-2.28.1.tar.gz",
			pkgName:  "requests",
			expected: "2.28.1",
		},
		{
			name:     "zip format",
			filename: "Flask-2.2.2.zip",
			pkgName:  "Flask",
			expected: "2.2.2",
		},
		{
			name:     "pkg name with dashes translated to underscores",
			filename: "foo_bar-1.0.0.tar.gz",
			pkgName:  "foo-bar",
			expected: "1.0.0",
		},
		{
			name:     "pkg name with exact match prefix",
			filename: "foo-bar-1.0.0.tar.gz",
			pkgName:  "foo-bar",
			expected: "1.0.0",
		},
		{
			name:     "wheel format complex name",
			filename: "foo_bar-1.2.3-py3-none-any.whl",
			pkgName:  "foo-bar",
			expected: "1.2.3",
		},
		{
			name:     "unknown extension",
			filename: "requests-2.28.1.exe",
			pkgName:  "requests",
			expected: "unknown",
		},
		{
			name:     "mismatched package name",
			filename: "otherpkg-1.0.tar.gz",
			pkgName:  "requests",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.filename, tt.pkgName)
			if result != tt.expected {
				t.Errorf("extractVersion(%q, %q) = %q, want %q", tt.filename, tt.pkgName, result, tt.expected)
			}
		})
	}
}
