package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

type Server struct {
	UpstreamURL *url.URL
	BindAddr    string
	TokenSource oauth2.TokenSource
}

func NewServer(upstream, bindAddr string, ts oauth2.TokenSource) (*Server, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}
	return &Server{
		UpstreamURL: u,
		BindAddr:    bindAddr,
		TokenSource: ts,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {

	revProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(s.UpstreamURL)
			pr.SetXForwarded()

			token, err := s.TokenSource.Token()
			if err != nil {
				slog.Error("Failed to get valid access token", "error", err)
			} else {
				pr.Out.Header.Set("Authorization", "Bearer "+token.AccessToken)
			}
		},
	}

	srv := &http.Server{
		Addr:    s.BindAddr,
		Handler: revProxy,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("Starting local proxy", "addr", s.BindAddr)
		slog.Info("Forwarding traffic", "upstream", s.UpstreamURL.String())
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server crashed: %w", err)

	case <-ctx.Done(): // Triggered by Ctrl+C (SIGINT)
		slog.Info("Shutdown signal received, initiating graceful shutdown...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		slog.Info("Proxy stopped successfully.")
	}

	return nil
}
