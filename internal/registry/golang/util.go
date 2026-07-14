package golang

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidPath       = errors.New("invalid goproxy path")
	ErrUnsupportedAction = errors.New("unsupported action")
)

// ParsePath parses a goproxy path (e.g. <module>/@v/<version>.info)
// and returns the module path, version (if any), and extension/action (.info, .mod, .zip, or list).
func ParsePath(path string) (modulePath, version, ext string, err error) {
	idx := strings.LastIndex(path, "/@v/")
	if idx == -1 {
		return "", "", "", ErrInvalidPath
	}

	modulePath = path[:idx]
	action := path[idx+4:]

	if action == "list" {
		return modulePath, "", "list", nil
	}

	if strings.HasSuffix(action, ".info") {
		return modulePath, strings.TrimSuffix(action, ".info"), ".info", nil
	}
	if strings.HasSuffix(action, ".mod") {
		return modulePath, strings.TrimSuffix(action, ".mod"), ".mod", nil
	}
	if strings.HasSuffix(action, ".zip") {
		return modulePath, strings.TrimSuffix(action, ".zip"), ".zip", nil
	}

	return modulePath, "", "", ErrUnsupportedAction
}

// goBlobKey builds a storage key scoped to the module so that different
// modules sharing a version (e.g. two modules both at v0.3.0) don't collide
// on the same "<version>.<ext>" filename.
func goBlobKey(repoID uint, modulePath, filename string) string {
	return fmt.Sprintf("%d/%s/%s", repoID, modulePath, filename)
}
