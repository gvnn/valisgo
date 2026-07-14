//go:build ignore

package main

import (
	"fmt"
	"os"

	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run pack.go <module@version> <dir> <out.zip>")
		os.Exit(1)
	}

	modPath := os.Args[1] // e.g. example.com/my-dummy-pkg@v1.0.0

	// Split "path@version" into a module.Version.
	var m module.Version
	for i, c := range modPath {
		if c == '@' {
			m.Path = modPath[:i]
			m.Version = modPath[i+1:]
			break
		}
	}

	f, err := os.Create(os.Args[3])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := zip.CreateFromDir(f, m, os.Args[2]); err != nil {
		panic(err)
	}
}
