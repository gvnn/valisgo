package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"valisgo/cmd/cli/proxy"

	"github.com/spf13/cobra"
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

		bindAddr := fmt.Sprintf("%s:%s", proxyBindAddr, proxyPort)

		srv, err := proxy.NewServer(proxyUpstream, bindAddr)
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
