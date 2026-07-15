package authapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"valisgo/internal/auth"

	"github.com/go-chi/chi/v5"
)

// isSecure reports whether the request arrived over HTTPS, accounting for a
// TLS-terminating proxy that forwards plaintext HTTP with X-Forwarded-Proto.
func isSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

type API struct {
	authenticator *auth.Authenticator
}

func NewAPI(authenticator *auth.Authenticator) *API {
	return &API{
		authenticator: authenticator,
	}
}

func (a *API) MountRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/login", a.handleLogin)
	r.Get("/callback", a.handleCallback)
	return r
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Set state cookie to prevent CSRF
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   300, // 5 minutes
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/browse"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_return_to",
		Value:    returnTo,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   300,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	authURL := a.authenticator.OAuth2Config().AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie missing", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code missing", http.StatusBadRequest)
		return
	}

	token, err := a.authenticator.OAuth2Config().Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// We prefer the id_token since it's meant for the client to verify identity,
	// but we fallback to access_token if needed.
	rawToken, ok := token.Extra("id_token").(string)
	if !ok {
		rawToken = token.AccessToken
	}

	maxAge := 3600 // 1 hour default
	if !token.Expiry.IsZero() {
		maxAge = int(time.Until(token.Expiry).Seconds())
	}
	if maxAge <= 0 {
		http.Error(w, "Received an already-expired token", http.StatusInternalServerError)
		return
	}

	// Set access_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   maxAge,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	returnTo := "/browse"
	if cookie, err := r.Cookie("oauth_return_to"); err == nil && cookie.Value != "" {
		returnTo = cookie.Value
	}

	// Clear return_to cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_return_to",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, returnTo, http.StatusFound)
}
