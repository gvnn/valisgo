package server

import (
	"net/http"

	"github.com/casbin/casbin/v3"
)

func CasbinAuthorization(enforcer *casbin.Enforcer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Hardcode subject as "anon" for now.
			sub := "anon"

			obj := r.URL.Path
			act := r.Method

			allowed, err := enforcer.Enforce(sub, obj, act)
			if err != nil {
				http.Error(w, "internal server error during authorization", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "forbidden: you don't have access to this resource", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
