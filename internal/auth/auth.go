package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	IssuerURL         string
	ClientID          string
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

// LoginBrowser spins up a local server, invokes the provided openBrowser func, and returns the Refresh Token
func (a *Authenticator) LoginBrowser(ctx context.Context, openBrowser func(url string)) (string, error) {
	oauthConf := &oauth2.Config{
		ClientID:    a.Config.ClientID,
		Endpoint:    a.Provider.Endpoint(),
		Scopes:      a.Config.Scopes,
		RedirectURL: a.Config.RedirectURL,
	}

	u, err := url.Parse(a.Config.RedirectURL)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URL: %w", err)
	}

	b := make([]byte, 16)
	rand.Read(b)
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

func (a *Authenticator) VerifyIDToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	verifier := a.Provider.Verifier(&oidc.Config{
		ClientID: a.Config.ClientID,
	})
	return verifier.Verify(ctx, rawToken)
}

func (a *Authenticator) VerifyAccessToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	verifier := a.Provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
	return verifier.Verify(ctx, rawToken)
}

func (a *Authenticator) GetTokenSource(ctx context.Context, storedRefreshToken string) (oauth2.TokenSource, error) {
	if a.Config.WorkloadTokenFile != "" || a.Config.WorkloadTokenEnv != "" {
		return nil, fmt.Errorf("workload identity federation is not yet implemented")
	}

	oauthConfig := oauth2.Config{
		ClientID: a.Config.ClientID,
		Endpoint: a.Provider.Endpoint(),
		Scopes:   a.Config.Scopes,
	}

	return oauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: storedRefreshToken,
	}), nil
}
