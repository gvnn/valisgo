package golang

import (
	"io"
	"net/http"
	"time"

	"valisgo/internal/domain"
	"valisgo/internal/proxy"
	"valisgo/internal/storage"

	"golang.org/x/sync/singleflight"
)

type GoProtocol struct {
	packageStore     domain.PackageStore
	packageFileStore domain.PackageFileStore
	storage          storage.Storage
	cacheService     *proxy.CacheService
	downloadSF       singleflight.Group
}

type trackedWriter struct {
	http.ResponseWriter
	written bool
}

func (tw *trackedWriter) Write(b []byte) (int, error) {
	tw.written = true
	return tw.ResponseWriter.Write(b)
}

func (tw *trackedWriter) WriteHeader(statusCode int) {
	tw.written = true
	tw.ResponseWriter.WriteHeader(statusCode)
}

type VersionInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

type lengthReader struct {
	r    io.Reader
	size int64
}

func (lr *lengthReader) Read(p []byte) (int, error) {
	n, err := lr.r.Read(p)
	lr.size += int64(n)
	return n, err
}
