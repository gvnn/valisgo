package registry

import (
	"context"
	"net/http"

	"valisgo/internal/domain"
	"github.com/go-chi/chi/v5"
)

type FallbackResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (fw *FallbackResponseWriter) WriteHeader(statusCode int) {
	fw.Status = statusCode
	if statusCode != http.StatusNotFound {
		fw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (fw *FallbackResponseWriter) Write(p []byte) (int, error) {
	if fw.Status != http.StatusNotFound {
		return fw.ResponseWriter.Write(p)
	}
	return len(p), nil
}

func DispatchVirtualDownload(w http.ResponseWriter, req *http.Request, repo *domain.Repository, protoRouter chi.Router) {
	for _, member := range repo.VirtualMembers {
		ctx := context.WithValue(req.Context(), domain.RepoCtxKey, &member.MemberRepo)
		reqWithCtx := req.WithContext(ctx)

		fw := &FallbackResponseWriter{ResponseWriter: w}
		protoRouter.ServeHTTP(fw, reqWithCtx)

		if fw.Status != http.StatusNotFound {
			return
		}
	}

	http.Error(w, "not found in any virtual member", http.StatusNotFound)
}
