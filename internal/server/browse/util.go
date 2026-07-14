package browse

import (
	"strings"
)

func parseGoPath(path string) (string, string) {
	isGoFile := strings.HasSuffix(path, ".mod") || strings.HasSuffix(path, ".zip") || strings.HasSuffix(path, ".info")
	if !isGoFile {
		return path, ""
	}

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return path, ""
	}

	return path[:lastSlash], path[lastSlash+1:]
}

func parseNPMPath(path string) (string, string) {
	parts := strings.Split(path, "/")
	
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "@") {
		pkgName := parts[0] + "/" + parts[1]
		if len(parts) == 2 {
			return pkgName, ""
		}
		return pkgName, strings.Join(parts[2:], "/")
	}

	pkgName := parts[0]
	if len(parts) == 1 {
		return pkgName, ""
	}
	return pkgName, strings.Join(parts[1:], "/")
}

func parseDefaultPath(path string) (string, string) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
