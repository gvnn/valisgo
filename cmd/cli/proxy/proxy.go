package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Server struct {
	UpstreamURL *url.URL
	BindAddr    string
}

func NewServer(upstream, bindAddr string) (*Server, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}
	return &Server{
		UpstreamURL: u,
		BindAddr:    bindAddr,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {

	revProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(s.UpstreamURL)

			pr.SetXForwarded()

			// INJECT FAKE TOKEN (For testing purposes)
			fakeToken := "fake-jwt-token-for-local-testing-12345"

			pr.Out.Header.Set("Authorization", "Bearer "+fakeToken)

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
