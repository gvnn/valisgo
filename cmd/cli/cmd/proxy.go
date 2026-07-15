package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	proxyPort     string
	proxyUpstream string
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start a local transparent proxy with credential injection",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		return nil
	},
}

func init() {
	defaultProxyPort := os.Getenv("VALISGO_PROXY_PORT")
	if defaultProxyPort == "" {
		defaultProxyPort = "9000"
	}

	proxyCmd.Flags().StringVar(&proxyPort, "port", defaultProxyPort, "Port to bind the local proxy (env: VALISGO_PROXY_PORT)")
	proxyCmd.Flags().StringVar(&proxyUpstream, "upstream", "", "Valisgo registry to proxy to (defaults to global --address)")

	rootCmd.AddCommand(proxyCmd)
}

func PreRun(c *cobra.Command) {
	// If the user didn't specify an upstream for the proxy, fall back to the global --address flag from root.go
	if proxyUpstream == "" {
		proxyUpstream = address
	}
}
