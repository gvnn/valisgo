package browse

import (
	"testing"
)

func TestParseGoPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantPkg      string
		wantFileName string
	}{
		{"Package only", "github.com/gin-gonic/gin", "github.com/gin-gonic/gin", ""},
		{"Mod file", "github.com/gin-gonic/gin/@v/v1.7.0.mod", "github.com/gin-gonic/gin/@v", "v1.7.0.mod"},
		{"Zip file", "github.com/gin-gonic/gin/v1.7.0.zip", "github.com/gin-gonic/gin", "v1.7.0.zip"},
		{"Info file", "github.com/gin-gonic/gin/v1.7.0.info", "github.com/gin-gonic/gin", "v1.7.0.info"},
		{"No slash file", "v1.7.0.mod", "v1.7.0.mod", ""},
		{"Not a file extension", "github.com/gin-gonic/gin/v1.7.0", "github.com/gin-gonic/gin/v1.7.0", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotFileName := parseGoPath(tt.path)
			if gotPkg != tt.wantPkg || gotFileName != tt.wantFileName {
				t.Errorf("parseGoPath(%q) = (%q, %q), want (%q, %q)", tt.path, gotPkg, gotFileName, tt.wantPkg, tt.wantFileName)
			}
		})
	}
}

func TestParseNPMPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantPkg      string
		wantFileName string
	}{
		{"Unscoped package only", "react", "react", ""},
		{"Unscoped package with file", "react/-/react-17.0.2.tgz", "react", "-/react-17.0.2.tgz"},
		{"Scoped package only", "@babel/core", "@babel/core", ""},
		{"Scoped package with file", "@babel/core/-/core-7.14.0.tgz", "@babel/core", "-/core-7.14.0.tgz"},
		{"Deep path in unscoped", "react/dist/index.js", "react", "dist/index.js"},
		{"Deep path in scoped", "@myorg/mypkg/dist/index.js", "@myorg/mypkg", "dist/index.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotFileName := parseNPMPath(tt.path)
			if gotPkg != tt.wantPkg || gotFileName != tt.wantFileName {
				t.Errorf("parseNPMPath(%q) = (%q, %q), want (%q, %q)", tt.path, gotPkg, gotFileName, tt.wantPkg, tt.wantFileName)
			}
		})
	}
}

func TestParseDefaultPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantPkg      string
		wantFileName string
	}{
		{"Package only", "requests", "requests", ""},
		{"Package with file", "requests/requests-2.25.1.tar.gz", "requests", "requests-2.25.1.tar.gz"},
		{"Package with deep file path", "root/a/b/c.txt", "root", "a/b/c.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkg, gotFileName := parseDefaultPath(tt.path)
			if gotPkg != tt.wantPkg || gotFileName != tt.wantFileName {
				t.Errorf("parseDefaultPath(%q) = (%q, %q), want (%q, %q)", tt.path, gotPkg, gotFileName, tt.wantPkg, tt.wantFileName)
			}
		})
	}
}
