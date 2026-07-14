package golang

import (
	"testing"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantModulePath string
		wantVersion    string
		wantExt        string
		wantErr        error
	}{
		{
			name:           "list action",
			path:           "github.com/foo/bar/@v/list",
			wantModulePath: "github.com/foo/bar",
			wantVersion:    "",
			wantExt:        "list",
			wantErr:        nil,
		},
		{
			name:           "info action",
			path:           "github.com/foo/bar/@v/v1.0.0.info",
			wantModulePath: "github.com/foo/bar",
			wantVersion:    "v1.0.0",
			wantExt:        ".info",
			wantErr:        nil,
		},
		{
			name:           "mod action",
			path:           "github.com/foo/bar/@v/v1.2.3-alpha.mod",
			wantModulePath: "github.com/foo/bar",
			wantVersion:    "v1.2.3-alpha",
			wantExt:        ".mod",
			wantErr:        nil,
		},
		{
			name:           "zip action",
			path:           "github.com/foo/bar/@v/v2.0.0.zip",
			wantModulePath: "github.com/foo/bar",
			wantVersion:    "v2.0.0",
			wantExt:        ".zip",
			wantErr:        nil,
		},
		{
			name:           "module with v2",
			path:           "github.com/foo/bar/v2/@v/v2.0.0.zip",
			wantModulePath: "github.com/foo/bar/v2",
			wantVersion:    "v2.0.0",
			wantExt:        ".zip",
			wantErr:        nil,
		},
		{
			name:           "invalid path without @v",
			path:           "github.com/foo/bar/v1.0.0",
			wantModulePath: "",
			wantVersion:    "",
			wantExt:        "",
			wantErr:        ErrInvalidPath,
		},
		{
			name:           "unsupported extension",
			path:           "github.com/foo/bar/@v/v1.0.0.tar.gz",
			wantModulePath: "github.com/foo/bar",
			wantVersion:    "",
			wantExt:        "",
			wantErr:        ErrUnsupportedAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modulePath, version, ext, err := ParsePath(tt.path)
			if err != tt.wantErr {
				t.Errorf("ParsePath() err = %v, want %v", err, tt.wantErr)
			}
			if modulePath != tt.wantModulePath {
				t.Errorf("ParsePath() modulePath = %v, want %v", modulePath, tt.wantModulePath)
			}
			if version != tt.wantVersion {
				t.Errorf("ParsePath() version = %v, want %v", version, tt.wantVersion)
			}
			if ext != tt.wantExt {
				t.Errorf("ParsePath() ext = %v, want %v", ext, tt.wantExt)
			}
		})
	}
}

func TestGoBlobKey(t *testing.T) {
	tests := []struct {
		name       string
		repoID     uint
		modulePath string
		filename   string
		want       string
	}{
		{
			name:       "basic info file",
			repoID:     1,
			modulePath: "github.com/foo/bar",
			filename:   "v1.0.0.info",
			want:       "1/github.com/foo/bar/v1.0.0.info",
		},
		{
			name:       "zip file with v2 module",
			repoID:     2,
			modulePath: "github.com/foo/bar/v2",
			filename:   "v2.1.0.zip",
			want:       "2/github.com/foo/bar/v2/v2.1.0.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := goBlobKey(tt.repoID, tt.modulePath, tt.filename); got != tt.want {
				t.Errorf("goBlobKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
