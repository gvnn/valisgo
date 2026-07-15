package server

import (
	"net/http"
	"net/url"

	"github.com/casbin/casbin/v3"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"valisgo/internal/auth"
)

type Server struct {
	enforcer     *casbin.Enforcer
	oidcVerifier *oidc.IDTokenVerifier
}

func NewServer(enforcer *casbin.Enforcer, oidcVerifier *oidc.IDTokenVerifier) *Server {
	return &Server{
		enforcer:     enforcer,
		oidcVerifier: oidcVerifier,
	}
}

func (s *Server) SetupRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	return r
}

func (s *Server) Protect(r chi.Router) {
	if s.oidcVerifier != nil {
		r.Use(auth.OIDCMiddleware(s.oidcVerifier))
	}

	if s.enforcer != nil {
		r.Use(CasbinAuthorization(s.enforcer))
	}
}

type redirectResponseWriter struct {
	http.ResponseWriter
	req        *http.Request
	loginURL   string
	redirected bool
}

func (rw *redirectResponseWriter) WriteHeader(statusCode int) {
	if rw.redirected {
		return
	}
	if statusCode == http.StatusUnauthorized {
		rw.redirected = true
		redirectURL := rw.loginURL
		if reqURI := rw.req.URL.RequestURI(); reqURI != "" {
			redirectURL += "?return_to=" + url.QueryEscape(reqURI)
		}
		http.Redirect(rw.ResponseWriter, rw.req, redirectURL, http.StatusFound)
		return
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *redirectResponseWriter) Write(b []byte) (int, error) {
	if rw.redirected {
		return len(b), nil
	}
	return rw.ResponseWriter.Write(b)
}

func (s *Server) ProtectWithRedirect(r chi.Router, loginURL string) {
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			rw := &redirectResponseWriter{
				ResponseWriter: w,
				req:            req,
				loginURL:       loginURL,
			}
			next.ServeHTTP(rw, req)
		})
	})
	s.Protect(r)
}
