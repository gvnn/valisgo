package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"valisgo/cmd/cli/proxy"
	"valisgo/internal/auth"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var (
	proxyPort     string
	proxyUpstream string
	proxyBindAddr string
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start a local transparent proxy with credential injection",
	PreRun: func(cmd *cobra.Command, args []string) {
		if proxyUpstream == "" {
			proxyUpstream = address
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		cfg := auth.OIDCConfig{
			IssuerURL:         oidcIssuer,
			ClientID:          oidcClientID,
			Scopes:            []string{"openid", "profile", "email", "offline_access"},
			WorkloadTokenFile: wifFile,
			WorkloadTokenEnv:  wifEnv,
		}

		authenticator, err := auth.NewAuthenticator(ctx, cfg)
		if err != nil {
			return err
		}

		var storedRefreshToken string
		if wifFile == "" && wifEnv == "" {
			token, err := keyring.Get("valisgo", "refresh_token")
			if err != nil {
				return fmt.Errorf("no refresh token found in keyring. Please run 'valisgo-cli login' first")
			}
			storedRefreshToken = token
		}

		tokenSource, err := authenticator.GetTokenSource(ctx, storedRefreshToken)
		if err != nil {
			return err
		}

		bindAddr := fmt.Sprintf("%s:%s", proxyBindAddr, proxyPort)

		srv, err := proxy.NewServer(proxyUpstream, bindAddr, tokenSource)
		if err != nil {
			return err
		}

		return srv.Start(ctx)
	},
}

func init() {
	defaultProxyPort := os.Getenv("VALISGO_PROXY_PORT")
	if defaultProxyPort == "" {
		defaultProxyPort = "9000"
	}

	defaultProxyBindAddr := os.Getenv("VALISGO_PROXY_BIND_ADDRESS")
	if defaultProxyBindAddr == "" {
		defaultProxyBindAddr = "0.0.0.0"
	}

	proxyCmd.Flags().StringVar(&proxyPort, "port", defaultProxyPort, "Port to bind the local proxy (env: VALISGO_PROXY_PORT)")
	proxyCmd.Flags().StringVar(&proxyBindAddr, "bind-address", defaultProxyBindAddr, "Address to bind the local proxy (env: VALISGO_PROXY_BIND_ADDRESS)")
	proxyCmd.Flags().StringVar(&proxyUpstream, "upstream", "", "Valisgo registry to proxy to (defaults to global --address)")

	rootCmd.AddCommand(proxyCmd)
}
