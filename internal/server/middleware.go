package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"valisgo/internal/auth"

	"github.com/casbin/casbin/v3"
)

func CasbinAuthorization(enforcer *casbin.Enforcer, authenticator *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Debug("Incoming request headers", "headers", r.Header)

			sub := "anon"

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {

				tokenStr := authHeader[7:]
				
				if authenticator != nil {
					idToken, err := authenticator.VerifyAccessToken(r.Context(), tokenStr)
					if err != nil {
						slog.Warn("Invalid or expired token", "error", err)
						http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
						return
					}

					var claims struct {
						Subject string `json:"sub"`
						Email   string `json:"email"`
					}
					if err := idToken.Claims(&claims); err != nil {
						slog.Error("Failed to parse token claims", "error", err)
						http.Error(w, "internal server error", http.StatusInternalServerError)
						return
					}

					if claims.Email != "" {
						sub = claims.Email
					} else if claims.Subject != "" {
						sub = claims.Subject
					}
				} else {
					slog.Warn("Authenticator is not configured, treating token as invalid")
					http.Error(w, "unauthorized: authenticator not configured", http.StatusUnauthorized)
					return
				}
			}

			obj := r.URL.Path
			act := r.Method

			slog.Debug("Evaluating Casbin policy", "sub", sub, "obj", obj, "act", act)

			allowed, err := enforcer.Enforce(sub, obj, act)
			if err != nil {
				slog.Error("Casbin enforcement error", "error", err)
				http.Error(w, "internal server error during authorization", http.StatusInternalServerError)
				return
			}

			if !allowed {
				slog.Warn("Forbidden access attempt", "sub", sub, "obj", obj, "act", act)
				http.Error(w, "forbidden: you don't have access to this resource", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
