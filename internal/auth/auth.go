package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	Scopes            []string
	RedirectURL       string
	WorkloadTokenFile string
	WorkloadTokenEnv  string
}

type Authenticator struct {
	Provider *oidc.Provider
	Config   OIDCConfig
}

func NewAuthenticator(ctx context.Context, cfg OIDCConfig) (*Authenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{oidc.ScopeOpenID, "profile", "email", "offline_access"}
	}

	if cfg.RedirectURL == "" {
		cfg.RedirectURL = "http://0.0.0.0:8585/callback"
	}

	return &Authenticator{
		Provider: provider,
		Config:   cfg,
	}, nil
}

// OAuth2Config builds the oauth2.Config for this authenticator.
func (a *Authenticator) OAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.Config.ClientID,
		ClientSecret: a.Config.ClientSecret,
		Endpoint:     a.Provider.Endpoint(),
		Scopes:       a.Config.Scopes,
		RedirectURL:  a.Config.RedirectURL,
	}
}

// Verifier returns an OIDC ID token verifier backed by this authenticator's provider.
func (a *Authenticator) Verifier() *oidc.IDTokenVerifier {
	return a.Provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
}

// LoginBrowser spins up a local server, invokes the provided openBrowser func, and returns the Refresh Token
func (a *Authenticator) LoginBrowser(ctx context.Context, openBrowser func(url string)) (string, error) {
	oauthConf := a.OAuth2Config()

	u, err := url.Parse(a.Config.RedirectURL)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URL: %w", err)
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	authURL := oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	codeChan := make(chan string)
	errChan := make(chan error)

	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			errChan <- fmt.Errorf("state mismatch")
			return
		}
		if err := r.URL.Query().Get("error"); err != "" {
			http.Error(w, "OAuth Error", http.StatusBadRequest)
			errChan <- fmt.Errorf("oauth error: %s", err)
			return
		}

		code := r.URL.Query().Get("code")
		fmt.Fprintln(w, "Login successful! You can close this window and return to your terminal.")
		codeChan <- code
	})
	server := &http.Server{Addr: u.Host, Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	defer server.Shutdown(context.Background())

	// Call the injected browser opening function
	slog.Info("Opening browser for authentication...")
	openBrowser(authURL)

	select {
	case err := <-errChan:
		return "", err
	case code := <-codeChan:
		token, err := oauthConf.Exchange(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to exchange code: %w", err)
		}
		if token.RefreshToken == "" {
			return "", fmt.Errorf("OIDC provider did not return a refresh token")
		}
		return token.RefreshToken, nil
	}
}

// GetTokenSource creates an oauth2.TokenSource from a refresh token
func (a *Authenticator) GetTokenSource(ctx context.Context, refreshToken string) oauth2.TokenSource {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	return a.OAuth2Config().TokenSource(ctx, token)
}

// OIDCMiddleware creates a middleware that verifies Bearer tokens or cookies
func OIDCMiddleware(verifier *oidc.IDTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var rawToken string

			authHeader := r.Header.Get("Authorization")
			if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
				rawToken = authHeader[7:]
			} else {
				cookie, err := r.Cookie("access_token")
				if err == nil {
					rawToken = cookie.Value
				}
			}

			if rawToken == "" {
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
			}

			_, err := verifier.Verify(r.Context(), rawToken)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
